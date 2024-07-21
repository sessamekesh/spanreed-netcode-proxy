// Main package for the Spanreed Hub default implementation of the Spanreed library
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/proxy"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/transport"
	"go.uber.org/zap"
)

func main() {
	logger := zap.Must(zap.NewProduction())
	if os.Getenv("APP_ENV") != "production" {
		logger = zap.Must(zap.NewDevelopment())
	}
	defer logger.Sync()

	//
	// Flags
	certPath := flag.String("cert", "", "Path to TLS cert file (requires key to be set also)")
	keyPath := flag.String("key", "", "Path to TLS key file (reqwuires cert to be set also)")

	useWebsockets := flag.Bool("websockets", true, "Set to false to disable WebSocket support")
	wsPort := flag.Int("ws-port", 3000, "Port on which the WebSocket server should run")
	wsEndpoint := flag.String("ws-endpoint", "/ws", "HTTP endpoint that listens for WebSocket connections")
	wsReadBufferSize := flag.Uint("ws-read-buffer-size", 0, "WebSocket read buffer size")
	wsWriteBufferSize := flag.Uint("ws-write-buffer-size", 0, "WebSocket write buffer size")
	wsHandshakeTimeoutMs := flag.Uint("ws-handshake-timeout", 0, "WebSocket handshake timeout, in milliseconds")

	useWebtransport := flag.Bool("webtransport", false, "Set true to enable WebTransport support. Requires cert and key to be set.")
	wtPort := flag.Int("wt-port", 3001, "Port on which WebTransport server should run")
	wtEndpoint := flag.String("wt-endpoint", "/wt", "HTTP3 endpoint that listens for WebTransport connections")

	allowAllhosts := flag.Bool("allow-all-hosts", false, "Set true to accept connections from all hosts (except forbidden hosts)")
	forbiddenHosts := flag.String("deny-hosts", "", "Comma-separated list of forbidden hosts")
	allowedHosts := flag.String("allow-hosts", "", "Comma-separated list of allowed hosts (if allow-all-hosts is false)")

	clientIncomingMessageQueueLength := flag.Uint("client-incoming-queue", 0, "Set size of incoming client message queue")
	clientOutgoingMessageQueueLength := flag.Uint("client-outgoing-queue", 0, "Set size of outgoing client message queue")

	useUdp := flag.Bool("udp", true, "Set to false to disable UDP support")
	udpPort := flag.Int("udp-port", 30321, "Port on which the UDP server operates")
	flag.Parse()

	//
	// Flag validation
	if (*keyPath == "") != (*certPath == "") {
		logger.Error("Cannot use TLS without providing both key and cert")
		return
	}

	//
	// Proxy setup + attach client handlers
	proxy := proxy.CreateProxy(proxy.ProxyConfig{
		Logger: logger,
	})
	magicNumber, version := proxy.GetMagicNumberAndVersion()
	shutdownCtx, shutdownRelease := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	defer shutdownRelease()

	wg := sync.WaitGroup{}

	if *useWebsockets {
		wsHandler, wsHandlerErr := proxy.CreateClientMessageHandler("WebSocket")
		if wsHandlerErr != nil {
			logger.Error("Failed to create WebSocket client message handler", zap.Error(wsHandlerErr))
			return
		}

		wsServer, wsServerErr := transport.CreateWebsocketHandler(wsHandler, transport.WebsocketSpanreedClientParams{
			ListenAddress:    fmt.Sprintf(":%d", *wsPort),
			ListenEndpoint:   *wsEndpoint,
			Logger:           logger,
			AllowAllHosts:    *allowAllhosts,
			AllowlistedHosts: strings.Split(*allowedHosts, ","),
			DenylistedHosts:  strings.Split(*forbiddenHosts, ","),
			CertPath:         *certPath,
			KeyPath:          *keyPath,

			IncomingMessageQueueLength: uint32(*clientIncomingMessageQueueLength),
			OutgoingMessageQueueLength: uint32(*clientOutgoingMessageQueueLength),

			IncomingMessageReadLimit:     uint32(*wsReadBufferSize),
			OutgoingMessageWriteLimit:    uint32(*wsWriteBufferSize),
			HandshakeTimeoutMilliseconds: uint32(*wsHandshakeTimeoutMs),
		})
		if wsServerErr != nil {
			logger.Error("Failed to create WebSocket server", zap.Error(wsServerErr))
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			wsServer.Start(shutdownCtx)
		}()
	}

	if *useWebtransport && *keyPath != "" && *certPath != "" {
		wtHandler, wtErr := proxy.CreateClientMessageHandler("WebTransport")
		if wtErr != nil {
			logger.Error("Failed to create WebTransport client message handler", zap.Error(wtErr))
			return
		}

		wtServer, wtServerErr := transport.CreateWebtransportHandler(wtHandler, transport.WebtransportSpanreedClientParams{
			ListenAddress:    fmt.Sprintf(":%d", *wtPort),
			ListenEndpoint:   *wtEndpoint,
			Logger:           logger,
			CertPath:         *certPath,
			KeyPath:          *keyPath,
			AllowAllHosts:    *allowAllhosts,
			AllowlistedHosts: strings.Split(*allowedHosts, ","),
			DenylistedHosts:  strings.Split(*forbiddenHosts, ","),

			IncomingMessageQueueLength: uint32(*clientIncomingMessageQueueLength),
			OutgoingMessageQueueLength: uint32(*clientOutgoingMessageQueueLength),
		})
		if wtServerErr != nil {
			logger.Error("Failed to create WebTransport server", zap.Error(wtServerErr))
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			wtServer.Start(shutdownCtx)
		}()
	}

	if *useUdp {
		udpHandler, udpHandlerErr := proxy.CreateDestinationMessageHandler("UdpDestination", transport.DefaultUdpDestinationHandlerMatchConnectionStringFn)
		if udpHandlerErr != nil {
			logger.Error("Failed to create UDP destination message handler", zap.Error(udpHandlerErr))
			return
		}

		udpServer, udpServerError := transport.CreateUdpDestinationHandler(udpHandler, transport.UdpSpanreedDestinationParams{
			MagicNumber:          magicNumber,
			Version:              version,
			UdpServerPort:        *udpPort,
			AllowAllDestinations: true,
			Logger:               logger,
		})
		if udpServerError != nil {
			logger.Error("Failed to create UDP server", zap.Error(udpServerError))
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("Starting UDP server", zap.Int("port", *udpPort))
			udpServer.Start(shutdownCtx)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy.Start(shutdownCtx)
	}()

	wg.Wait()
}
