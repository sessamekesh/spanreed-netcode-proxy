package bmproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"go.uber.org/zap"
)

type UdpServerParams struct {
	Logger          *zap.Logger
	Port            uint16
	DestinationAddr string
}

type udpServer struct {
	proxyConnection *handlers.DestinationMessageHandler
	logger          *zap.Logger
	params          UdpServerParams
	destAddr        *net.UDPAddr
}

func CreateUdpServer(proxyConnection *handlers.DestinationMessageHandler, params UdpServerParams) (*udpServer, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	udpAddr, udpAddrErr := net.ResolveUDPAddr("udp", params.DestinationAddr)
	if udpAddrErr != nil {
		logger.Error("Failed to resolve endpoint address", zap.Error(udpAddrErr))
		return nil, udpAddrErr
	}

	return &udpServer{
		proxyConnection: proxyConnection,
		logger:          logger,
		params:          params,
		destAddr:        udpAddr,
	}, nil
}

func (s *udpServer) Start(ctx context.Context) error {
	port := s.params.Port
	if port == 0 {
		port = 30300
	}

	hostAddr, hostAddrErr := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if hostAddrErr != nil {
		return hostAddrErr
	}

	conn, listenErr := net.ListenUDP("udp", hostAddr)
	if listenErr != nil {
		return listenErr
	}
	defer conn.Close()

	conn.SetReadBuffer(10240)
	conn.SetWriteBuffer(10240)

	wg := sync.WaitGroup{}
	is_running := true
	//
	// Destination listening goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		s.logger.Info("Starting UDP listener goroutine")
		defer s.logger.Info("Stopping UDP listener goroutine")

		for {
			conn.SetReadDeadline(time.Now().Add(time.Second * 15))
			var buf [10240]byte
			bytesRead, clientAddr, err := conn.ReadFromUDP(buf[0:])
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					s.logger.Info("UDP server connection close requested - exiting")
					return
				} else if strings.Contains(err.Error(), "WSAEMSGSIZE") {
					//  WINDOWS ONLY - Unix silently discards excess data.
					s.logger.Warn("Overflow of data, continuing but may be malformed")
				} else if strings.Contains(err.Error(), "i/o timeout") {
					if is_running {
						continue
					} else {
						return
					}
				} else {
					s.logger.Error("Error reading UDP datagram, closing", zap.Error(err))
					return
				}
			}

			if !clientAddr.IP.Equal(s.destAddr.IP) ||
				clientAddr.Port != s.destAddr.Port {
				s.logger.Info("Message did not come from destination, skipping", zap.String("clientAddr", clientAddr.String()),
					zap.String("expectedAddr", s.destAddr.String()))
				continue
			}

			payload := buf[0:bytesRead]

			msgType := GetDestinationMessageType(payload)
			clientId := GetDestinationClientId(payload)
			if clientId == 0xFFFFFFFF {
				s.logger.Error("Could not extract client ID, skipping")
				continue
			}

			switch msgType {
			case DestinationMessageType_Unknown:
				s.logger.Warn("Invalid message type from client")
				return
			case DestinationMessageType_ConnectionVerdict:
				s.logger.Info("Received auth verdict", zap.Uint32("clientId", clientId), zap.Bool("verdict", GetVerdict((payload))))
				s.proxyConnection.OpenClientVerdictChannel <- handlers.OpenClientConnectionVerdict{
					ClientId: clientId,
					Verdict:  GetVerdict(payload),
					AppData:  payload,
				}
			case DestinationMessageType_Pong:
				SetPongRecvTimestamp(payload, uint64(s.proxyConnection.GetNowTimestamp()))
				fallthrough
			case DestinationMessageType_Stats:
				s.proxyConnection.IncomingMessageChannel <- handlers.DestinationMessage{
					ClientId: clientId,
					Data:     payload,
				}
			}
		}
	}()

	//
	// Proxy listening goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		s.logger.Info("Starting UDP proxy message listener")
		defer s.logger.Info("Stopping UDP proxy message listener")

		for {
			select {
			case <-ctx.Done():
				is_running = false
				return
			case connReq := <-s.proxyConnection.OpenClientChannel:
				SetClientId(connReq.AppData, connReq.ClientId)
				conn.WriteToUDP(connReq.AppData, s.destAddr)
			case closeReq := <-s.proxyConnection.ProxyCloseRequests:
				SetClientId(closeReq.AppData, closeReq.ClientId)
				conn.WriteToUDP(closeReq.AppData, s.destAddr)
			case outgoingMessage := <-s.proxyConnection.OutgoingMessageChannel:
				SetClientId(outgoingMessage.Data, outgoingMessage.ClientId)
				msgType := GetClientMessageType(outgoingMessage.Data)
				if msgType == ClientMessageType_Ping {
					SetPingForwardTimestamp(outgoingMessage.Data, uint64(s.proxyConnection.GetNowTimestamp()))
				}
				conn.WriteToUDP(outgoingMessage.Data, s.destAddr)
			}
		}
	}()

	wg.Wait()

	return nil
}
