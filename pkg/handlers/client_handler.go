package handlers

type ClientMessageHandler struct {
	Name            string
	GetNextClientId func() (uint32, error)
	GetNowTimestamp func() int64

	OpenClientChannel        chan<- OpenClientConnectionCommand
	OpenClientVerdictChannel <-chan OpenClientConnectionVerdict

	IncomingMessageChannel chan<- ClientMessage
	OutgoingMessageChannel <-chan ClientMessage

	IncomingCloseRequests chan<- ClientCloseCommand
	OutgoingCloseRequests <-chan ClientCloseCommand
}
