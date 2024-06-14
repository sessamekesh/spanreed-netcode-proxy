package transport

import (
	"context"
	"errors"
	goerrs "errors"
	"fmt"
	"net"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/transport/SpanreedMessage"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"
)

type udpSpanreedDestinationChannels struct {
	Address          *net.UDPAddr
	OutgoingMessages chan<- handlers.ClientMessage
	CloseRequest     chan<- handlers.ClientCloseCommand
}

type udpSpanreedDestination struct {
	params UdpSpanreedDestinationParams

	log       *zap.Logger
	stringGen *utils.RandomStringGenerator

	resolveUdpAddr  func(connStr string) (*net.UDPAddr, error)
	proxyConnection *handlers.DestinationMessageHandler

	mut_destinationConnections sync.RWMutex
	destinationConnections     map[uint32]*udpSpanreedDestinationChannels
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

	ResolveUdpAddress func(connStr string) (*net.UDPAddr, error)
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

func DefaultUdpDestinationHandlerMatchConnectionStringFn(connStr string) bool {
	return connStr[0:4] == "udp:"
}

func defaultAddressResolution(connStr string) (*net.UDPAddr, error) {
	if connStr[0:4] != "udp:" {
		return nil, goerrs.New("not a UDP connection string")
	}

	return net.ResolveUDPAddr("udp", connStr[4:])
}

func CreateUdpDestinationHandler(proxyConnection *handlers.DestinationMessageHandler, params UdpSpanreedDestinationParams) (*udpSpanreedDestination, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}
	resolveUdpAddrFunc := params.ResolveUdpAddress
	if resolveUdpAddrFunc == nil {
		resolveUdpAddrFunc = defaultAddressResolution
	}

	return &udpSpanreedDestination{
		params:          params,
		log:             logger.With(zap.String("handler", "udpDestination")),
		stringGen:       utils.CreateRandomstringGenerator(time.Now().UnixMicro()),
		resolveUdpAddr:  resolveUdpAddrFunc,
		proxyConnection: proxyConnection,

		mut_destinationConnections: sync.RWMutex{},
		destinationConnections:     make(map[uint32]*udpSpanreedDestinationChannels),
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
	conn.SetReadBuffer(2048)
	conn.SetWriteBuffer(2048)

	//
	// Connection closing goroutine
	wg.Add(1)
	go func() {
		wg.Done()
		<-ctx.Done()
		conn.Close()
	}()

	//
	// Connection message listening goroutine
	wg.Add(1)
	go func() {
		wg.Done()
		for {
			var buf [1400]byte
			bytesRead, clientAddr, err := conn.ReadFromUDP(buf[0:])
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					s.log.Info("UDP server connection close requested - exiting connection message listening goroutine")
					return
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
					// TODO (sessamekesh): Handle cv verdict
					// TODO (sessamekesh): Make this safe!
				case SpanreedMessage.InnerMsgProxyMessage:
					pm := new(SpanreedMessage.ProxyMessage)
					pm.Init(ut.Bytes, ut.Pos)
					// TODO (sessamekesh): Handle this incoming message
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
		wg.Done()
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
				}
			case closeRequest := <-s.proxyConnection.IncomingCloseRequests:
				s.log.Info("TODO: Handle close request", zap.Uint32("clientId", closeRequest.ClientId))
			case msgRequest := <-s.proxyConnection.OutgoingMessageChannel:
				s.log.Info("TODO: Handle message request", zap.Uint32("clientId", msgRequest.ClientId))
			}
		}
	}()

	wg.Wait()

	return nil
}

func (s *udpSpanreedDestination) onConnectClient(outgoingMsg handlers.OpenClientConnectionCommand) error {
	log := s.log.With(zap.Uint32("clientId", outgoingMsg.ClientId))
	udpAddr, udpAddrErr := s.resolveUdpAddr(outgoingMsg.ConnectionString)
	if udpAddrErr != nil {
		log.Error("Cannot resolve UDP address from connection string", zap.String("udpAddr", outgoingMsg.ConnectionString), zap.Error(udpAddrErr))
		return udpAddrErr
	}

	// TODO (sessamekesh): Finish this
	log.Info("UDP addr", zap.String("udpAddr", udpAddr.String()))

	return nil
}
