package transport

import (
	"context"
	"errors"
	goerrs "errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"

	destinationmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/destination"
	spanreedclient "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/spanreed_client"
)

type udpSpanreedDestinationChannels struct {
	Address          *net.UDPAddr
	OutgoingMessages chan<- handlers.OutgoingDestinationMessage
	CloseRequest     chan<- handlers.DestinationCloseCommand
}

type udpSpanreedDestination struct {
	params UdpSpanreedDestinationParams

	log       *zap.Logger
	stringGen *utils.RandomStringGenerator

	destinationMessageSerializer    *destinationmsg.DestinationMessageSerializer
	spanreedClientMessageSerializer *spanreedclient.SpanreedClientMessageSerializer

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
		params:    params,
		log:       logger.With(zap.String("handler", "udpDestination")),
		stringGen: utils.CreateRandomstringGenerator(time.Now().UnixMicro()),
		destinationMessageSerializer: &destinationmsg.DestinationMessageSerializer{
			MagicNumber: params.MagicNumber,
			Version:     params.Version,
		},
		spanreedClientMessageSerializer: &spanreedclient.SpanreedClientMessageSerializer{
			MagicNumber: params.MagicNumber,
			Version:     params.Version,
		},
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
			parsedMsg, msgParseError := s.destinationMessageSerializer.Parse(rawMsg)
			if msgParseError != nil {
				s.log.Warn("Failed to parse incoming datagram", zap.Error(msgParseError))
				continue
			}

			// TODO (sessamekesh): Handle parsed message correctly here!
			// TODO (sessamekesh): That involves making sure that the destination server is at the right address!
			s.log.Info("Parsed message received!", zap.Uint32("clientId", parsedMsg.ClientId), zap.String("clientAddr", clientAddr.String()))
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
			case openConnectionRequest := <-s.proxyConnection.ConnectionOpenRequests:
				s.log.Info("Connection open request", zap.Uint32("clientId", openConnectionRequest.ClientId), zap.String("connectionString", openConnectionRequest.ConnectionString))
				err := s.onConnectClient(openConnectionRequest)
				if err != nil {
					s.log.Warn("Could not open destination connection for client", zap.Uint32("clientId", openConnectionRequest.ClientId), zap.Error(err))
				}
			case closeRequest := <-s.proxyConnection.CloseRequests:
				s.log.Info("TODO: Handle close request", zap.Uint32("clientId", closeRequest.ClientId))
			case msgRequest := <-s.proxyConnection.OutgoingDestinationMessageChannel:
				err := s.onOutgoingMsg(msgRequest)
				if err != nil {
					s.log.Warn("Failed to forward message for client", zap.Uint32("clientId", msgRequest.ClientId), zap.Error(err))
				}
			}
		}
	}()

	wg.Wait()

	return nil
}

func (s *udpSpanreedDestination) onConnectClient(outgoingMsg handlers.OpenDestinationConnectionCommand) error {
	log := s.log.With(zap.Uint32("clientId", outgoingMsg.ClientId))
	udpAddr, udpAddrErr := s.resolveUdpAddr(outgoingMsg.ConnectionString)
	if udpAddrErr != nil {
		log.Error("Cannot resolve UDP address from connection string", zap.String("udpAddr", outgoingMsg.ConnectionString), zap.Error(udpAddrErr))
		return udpAddrErr
	}

	// TODO (sessamekesh): This
	log.Info("UDP addr", zap.String("udpAddr", udpAddr.String()))

	return nil
}

func (s *udpSpanreedDestination) onOutgoingMsg(_ handlers.OutgoingDestinationMessage) error {
	return nil
}
