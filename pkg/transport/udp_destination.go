package transport

import (
	"context"
	goerrs "errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/transport/SpanreedMessage"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"
)

// TODO (sessamekesh): Allow configuring UDP ranges for client connections

type udpSpanreedDestinationChannels struct {
	Address *net.UDPAddr
}

type outgoingMessage struct {
	addr    *net.UDPAddr
	payload []byte
}

type udpSpanreedDestination struct {
	params UdpSpanreedDestinationParams

	log       *zap.Logger
	stringGen *utils.RandomStringGenerator

	resolveUdpHost  func(connStr string) (string, error)
	proxyConnection *handlers.DestinationMessageHandler

	mut_destinationConnections sync.RWMutex
	destinationConnections     map[uint32]*udpSpanreedDestinationChannels

	outgoingMessages chan outgoingMessage
}

type UdpSpanreedDestinationParams struct {
	UdpServerPort int

	AllowAllDestinations    bool
	AllowlistedDestinations []string
	DenylistedDestinations  []string

	MagicNumber uint32
	Version     uint8

	MaxReadMessageSize  int64
	MaxWriteMessageSize int64

	Logger *zap.Logger

	ResolveUdpHost func(connStr string) (string, error)
}

func safeParseDestinationMessage(payload []byte) (msg *SpanreedMessage.DestinationMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			msg = nil
			err = fmt.Errorf("deformed message: %v", err)
		}
	}()

	return SpanreedMessage.GetRootAsDestinationMessage(payload, 0), nil
}

func isDestinationAllowed(allowAll bool, allowlist, denylist []string, destination string) bool {
	if utils.Contains(destination, denylist) {
		return false
	}

	if allowAll {
		return true
	}

	return utils.Contains(destination, allowlist)
}

func DefaultUdpDestinationHandlerMatchConnectionStringFn(connStr string) bool {
	return connStr[0:4] == "udp:"
}

func defaultHostResolution(connStr string) (string, error) {
	if connStr[0:4] != "udp:" {
		return "", goerrs.New("not a UDP connection string")
	}

	return connStr[4:], nil
}

func CreateUdpDestinationHandler(proxyConnection *handlers.DestinationMessageHandler, params UdpSpanreedDestinationParams) (*udpSpanreedDestination, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}
	resolveUdpAddrFunc := params.ResolveUdpHost
	if resolveUdpAddrFunc == nil {
		resolveUdpAddrFunc = defaultHostResolution
	}

	return &udpSpanreedDestination{
		params:          params,
		log:             logger.With(zap.String("handler", "udpDestination")),
		stringGen:       utils.CreateRandomstringGenerator(time.Now().UnixMicro()),
		resolveUdpHost:  resolveUdpAddrFunc,
		proxyConnection: proxyConnection,

		mut_destinationConnections: sync.RWMutex{},
		destinationConnections:     make(map[uint32]*udpSpanreedDestinationChannels),

		outgoingMessages: make(chan outgoingMessage, 128), // TODO (sessamekesh): Config for length
	}, nil
}

func (s *udpSpanreedDestination) Start(ctx context.Context) error {
	wg := sync.WaitGroup{}

	port := s.params.UdpServerPort
	if port == 0 {
		port = 30321 // why not
	}

	hostAddr, hostAddrErr := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if hostAddrErr != nil {
		return hostAddrErr
	}

	conn, listenErr := net.ListenUDP("udp", hostAddr)
	if listenErr != nil {
		return listenErr
	}

	// TODO (sessamekesh): Configure connection correctly!
	conn.SetReadBuffer(4096)
	conn.SetWriteBuffer(4096)

	//
	// Connection closing goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		s.log.Info("UDP destination closing, sending close request to all connected clients...")
		defer s.log.Info("Successfully sent close requests to all connected clients.")
		s.mut_destinationConnections.Lock()
		defer s.mut_destinationConnections.Unlock()
		for clientId, connData := range s.destinationConnections {
			b := flatbuffers.NewBuilder(64)
			SpanreedMessage.ProxyDestCloseConnectionStart(b)
			SpanreedMessage.ProxyDestCloseConnectionAddClientId(b, clientId)
			inner_msg := SpanreedMessage.ProxyDestCloseConnectionEnd(b)

			SpanreedMessage.ProxyDestinationMessageStart(b)
			SpanreedMessage.ProxyDestinationMessageAddInnerMessageType(b, SpanreedMessage.ProxyDestInnerMsgProxyDestCloseConnection)
			SpanreedMessage.ProxyDestinationMessageAddInnerMessage(b, inner_msg)
			cmsg := SpanreedMessage.ProxyDestinationMessageEnd(b)
			b.Finish(cmsg)
			buf := b.FinishedBytes()

			conn.WriteToUDP(buf, connData.Address)
			delete(s.destinationConnections, clientId)
		}

		conn.Close()
	}()

	//
	// Connection message listening goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			var buf [2048]byte
			bytesRead, clientAddr, err := conn.ReadFromUDP(buf[0:])
			if err != nil {
				if goerrs.Is(err, net.ErrClosed) {
					s.log.Info("UDP server connection close requested - exiting connection message listening goroutine")
					return
				} else if strings.Contains(err.Error(), "WSAEMSGSIZE") {
					//  WINDOWS ONLY - Unix silently discards excess data.
					s.log.Warn("Overflow of data buffer - continuing, but may contain truncated data!", zap.Int("bytesRead", bytesRead))
				} else {
					s.log.Error("Error reading UDP datagram from connection, closing!", zap.Error(err))
					return
				}
			}

			rawMsg := buf[0:bytesRead]
			parsedMsg, msgParseError := safeParseDestinationMessage(rawMsg)
			if msgParseError != nil {
				s.log.Warn("Failed to parse incoming datagram", zap.Error(msgParseError))
				continue
			}

			// TODO (sessamekesh): Handle parsed message correctly here!
			// TODO (sessamekesh): That involves making sure that the destination server is at the right address!
			ut := new(flatbuffers.Table)
			if parsedMsg.Msg(ut) {
				switch parsedMsg.MsgType() {
				case SpanreedMessage.InnerMsgConnectionVerdict:
					cv := new(SpanreedMessage.ConnectionVerdict)
					cv.Init(ut.Bytes, ut.Pos)
					s.onReceiveVerdict(cv, parsedMsg.AppDataBytes())
					// TODO (sessamekesh): Make this safe!
				case SpanreedMessage.InnerMsgProxyMessage:
					pm := new(SpanreedMessage.ProxyMessage)
					pm.Init(ut.Bytes, ut.Pos)
					s.onReceiveProxyMessage(pm, parsedMsg.AppDataBytes())
					// TODO (sessamekesh): Make this safe!
				case SpanreedMessage.InnerMsgCloseConnection:
					cr := new(SpanreedMessage.CloseConnection)
					cr.Init(ut.Bytes, ut.Pos)
					s.onReceiveCloseRequest(cr, parsedMsg.AppDataBytes())
					// TODO (sessamekesh): Make this safe!
				case SpanreedMessage.InnerMsgNONE:
				default:
					s.log.Warn("Unexpected message type from proxy, skipping", zap.String("clientAddr", clientAddr.String()))
				}
			}
		}
	}()

	//
	// Proxy message listener goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.log.Info("Starting UDP proxy message goroutine loop")

		for {
			select {
			case <-ctx.Done():
				return
			case openConnectionRequest := <-s.proxyConnection.OpenClientChannel:
				s.log.Info("Connection open request", zap.Uint32("clientId", openConnectionRequest.ClientId), zap.String("connectionString", openConnectionRequest.ConnectionString))
				err := s.onConnectClient(openConnectionRequest)
				if err != nil {
					s.log.Warn("Could not open destination connection for client", zap.Uint32("clientId", openConnectionRequest.ClientId), zap.Error(err))
					s.proxyConnection.OpenClientVerdictChannel <- handlers.OpenClientConnectionVerdict{
						ClientId: openConnectionRequest.ClientId,
						Verdict:  false,
						AppData:  nil,
						Error:    err,
					}
				}
			case closeRequest := <-s.proxyConnection.ProxyCloseRequests:
				s.log.Info("Handle close request", zap.Uint32("clientId", closeRequest.ClientId))
				s.onProxyRequestClose(closeRequest)
			case msgRequest := <-s.proxyConnection.OutgoingMessageChannel:
				s.onClientMessage(msgRequest)
			}
		}
	}()

	//
	// Message sending goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.log.Info("Starting UDP proxy destination message dispatch loop")
		for {
			select {
			case <-ctx.Done():
				return
			case outgoingMessage := <-s.outgoingMessages:
				conn.WriteToUDP(outgoingMessage.payload, outgoingMessage.addr)
			}
		}
	}()

	wg.Wait()

	return nil
}

func (s *udpSpanreedDestination) onConnectClient(outgoingMsg handlers.OpenClientConnectionCommand) error {
	log := s.log.With(zap.Uint32("clientId", outgoingMsg.ClientId))
	udpHost, udpHostErr := s.resolveUdpHost(outgoingMsg.ConnectionString)
	if udpHostErr != nil {
		log.Error("Cannot resolve UDP host from connection string", zap.String("udpHost", outgoingMsg.ConnectionString), zap.Error(udpHostErr))
		return udpHostErr
	}

	if !isDestinationAllowed(s.params.AllowAllDestinations, s.params.AllowlistedDestinations, s.params.DenylistedDestinations, udpHost) {
		log.Error("Attempted connection to disallowed destination, not allowing!")
		return goerrs.New("cannot connect to denied destination")
	}

	udpAddr, udpAddrErr := net.ResolveUDPAddr("udp", udpHost)
	if udpAddrErr != nil {
		log.Error("Cannot resolve UDP address from connection string", zap.String("udpHost", udpHost), zap.Error(udpAddrErr))
		return udpAddrErr
	}

	log.Info("Attempting connection to UDP addr", zap.String("udpAddr", udpAddr.String()))

	destinationChannels := &udpSpanreedDestinationChannels{
		Address: udpAddr,
	}
	func() {
		s.mut_destinationConnections.Lock()
		defer s.mut_destinationConnections.Unlock()

		s.destinationConnections[outgoingMsg.ClientId] = destinationChannels
	}()

	b := flatbuffers.NewBuilder(64 + len(outgoingMsg.AppData))
	// Inner message
	SpanreedMessage.ProxyDestConnectionRequestStart(b)
	SpanreedMessage.ProxyDestConnectionRequestAddClientId(b, outgoingMsg.ClientId)
	inner_msg := SpanreedMessage.ProxyDestConnectionRequestEnd(b)

	var appDataLoc flatbuffers.UOffsetT
	if outgoingMsg.AppData != nil {
		appDataLoc = b.CreateByteVector(outgoingMsg.AppData)
	}

	SpanreedMessage.ProxyDestinationMessageStart(b)
	SpanreedMessage.ProxyDestinationMessageAddInnerMessageType(b, SpanreedMessage.ProxyDestInnerMsgProxyDestConnectionRequest)
	SpanreedMessage.ProxyDestinationMessageAddInnerMessage(b, inner_msg)
	if outgoingMsg.AppData != nil {
		SpanreedMessage.ProxyDestinationMessageAddAppData(b, appDataLoc)
	}
	cmsg := SpanreedMessage.ProxyDestinationMessageEnd(b)
	b.Finish(cmsg)
	buf := b.FinishedBytes()
	s.outgoingMessages <- outgoingMessage{
		addr:    udpAddr,
		payload: buf,
	}

	return nil
}

func (s *udpSpanreedDestination) onClientMessage(msg handlers.DestinationMessage) error {
	addr := func() *net.UDPAddr {
		s.mut_destinationConnections.RLock()
		defer s.mut_destinationConnections.RUnlock()

		addr, has := s.destinationConnections[msg.ClientId]
		if !has {
			return nil
		}
		return addr.Address
	}()
	if addr == nil {
		s.log.Warn("Cannot forward message, no client address found", zap.Uint32("clientId", msg.ClientId))
		// TODO (sessamekesh): Error state here instead
		return nil
	}

	b := flatbuffers.NewBuilder(64 + len(msg.Data))
	SpanreedMessage.ProxyDestClientMessageStart(b)
	SpanreedMessage.ProxyDestClientMessageAddClientId(b, msg.ClientId)
	inner_msg := SpanreedMessage.ProxyDestClientMessageEnd(b)

	var appDataLoc flatbuffers.UOffsetT
	if msg.Data != nil {
		appDataLoc = b.CreateByteVector(msg.Data)
	}

	SpanreedMessage.ProxyDestinationMessageStart(b)
	SpanreedMessage.ProxyDestinationMessageAddInnerMessageType(b, SpanreedMessage.ProxyDestInnerMsgProxyDestClientMessage)
	SpanreedMessage.ProxyDestinationMessageAddInnerMessage(b, inner_msg)
	if msg.Data != nil {
		SpanreedMessage.ProxyDestinationMessageAddAppData(b, appDataLoc)
	}
	cmsg := SpanreedMessage.ProxyDestinationMessageEnd(b)
	b.Finish(cmsg)
	buf := b.FinishedBytes()
	s.outgoingMessages <- outgoingMessage{
		addr:    addr,
		payload: buf,
	}

	return nil
}

func (s *udpSpanreedDestination) onProxyRequestClose(msg handlers.ClientCloseCommand) error {
	addr := func() *net.UDPAddr {
		s.mut_destinationConnections.RLock()
		defer s.mut_destinationConnections.RUnlock()

		addr, has := s.destinationConnections[msg.ClientId]
		if !has {
			return nil
		}
		return addr.Address
	}()
	if addr == nil {
		s.log.Warn("Cannot forward message, no client address found", zap.Uint32("clientId", msg.ClientId))
		// TODO (sessamekesh): Error state here instead
		return nil
	}

	b := flatbuffers.NewBuilder(64)
	SpanreedMessage.ProxyDestCloseConnectionStart(b)
	SpanreedMessage.ProxyDestCloseConnectionAddClientId(b, msg.ClientId)
	inner_msg := SpanreedMessage.ProxyDestCloseConnectionEnd(b)

	SpanreedMessage.ProxyDestinationMessageStart(b)
	SpanreedMessage.ProxyDestinationMessageAddInnerMessageType(b, SpanreedMessage.ProxyDestInnerMsgProxyDestCloseConnection)
	SpanreedMessage.ProxyDestinationMessageAddInnerMessage(b, inner_msg)
	cmsg := SpanreedMessage.ProxyDestinationMessageEnd(b)
	b.Finish(cmsg)
	buf := b.FinishedBytes()
	s.outgoingMessages <- outgoingMessage{
		addr:    addr,
		payload: buf,
	}

	s.mut_destinationConnections.Lock()
	defer s.mut_destinationConnections.Unlock()

	delete(s.destinationConnections, msg.ClientId)

	return nil
}

func (s *udpSpanreedDestination) onReceiveVerdict(msg *SpanreedMessage.ConnectionVerdict, appData []byte) {
	s.log.Info("Received verdict message", zap.Uint32("clientId", msg.ClientId()), zap.Bool("verdict", msg.Accepted()))

	s.mut_destinationConnections.RLock()
	defer s.mut_destinationConnections.RUnlock()

	_, has := s.destinationConnections[msg.ClientId()]
	if !has {
		s.log.Warn("Received a verdict for a client that's not registered", zap.Uint32("clientId", msg.ClientId()))
		return
	}

	s.proxyConnection.OpenClientVerdictChannel <- handlers.OpenClientConnectionVerdict{
		ClientId: msg.ClientId(),
		Verdict:  msg.Accepted(),
		AppData:  appData,
		Error:    nil,
	}
}

func (s *udpSpanreedDestination) onReceiveProxyMessage(msg *SpanreedMessage.ProxyMessage, appData []byte) {
	s.mut_destinationConnections.RLock()
	defer s.mut_destinationConnections.RUnlock()

	_, has := s.destinationConnections[msg.ClientId()]
	if !has {
		s.log.Warn("Received a message for a client that's not registered", zap.Uint32("clientId", msg.ClientId()))
		return
	}

	s.proxyConnection.IncomingMessageChannel <- handlers.DestinationMessage{
		ClientId:      msg.ClientId(),
		RecvTimestamp: s.proxyConnection.GetNowTimestamp(),
		Data:          appData,
	}
}

func (s *udpSpanreedDestination) onReceiveCloseRequest(msg *SpanreedMessage.CloseConnection, appData []byte) {
	s.mut_destinationConnections.RLock()
	defer s.mut_destinationConnections.RUnlock()

	_, has := s.destinationConnections[msg.ClientId()]
	if !has {
		s.log.Warn("Received a verdict for a client that's not registered", zap.Uint32("clientId", msg.ClientId()))
		return
	}

	s.proxyConnection.DestinationCloseRequests <- handlers.ClientCloseCommand{
		ClientId: msg.ClientId(),
		Reason:   string(msg.Reason()),
		Error:    nil,
		AppData:  appData,
	}
}
