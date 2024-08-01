package main

import (
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"go.uber.org/zap"
)

type UdpServerParams struct {
	Logger *zap.Logger
}

type udpServer struct {
	proxyConnection *handlers.DestinationMessageHandler
	logger          *zap.Logger
}

func CreateUdpServer(proxyConnection *handlers.DestinationMessageHandler, params UdpServerParams) (*udpServer, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	return &udpServer{
		proxyConnection: proxyConnection,
		logger:          logger,
	}, nil
}

// TODO (sessamekesh): Continue writing this
