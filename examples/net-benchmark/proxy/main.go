package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sessamekesh/spanreed-netcode-proxy/examples/net-benchmark/proxy/bmproxy"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/proxy"

	"go.uber.org/zap"
)

func main() {
	if dotenvErr := godotenv.Load(); dotenvErr != nil && !os.IsNotExist(dotenvErr) {
		fmt.Printf("Failed to load .env file! %s", dotenvErr.Error())
	}

	logger := zap.Must(zap.NewProduction())
	if os.Getenv("APP_ENV") == "development" {
		logger = zap.Must(zap.NewDevelopment())
	}
	defer logger.Sync()

	//
	// Flags
	certPath := os.Getenv("SPANREED_TLS_CERT_PATH")
	keyPath := os.Getenv("SPANREED_TLS_KEY_PATH")
	serverEndpoint := os.Getenv("SPANREED_DESTINATION_ADDRESS")

	if certPath == "" || keyPath == "" {
		logger.Error("Need cert+key path for TLS setup")
		return
	}

	port, portErr := strconv.ParseUint(os.Getenv("SPANREED_WT_SERVER_PORT"), 0, 16)
	if portErr != nil {
		logger.Error("Invalid value for SPANREED_WT_SERVER_PORT, falling back to 3000")
		port = 3000
	}

	udpPort, udpPortErr := strconv.ParseUint(os.Getenv("SPANREED_UDP_SERVER_PORT"), 0, 16)
	if udpPortErr != nil {
		logger.Error("Invalid value for SPANREED_UDP_SERVER_PORT, falling back to 30300")
		udpPort = 30300
	}

	//
	// Proxy setup
	proxy := proxy.CreateProxy(proxy.ProxyConfig{
		Logger: logger,
	})
	shutdownCtx, shutdownRelease := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer shutdownRelease()

	//
	// WebTransport handler
	wtHandler, wtHandlerErr := proxy.CreateClientMessageHandler("Benchmark WebTransport Client Handler")
	if wtHandlerErr != nil {
		logger.Error("Could not create WT handler", zap.Error(wtHandlerErr))
		return
	}
	wtClient, wtClientErr := bmproxy.CreateWebtransportBenchmarkHandler(wtHandler, bmproxy.WebtransportBenchmarkClientParams{
		CertPath:   certPath,
		KeyPath:    keyPath,
		Logger:     logger,
		ServerAddr: serverEndpoint,
		ListenAddr: fmt.Sprintf("localhost:%d", port),
	})
	if wtClientErr != nil {
		logger.Error("Failed to create WT server", zap.Error(wtClientErr))
		return
	}

	//
	// UDP handler
	udpHandler, udpHandlerErr := proxy.CreateDestinationMessageHandler("Benchmark UDP Destination Handler", func(s string) bool { return true })
	if udpHandlerErr != nil {
		logger.Error("Could not create UDP handler", zap.Error(udpHandlerErr))
		return
	}
	udpServer, udpServerErr := bmproxy.CreateUdpServer(udpHandler, bmproxy.UdpServerParams{
		Logger:          logger,
		Port:            uint16(udpPort),
		DestinationAddr: serverEndpoint,
	})
	if udpServerErr != nil {
		logger.Error("Could not create UDP server", zap.Error(udpServerErr))
		return
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting proxy middleware")
		defer logger.Info("Stopping proxy middleware")
		proxy.Start(shutdownCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting top-level WT handler goroutine")
		defer logger.Info("Stopping top-level WT handler goroutine")
		wtClient.Start(shutdownCtx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting top-level UDP handler goroutine")
		defer logger.Info("Stopping top-level UDP handler goroutine")
		udpServer.Start(shutdownCtx)
	}()

	wg.Wait()

	logger.Info("Successfully shutdown Spanreed benchmark proxy service!")
}
