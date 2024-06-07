package handlers

import (
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/client"
	spanreeddestination "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/spanreed_destination"
)

//
// Messages to/from the client connection

type IncomingClientMessage struct {
	ClientId uint32
	Message  *client.ClientMessage

	// Telemetry
	RecvTime int64
}

type OutgoingClientMessage struct {
	ClientId uint32
	Message  *spanreeddestination.SpanreedDestinationMessage

	// Telemetry
	RecvTime    int64
	ProcessTime int64
}

//
// Logistics around connection state

type ClientCloseCommand struct {
	ClientId uint32
	Reason   error
}
