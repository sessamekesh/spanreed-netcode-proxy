// Main package for the Spanreed Hub default implementation of the Spanreed library
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/proxy"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/transport"
	"go.uber.org/zap"
)

// TODO (sessamekesh): There's some sort of deadlock in here somewhere, around UDP shutdown.
//  Also something else that's preventing further messages from propagating all the way to the client?
//  Seems like the verdict makes it back, but then no further server messages do...

func main() {
	if dotenvErr := godotenv.Load(); dotenvErr != nil && !os.IsNotExist(dotenvErr) {
		fmt.Printf("Failed to load .env file! %s", dotenvErr.Error())
	}

	logger := zap.Must(zap.NewProduction())
	if os.Getenv("APP_ENV") != "production" {
		logger = zap.Must(zap.NewDevelopment())
	}
	defer logger.Sync()

	//
	// Flags
	certPath := os.Getenv("SPANREED_TLS_CERT_PATH")
	keyPath := os.Getenv("SPANREED_TLS_KEY_PATH")

	useWebsockets, useWsErr := strconv.ParseBool(os.Getenv("SPANREED_WEBSOCKET"))
	if useWsErr != nil {
		logger.Warn("Invalid value for SPANREED_WEBSOCKET, falling back to false")
	}
	wsPort, wsPortErr := strconv.ParseUint(os.Getenv("SPANREED_WEBSOCKET_PORT"), 0, 16)
	if wsPortErr != nil {
		logger.Warn("Invalid value for SPANREED_WEBSOCKET_PORT, falling back to 3000")
		wsPort = 3000
	}
	wsEndpoint := os.Getenv("SPANREED_WEBSOCKET_ENDPOINT")
	if wsEndpoint == "" {
		wsEndpoint = "/"
	}
	wsReadBufferSize, wsReadBufferSizeErr := strconv.ParseUint(os.Getenv("SPANREED_WEBSOCKET_READ_BUFFER_SIZE"), 0, 32)
	if wsReadBufferSizeErr != nil {
		logger.Warn("Invalid value for SPANREED_WEBSOCKET_READ_BUFFER_SIZE, defaulting to 10KB")
		wsReadBufferSize = 10240
	}
	wsWriteBufferSize, wsWriteBufferSizeErr := strconv.ParseUint(os.Getenv("SPANREED_WEBSOCKET_WRITE_BUFFER_SIZE"), 0, 32)
	if wsWriteBufferSizeErr != nil {
		logger.Warn("Invalid value for SPANREED_WEBSOCKET_WRITE_BUFFER_SIZE, defaulting to 10KB")
		wsWriteBufferSize = 10240
	}
	wsHandshakeTimeoutMs, wsHandshakeTimeoutMsErr := strconv.ParseUint(os.Getenv("SPANREED_WEBSOCKET_HANDSHAKE_TIMEOUT_MS"), 0, 32)
	if wsHandshakeTimeoutMsErr != nil {
		logger.Warn("Invalid value for SPANREED_WEBSOCKET_HANDSHAKE_TIMEOUT_MS, defaulting to 1500")
		wsHandshakeTimeoutMs = 1500
	}

	useWebtransport, useWebtransportErr := strconv.ParseBool(os.Getenv("SPANREED_WEBTRANSPORT"))
	if useWebtransportErr != nil {
		logger.Warn("Invalid value for SPANREED_WEBTRANSPORT, falling back to false")
	}
	wtPort, wtPortErr := strconv.ParseUint(os.Getenv("SPANREED_WEBTRANSPORT_PORT"), 0, 32)
	if wtPortErr != nil {
		logger.Warn("Invalid value for SPANREED_WEBTRANSPORT_PORT, falling back to 3000")
		wtPort = 3000
	}
	wtEndpoint := os.Getenv("SPANREED_WEBTRANSPORT_ENDPOINT")
	if wtEndpoint == "" {
		wtEndpoint = "/"
	}

	//
	// Client ingress parameters
	allowAllHosts, allowAllHostsErr := strconv.ParseBool(os.Getenv("SPANREED_ALLOW_ALL_HOSTS"))
	if allowAllHostsErr != nil {
		logger.Warn("Invalid value for SPANREED_ALLOW_ALL_HOSTS, falling back to false")
	}
	forbiddenHosts := os.Getenv("SPANREED_DENY_HOSTS")
	allowedHosts := os.Getenv("SPANREED_ALLOW_HOSTS")

	clientIncomingMessageQueueLength, clientIncomingMessageQueueLengthErr := strconv.ParseUint(os.Getenv("SPANREED_CLIENT_INCOMING_MESSAGE_QUEUE_LENGTH"), 0, 32)
	if clientIncomingMessageQueueLengthErr != nil {
		logger.Warn("Invalid value for SPANREED_CLIENT_INCOMING_MESSAGE_QUEUE_LENGTH, falling back to 16")
		clientIncomingMessageQueueLength = 16
	}
	clientOutgoingMessageQueueLength, clientOutgoingMessageQueueLengthErr := strconv.ParseUint(os.Getenv("SPANREED_CLIENT_OUTGOING_MESSAGE_QUEUE_LENGTH"), 0, 32)
	if clientOutgoingMessageQueueLengthErr != nil {
		logger.Warn("Invalid value for SPANREED_CLIENT_OUTGOING_MESSAGE_QUEUE_LENGTH, falling back to 16")
		clientOutgoingMessageQueueLength = 16
	}

	//
	// Destination egress parameters
	allowAllDestinations, allowAllDestinationsErr := strconv.ParseBool(os.Getenv("SPANREED_ALLOW_ALL_DESTINATIONS"))
	if allowAllDestinationsErr != nil {
		logger.Warn("Invalid value for SPANREED_ALLOW_ALL_DESTINATIONS, falling back to false")
	}
	allowedDestinations := os.Getenv("SPANREED_ALLOW_DESTINATIONS")
	forbiddenDestinations := os.Getenv("SPANREED_DENY_DESTINATIONS")

	useUdp, useUdpErr := strconv.ParseBool(os.Getenv("SPANREED_UDP"))
	if useUdpErr != nil {
		logger.Warn("Invalid value for SPANREED_UDP, falling back to false")
	}
	udpPort, udpPortErr := strconv.ParseUint(os.Getenv("SPANREED_UDP_PORT"), 0, 16)
	if udpPortErr != nil {
		logger.Warn("Invalid value for SPANREED_UDP_PORT, falling back to 30321")
		udpPort = 30321
	}

	//
	// Flag validation
	if (keyPath == "") != (certPath == "") {
		logger.Error("Cannot use TLS without providing both key and cert")
		return
	}

	//
	// Proxy setup + attach client handlers
	proxy := proxy.CreateProxy(proxy.ProxyConfig{
		Logger: logger,
	})
	magicNumber, version := proxy.GetMagicNumberAndVersion()
	shutdownCtx, shutdownRelease := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer shutdownRelease()

	wg := sync.WaitGroup{}

	if useWebsockets {
		wsHandler, wsHandlerErr := proxy.CreateClientMessageHandler("WebSocket")
		if wsHandlerErr != nil {
			logger.Error("Failed to create WebSocket client message handler", zap.Error(wsHandlerErr))
			return
		}

		wsServer, wsServerErr := transport.CreateWebsocketHandler(wsHandler, transport.WebsocketSpanreedClientParams{
			ListenAddress:    fmt.Sprintf(":%d", wsPort),
			ListenEndpoint:   wsEndpoint,
			Logger:           logger,
			AllowAllHosts:    allowAllHosts,
			AllowlistedHosts: strings.Split(allowedHosts, ","),
			DenylistedHosts:  strings.Split(forbiddenHosts, ","),
			CertPath:         certPath,
			KeyPath:          keyPath,

			IncomingMessageQueueLength: uint32(clientIncomingMessageQueueLength),
			OutgoingMessageQueueLength: uint32(clientOutgoingMessageQueueLength),

			IncomingMessageReadLimit:     uint32(wsReadBufferSize),
			OutgoingMessageWriteLimit:    uint32(wsWriteBufferSize),
			HandshakeTimeoutMilliseconds: uint32(wsHandshakeTimeoutMs),
		})
		if wsServerErr != nil {
			logger.Error("Failed to create WebSocket server", zap.Error(wsServerErr))
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("Starting WebSocket server", zap.Uint16("port", uint16(wsPort)), zap.Bool("ssl", certPath != "" && keyPath != ""))
			defer logger.Info("Successfully shutdown WebSocket server")
			wsServer.Start(shutdownCtx)
		}()
	}

	if useWebtransport && keyPath != "" && certPath != "" {
		wtHandler, wtErr := proxy.CreateClientMessageHandler("WebTransport")
		if wtErr != nil {
			logger.Error("Failed to create WebTransport client message handler", zap.Error(wtErr))
			return
		}

		wtServer, wtServerErr := transport.CreateWebtransportHandler(wtHandler, transport.WebtransportSpanreedClientParams{
			ListenAddress:    fmt.Sprintf(":%d", wtPort),
			ListenEndpoint:   wtEndpoint,
			Logger:           logger,
			CertPath:         certPath,
			KeyPath:          keyPath,
			AllowAllHosts:    allowAllHosts,
			AllowlistedHosts: strings.Split(allowedHosts, ","),
			DenylistedHosts:  strings.Split(forbiddenHosts, ","),

			IncomingMessageQueueLength: uint32(clientIncomingMessageQueueLength),
			OutgoingMessageQueueLength: uint32(clientOutgoingMessageQueueLength),
		})
		if wtServerErr != nil {
			logger.Error("Failed to create WebTransport server", zap.Error(wtServerErr))
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("Starting WebTransport server", zap.Uint16("port", uint16(wtPort)))
			defer logger.Info("Successfully shutdown WebTransport server")
			wtServer.Start(shutdownCtx)
		}()
	}

	if useUdp {
		udpHandler, udpHandlerErr := proxy.CreateDestinationMessageHandler("UdpDestination", transport.DefaultUdpDestinationHandlerMatchConnectionStringFn)
		if udpHandlerErr != nil {
			logger.Error("Failed to create UDP destination message handler", zap.Error(udpHandlerErr))
			return
		}

		udpServer, udpServerError := transport.CreateUdpDestinationHandler(udpHandler, transport.UdpSpanreedDestinationParams{
			MagicNumber:             magicNumber,
			Version:                 version,
			UdpServerPort:           int(udpPort),
			AllowAllDestinations:    allowAllDestinations,
			AllowlistedDestinations: strings.Split(allowedDestinations, ","),
			DenylistedDestinations:  strings.Split(forbiddenDestinations, ","),
			Logger:                  logger,
		})
		if udpServerError != nil {
			logger.Error("Failed to create UDP server", zap.Error(udpServerError))
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("Starting UDP server", zap.Uint16("port", uint16(udpPort)))
			defer logger.Info("Successfully shutdown UDP server")
			if udpErr := udpServer.Start(shutdownCtx); udpErr != nil {
				logger.Error("Error running UDP server", zap.Error(udpErr))
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting proxy middleware")
		defer logger.Info("Proxy middleware successfully shut down")
		proxy.Start(shutdownCtx)
	}()

	wg.Wait()

	logger.Info("Successfully shutdown Spanreed Hub server!")
}
