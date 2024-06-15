package handlers

//
// Connection logistics (opening / closing connections)
type OpenClientConnectionCommand struct {
	ClientId         uint32
	RecvTimestamp    int64
	RouteTimestamp   int64
	ConnectionString string
	AppData          []byte
}

type OpenClientConnectionVerdict struct {
	ClientId uint32
	Verdict  bool
	Error    error
	AppData  []byte
}

type ClientCloseCommand struct {
	ClientId uint32
	Reason   string
	Error    error
	AppData  []byte
}

//
// Message Forwarding
type ClientMessage struct {
	ClientId       uint32
	RecvTimestamp  int64
	RouteTimestamp int64
	Data           []byte
}

type DestinationMessage struct {
	ClientId       uint32
	RecvTimestamp  int64
	RouteTimestamp int64
	Data           []byte
}
