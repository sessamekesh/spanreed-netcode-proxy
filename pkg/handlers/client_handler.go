package handlers

type ClientMessageHandler struct {
	Name            string
	GetNextClientId func() (uint32, error)
	GetNowTimestamp func() int64

	IncomingClientMessageChannel chan<- IncomingClientMessage
	OutgoingClientMessageChannel <-chan OutgoingClientMessage

	CloseRequests <-chan ClientCloseCommand
}
