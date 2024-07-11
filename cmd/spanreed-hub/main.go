// Main package for the Spanreed Hub default implementation of the Spanreed library
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/proxy"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/transport"
	"go.uber.org/zap"
)

// TODO (sessamekesh):

/*
Something like this:

spanreedHub, err := CreateSpanreedClientServer(config)

webSocketListener := CreateSpanreedWSListener(incoming)
webTransportListener := CreateSpanreedWTListener(incoming)
tcpClientManager := CreateSpanreedTCPClientManager(outgoing)
udpDatagramForwarder := CreateSpanreedUDPDatagramThingy(outgoing)

webSocketListener.AddDestinationRule(AllowedHostsOrWhatever)

Default server (launch as Docker image) should have pretty bare bones
*/
func main() {
	logger := zap.Must(zap.NewProduction())
	if os.Getenv("APP_ENV") != "production" {
		logger = zap.Must(zap.NewDevelopment())
	}
	defer logger.Sync()

	//
	// Flags
	useWebsockets := flag.Bool("websockets", true, "Set to false to disable WebSocket support")
	wsPort := flag.Int("ws-port", 3000, "Port on which the WebSocket server should run")
	wsEndpoint := flag.String("ws-endpoint", "/ws", "HTTP endpoint that listens for WebSocket connections")

	useUdp := flag.Bool("udp", true, "Set to false to disable UDP support")
	flag.Parse()

	//
	// Proxy setup + attach client handlers
	proxy := proxy.CreateProxy(proxy.ProxyConfig{
		Logger: logger,
	})
	magicNumber, version := proxy.GetMagicNumberAndVersion()
	shutdownCtx, shutdownRelease := context.WithCancel(context.Background())
	defer shutdownRelease()

	wg := sync.WaitGroup{}

	if *useWebsockets {
		wsHandler, wsHandlerErr := proxy.CreateClientMessageHandler("WebSocket")
		if wsHandlerErr != nil {
			logger.Error("Failed to create WebSocket client message handler", zap.Error(wsHandlerErr))
			return
		}

		wsServer, wsServerErr := transport.CreateWebsocketHandler(wsHandler, transport.WebsocketSpanreedClientParams{
			MagicNumber:    magicNumber,
			Version:        version,
			ListenAddress:  fmt.Sprintf(":%d", *wsPort),
			ListenEndpoint: *wsEndpoint,
			AllowAllHosts:  true,
			Logger:         logger,
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

	if *useUdp {
		udpHandler, udpHandlerErr := proxy.CreateDestinationMessageHandler("UdpDestination", transport.DefaultUdpDestinationHandlerMatchConnectionStringFn)
		if udpHandlerErr != nil {
			logger.Error("Failed to create UDP destination message handler", zap.Error(udpHandlerErr))
			return
		}

		udpServer, udpServerError := transport.CreateUdpDestinationHandler(udpHandler, transport.UdpSpanreedDestinationParams{
			MagicNumber:          magicNumber,
			Version:              version,
			UdpServerPort:        30321,
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
			logger.Info("Starting UDP server", zap.Int("port", 30321))
			udpServer.Start(shutdownCtx)
		}()
	}

	// TODO (sessamekesh): Add shutdownRelease to SIGTERM handler

	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy.Start(shutdownCtx)
	}()

	wg.Wait()
}
