// Main package for the Spanreed Hub default implementation of the Spanreed library
package main

import (
	"context"
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
	if os.Getenv("APP_ENV") == "development" {
		logger = zap.Must(zap.NewDevelopment())
	}
	defer logger.Sync()

	proxy := proxy.CreateProxy(proxy.ProxyConfig{
		Logger: logger,
	})
	magicNumber, version := proxy.GetMagicNumberAndVersion()

	wsHandler, wsHandlerErr := proxy.CreateClientMessageHandler("WebSocket")
	if wsHandlerErr != nil {
		logger.Error("Failed to create WebSocket client message handler", zap.Error(wsHandlerErr))
		return
	}

	udpHandler, udpHandlerErr := proxy.CreateDestinationMessageHandler("UdpDestination", transport.DefaultUdpDestinationHandlerMatchConnectionStringFn)
	if udpHandlerErr != nil {
		logger.Error("Failed to create UDP destination message handler", zap.Error(udpHandlerErr))
		return
	}

	wsServer, wsServerErr := transport.CreateWebsocketHandler(wsHandler, transport.WebsocketSpanreedClientParams{
		MagicNumber:    magicNumber,
		Version:        version,
		ListenAddress:  ":8080",
		ListenEndpoint: "/ws",
		AllowAllHosts:  true,
		Logger:         logger,
	})
	if wsServerErr != nil {
		logger.Error("Failed to create WebSocket server", zap.Error(wsServerErr))
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

	shutdownCtx, shutdownRelease := context.WithCancel(context.Background())
	defer shutdownRelease()

	// TODO (sessamekesh): Add shutdownRelease to SIGTERM handler

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		wsServer.Start(shutdownCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		udpServer.Start(shutdownCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy.Start(shutdownCtx)
	}()

	wg.Wait()
}
