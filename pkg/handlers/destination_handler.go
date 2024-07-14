package handlers

type DestinationMessageHandler struct {
	Name                  string
	GetNowTimestamp       func() int64
	MatchConnectionString func(connStr string) bool

	OpenClientChannel        <-chan OpenClientConnectionCommand
	OpenClientVerdictChannel chan<- OpenClientConnectionVerdict

	IncomingMessageChannel chan<- DestinationMessage
	OutgoingMessageChannel <-chan DestinationMessage

	ProxyCloseRequests       <-chan ClientCloseCommand
	DestinationCloseRequests chan<- ClientCloseCommand
}
