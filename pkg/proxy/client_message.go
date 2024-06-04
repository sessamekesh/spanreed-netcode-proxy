package proxy

type IncomingClientMessage struct {
	ClientId   uint32
	RawMessage []byte

	// Telemetry
	RecvTime int64
}

type OutgoingClientMessage struct {
	ClientId   uint32
	RawMessage []byte

	// Telemetry
	RecvTime int64
	DeqTime  int64
}

type ClientConnectionResponse struct {
	ClientId      uint32
	Verdict       bool
	IsTimeout     bool
	InternalError error

	// Telemetry
	RecvTime int64
	DeqTime  int64
}

type ClientCloseConnectionRequest struct {
	ClientId        uint32
	IsTimeout       bool
	UnderlyingError error

	// Telemetry
	RecvTime int64
	DeqTime  int64
}
