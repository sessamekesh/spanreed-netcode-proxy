package proxy

type IncomingDestinationMessage struct {
	ClientId   uint32
	RawMessage []byte

	// Telemetry
	RecvTime int64
}

type OutgoingDestinationMessage struct {
	ClientId   uint32
	RawMessage []byte

	// Telemetry
	RecvTime int64
	DeqTime  int64
}
