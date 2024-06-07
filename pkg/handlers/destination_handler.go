package handlers

type DestinationMessageHandler struct {
	Name                              string
	GetNowTimestamp                   func() int64
	IncomingDestinationMessageChannel chan<- IncomingDestinationMessage
	OutgoingDestinationMessageChannel <-chan OutgoingDestinationMessage
	ConnectionOpenRequests            <-chan OpenDestinationConnectionCommand
	CloseRequests                     <-chan DestinationCloseCommand
	MatchConnectionString             func(connStr string) bool
}
