package handlers

import (
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/destination"
	spanreedclient "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/spanreed_client"
)

type IncomingDestinationMessage struct {
	ClientId uint32
	Message  *destination.DestinationMessage

	// Telemetry
	RecvTime int64
}

type OutgoingDestinationMessage struct {
	ClientId uint32
	Message  *spanreedclient.SpanreedClientMessage

	// Telemetry
	RecvTime    int64
	ProcessTime int64
}

// Logistics around connection state
type OpenDestinationConnectionCommand struct {
	ClientId             uint32
	ConnectionRequestMsg *spanreedclient.SpanreedClientMessage
	ConnectionString     string

	// Telemetry
	RecvTime    int64
	ProcessTime int64
}

type DestinationCloseCommand struct {
	ClientId uint32
	Reason   error
}
