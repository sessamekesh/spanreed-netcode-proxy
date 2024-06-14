package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sessamekesh/spanreed-netcode-proxy/internal"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/errors"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"go.uber.org/zap"
)

type MissingDestinationHandler struct {
	Name string
}

func (e *MissingDestinationHandler) Error() string {
	return fmt.Sprintf("Missing destination handler with name %s", e.Name)
}

type MissingClientHandler struct {
	Name string
}

func (e *MissingClientHandler) Error() string {
	return fmt.Sprintf("Missing client message handler with name %s", e.Name)
}

type NoMatchingHandlerForConnectionString struct {
	ConnectionString string
}

func (e *NoMatchingHandlerForConnectionString) Error() string {
	return fmt.Sprintf("No destination handler can match connection string %s", e.ConnectionString[0:32])
}

type clientHandlerChannels struct {
	connectionVerdicts chan<- handlers.OpenClientConnectionVerdict
	outgoingMessages   chan<- handlers.ClientMessage
	closeRequests      chan<- handlers.ClientCloseCommand
}

type destinationHandlerChannels struct {
	outgoingMessages       chan<- handlers.DestinationMessage
	openConnectionsChannel chan<- handlers.OpenClientConnectionCommand
	closeRequests          chan<- handlers.ClientCloseCommand
}

type proxy struct {
	config      ProxyConfig
	magicNumber uint32
	version     uint8

	clientStore *internal.ClientStore
	startTime   time.Time

	incomingClientMessageSendChannel        chan<- handlers.ClientMessage
	incomingClientMessageRecvChannel        <-chan handlers.ClientMessage
	incomingClientCloseRequestsSendChannel  chan<- handlers.ClientCloseCommand
	incomingClientCloseRequestsRecvChannel  <-chan handlers.ClientCloseCommand
	incomingClientOpenConnectionSendChannel chan<- handlers.OpenClientConnectionCommand
	incomingClientOpenConnectionRecvChannel <-chan handlers.OpenClientConnectionCommand

	incomingDestinationVerdictSendChannel       chan<- handlers.OpenClientConnectionVerdict
	incomingDesetinationVerdictRecvChannel      <-chan handlers.OpenClientConnectionVerdict
	incomingDestinationMessageSendChannel       chan<- handlers.DestinationMessage
	incomingDestinationMessageRecvChannel       <-chan handlers.DestinationMessage
	incomingDestinationCloseRequestsSendChannel chan<- handlers.ClientCloseCommand
	incomingDestinationCloseRequestsRecvChannel <-chan handlers.ClientCloseCommand

	mut_outgoingClientMessageChannels sync.RWMutex
	outgoingClientMessageChannels     map[string]clientHandlerChannels

	mut_outgoingDestinationMessageSendChannels sync.RWMutex
	outgoingDestinationMessageSendChannels     map[string]destinationHandlerChannels

	mut_connectedDestinationHandlers sync.RWMutex
	connectedDestinationHandlers     []*handlers.DestinationMessageHandler

	clientMessageTimeout      time.Duration
	clientAuthTimeout         time.Duration
	destinationMessageTimeout time.Duration
	maxConnectionLength       time.Duration
	connectionKickLoopTime    time.Duration

	log *zap.Logger
}

type ProxyConfig struct {
	MagicNumber uint32
	Version     uint8

	IncomingClientConnectionBufferLength int

	IncomingClientMessageBufferLength      int
	IncomingDestinationMessageBufferLength int

	OutgoingClientMessageBufferLength      int
	OutgoingDestinationMessageBufferLength int

	MaxConcurrentConnections int

	// Amount of time the client or destination is allowed to be silent before connections are closed
	ClientMessageTimeoutMilliseconds      int
	ClientAuthTimeoutMilliseconds         int
	DestinationMessageTimeoutMilliseconds int
	MaxConnectionLengthMilliseconds       int
	ConnectionKickLoopMilliseconds        int

	Logger *zap.Logger
}

func CreateProxy(config ProxyConfig) *proxy {
	incomingClientMessageBufferLength := 256
	incomingDestinationMessageBufferLength := 256
	maxConcurrentConnections := 512

	if config.IncomingClientMessageBufferLength > 0 {
		incomingClientMessageBufferLength = config.IncomingClientMessageBufferLength
	}
	if config.IncomingDestinationMessageBufferLength > 0 {
		incomingDestinationMessageBufferLength = config.IncomingDestinationMessageBufferLength
	}
	if config.MaxConcurrentConnections > 0 {
		maxConcurrentConnections = config.MaxConcurrentConnections
	}

	incomingClientMessagesChannel := make(chan handlers.ClientMessage, incomingClientMessageBufferLength)
	incomingClientCloseRequests := make(chan handlers.ClientCloseCommand, incomingClientMessageBufferLength)
	incomingClientOpenRequests := make(chan handlers.OpenClientConnectionCommand, incomingClientMessageBufferLength)

	incomingDestinationMessagesChannel := make(chan handlers.DestinationMessage, incomingDestinationMessageBufferLength)
	incomingDestinationVerdictsChannel := make(chan handlers.OpenClientConnectionVerdict, incomingDestinationMessageBufferLength)
	incomingDestinationCloseRequests := make(chan handlers.ClientCloseCommand, incomingDestinationMessageBufferLength)

	var magicNumber uint32 = 0x52554259
	var version uint8 = 0
	if config.MagicNumber != 0 {
		magicNumber = config.MagicNumber
	}
	if config.Version > 0 {
		version = config.Version
	}

	clientMessageTimeout := 60 * time.Second
	destinationMessageTimeout := 15 * time.Second
	clientAuthTimeout := 10 * time.Second
	if config.ClientMessageTimeoutMilliseconds > 0 {
		clientMessageTimeout = time.Millisecond * time.Duration(config.ClientMessageTimeoutMilliseconds)
	}
	if config.DestinationMessageTimeoutMilliseconds > 0 {
		destinationMessageTimeout = time.Millisecond * time.Duration(config.DestinationMessageTimeoutMilliseconds)
	}
	if config.ClientAuthTimeoutMilliseconds > 0 {
		clientAuthTimeout = time.Millisecond * time.Duration(config.ClientAuthTimeoutMilliseconds)
	}

	maxConnectionLength := 6 * time.Hour
	if config.MaxConnectionLengthMilliseconds > 0 {
		maxConnectionLength = time.Millisecond * time.Duration(config.MaxConnectionLengthMilliseconds)
	}

	connectionKickLoopTime := 3 * time.Second
	if config.ConnectionKickLoopMilliseconds > 0 {
		connectionKickLoopTime = time.Millisecond * time.Duration(config.ConnectionKickLoopMilliseconds)
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	return &proxy{
		config:      config,
		magicNumber: magicNumber,
		version:     version,

		clientStore: internal.CreateClientStore(maxConcurrentConnections),
		startTime:   time.Now(),

		incomingClientMessageSendChannel:        incomingClientMessagesChannel,
		incomingClientMessageRecvChannel:        incomingClientMessagesChannel,
		incomingClientCloseRequestsSendChannel:  incomingClientCloseRequests,
		incomingClientCloseRequestsRecvChannel:  incomingClientCloseRequests,
		incomingClientOpenConnectionSendChannel: incomingClientOpenRequests,
		incomingClientOpenConnectionRecvChannel: incomingClientOpenRequests,

		incomingDestinationMessageSendChannel:       incomingDestinationMessagesChannel,
		incomingDestinationMessageRecvChannel:       incomingDestinationMessagesChannel,
		incomingDestinationVerdictSendChannel:       incomingDestinationVerdictsChannel,
		incomingDesetinationVerdictRecvChannel:      incomingDestinationVerdictsChannel,
		incomingDestinationCloseRequestsSendChannel: incomingDestinationCloseRequests,
		incomingDestinationCloseRequestsRecvChannel: incomingDestinationCloseRequests,

		mut_outgoingClientMessageChannels: sync.RWMutex{},
		outgoingClientMessageChannels:     make(map[string]clientHandlerChannels),

		mut_outgoingDestinationMessageSendChannels: sync.RWMutex{},
		outgoingDestinationMessageSendChannels:     make(map[string]destinationHandlerChannels),

		mut_connectedDestinationHandlers: sync.RWMutex{},
		connectedDestinationHandlers:     []*handlers.DestinationMessageHandler{},

		clientMessageTimeout:      clientMessageTimeout,
		clientAuthTimeout:         clientAuthTimeout,
		destinationMessageTimeout: destinationMessageTimeout,
		maxConnectionLength:       maxConnectionLength,
		connectionKickLoopTime:    connectionKickLoopTime,

		log: logger.With(zap.String("handler", "spanreed_proxy_hub")),
	}
}

func (p *proxy) getNowTime() int64 {
	return time.Since(p.startTime).Microseconds()
}

func (p *proxy) GetMagicNumberAndVersion() (uint32, uint8) {
	return p.magicNumber, p.version
}

func (p *proxy) CreateClientMessageHandler(name string) (*handlers.ClientMessageHandler, error) {
	p.mut_outgoingClientMessageChannels.Lock()
	defer p.mut_outgoingClientMessageChannels.Unlock()

	if _, alreadyHasName := p.outgoingClientMessageChannels[name]; alreadyHasName {
		return nil, &errors.NameCollision{
			CollisionContext: "CreateClientMessageHandler",
			Name:             name,
		}
	}

	outgoingMessageChannelLength := 256
	if p.config.OutgoingClientMessageBufferLength > 0 {
		outgoingMessageChannelLength = p.config.OutgoingClientMessageBufferLength
	}
	incomingClientConnectionBufferLength := 32
	if p.config.IncomingClientConnectionBufferLength > 0 {
		incomingClientConnectionBufferLength = p.config.IncomingClientConnectionBufferLength
	}

	outgoingMessageChannel := make(chan handlers.ClientMessage, outgoingMessageChannelLength)
	outgoingVerdictsChannel := make(chan handlers.OpenClientConnectionVerdict, incomingClientConnectionBufferLength)
	closeRequests := make(chan handlers.ClientCloseCommand, incomingClientConnectionBufferLength)

	p.outgoingClientMessageChannels[name] = clientHandlerChannels{
		connectionVerdicts: outgoingVerdictsChannel,
		outgoingMessages:   outgoingMessageChannel,
		closeRequests:      closeRequests,
	}

	return &handlers.ClientMessageHandler{
		Name:            name,
		GetNextClientId: p.getClientIdCbForMessageHandler(name),
		GetNowTimestamp: p.getNowTime,

		OpenClientChannel:        p.incomingClientOpenConnectionSendChannel,
		OpenClientVerdictChannel: outgoingVerdictsChannel,

		IncomingMessageChannel: p.incomingClientMessageSendChannel,
		OutgoingMessageChannel: outgoingMessageChannel,

		IncomingCloseRequests: p.incomingClientCloseRequestsRecvChannel,
		OutgoingCloseRequests: closeRequests,
	}, nil
}

func (p *proxy) getClientIdCbForMessageHandler(name string) func() (uint32, error) {
	return func() (uint32, error) {
		clientId := p.clientStore.GetNewClientId()

		err := p.clientStore.CreateClient(clientId, name, p.getNowTime())
		if err != nil {
			return 0, err
		}

		return clientId, nil
	}
}

func (p *proxy) CreateDestinationMessageHandler(name string, matchConnectionStringFn func(string) bool) (*handlers.DestinationMessageHandler, error) {
	p.mut_outgoingDestinationMessageSendChannels.Lock()
	defer p.mut_outgoingDestinationMessageSendChannels.Unlock()

	if _, alreadyHasName := p.outgoingDestinationMessageSendChannels[name]; alreadyHasName {
		return nil, &errors.NameCollision{
			CollisionContext: "CreateDestinationMessageHandler",
			Name:             name,
		}
	}

	outgoingMessageChannelLength := 256
	if p.config.OutgoingDestinationMessageBufferLength > 0 {
		outgoingMessageChannelLength = p.config.OutgoingDestinationMessageBufferLength
	}
	incomingClientConnectionBufferLength := 32
	if p.config.IncomingClientConnectionBufferLength > 0 {
		incomingClientConnectionBufferLength = p.config.IncomingClientConnectionBufferLength
	}

	outgoingMessageChannel := make(chan handlers.DestinationMessage, outgoingMessageChannelLength)
	openConnectionsChannel := make(chan handlers.OpenClientConnectionCommand, incomingClientConnectionBufferLength)
	closeRequestsChannel := make(chan handlers.ClientCloseCommand, incomingClientConnectionBufferLength)

	p.mut_connectedDestinationHandlers.Lock()
	defer p.mut_connectedDestinationHandlers.Unlock()

	newHandler := &handlers.DestinationMessageHandler{
		MatchConnectionString: matchConnectionStringFn,
		Name:                  name,
		GetNowTimestamp:       p.getNowTime,

		OpenClientChannel:        openConnectionsChannel,
		OpenClientVerdictChannel: p.incomingDestinationVerdictSendChannel,

		IncomingMessageChannel: p.incomingDestinationMessageSendChannel,
		OutgoingMessageChannel: outgoingMessageChannel,

		OutgoingCloseRequests: closeRequestsChannel,
		IncomingCloseRequests: p.incomingDestinationCloseRequestsRecvChannel,
	}

	destinationHandlerChannels := destinationHandlerChannels{
		outgoingMessages:       outgoingMessageChannel,
		openConnectionsChannel: openConnectionsChannel,
		closeRequests:          closeRequestsChannel,
	}

	p.outgoingDestinationMessageSendChannels[name] = destinationHandlerChannels
	p.connectedDestinationHandlers = append(p.connectedDestinationHandlers, newHandler)

	return newHandler, nil
}

func (p *proxy) Start(ctx context.Context) {
	p.log.Info("Starting Spanreed Proxy Hub!")
	wg := sync.WaitGroup{}

	//
	// Client message handling goroutine...
	wg.Add(1)
	go func() {
		p.log.Info("Starting Spanreed goroutine for handling incoming client messages...")
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				p.log.Info("Spanreed incoming client message goroutine attempting graceful shutdown")
				return
			case clientMsg := <-p.incomingClientMessageRecvChannel:
				p.log.Debug("Spanreed proxy received incoming message from client handler to forward on to destination")
				err := p.forwardClientMessage(clientMsg)
				if err == nil {
					p.clientStore.SetClientRecvTimestamp(clientMsg.ClientId, p.getNowTime())
				} else {
					p.log.Error("Error handling client message", zap.Uint32("clientId", clientMsg.ClientId), zap.Error(err))
				}
			case connRequest := <-p.incomingClientOpenConnectionRecvChannel:
				p.log.Debug("Spanreed proxy received connection request from client handler")
				connErr := p.handleClientConnectionRequestMessage(connRequest)
				if connErr != nil {
					p.log.Error("Error handling client connection request", zap.Uint32("clientId", connRequest.ClientId), zap.Error(connErr))
				}
			case closeRequest := <-p.incomingClientCloseRequestsRecvChannel:
				p.log.Debug("Spanreed proxy received a client close request from a client")
				kickErr := p.kickClient(closeRequest.ClientId, closeRequest.Reason)
				if kickErr != nil {
					p.log.Error("Error handling client kick request", zap.Error(kickErr))
				}
			}
		}
	}()

	//
	// Server message handling goroutine...
	wg.Add(1)
	go func() {
		p.log.Info("Starting Spanreed goroutine for handling incoming destination messages...")
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				p.log.Info("Spanreed incoming destination message goroutine attempting graceful shutdown")
				return
			case destinationMsg := <-p.incomingDestinationMessageRecvChannel:
				p.log.Debug("Spanreed proxy received incoming message from destination handler to forward on to client")
				err := p.forwardDestinationMessage(destinationMsg)
				if err == nil {
					p.clientStore.SetDestinationRecvTimestamp(destinationMsg.ClientId, p.getNowTime())
				}
			case verdict := <-p.incomingDesetinationVerdictRecvChannel:
				err := p.forwardDestinationVerdict(verdict)
				if err != nil {
					p.log.Error("Error forwarding destination verdict", zap.Uint32("clientId", verdict.ClientId), zap.Error(err))
				}
			case kick := <-p.incomingDestinationCloseRequestsRecvChannel:
				p.log.Debug("Spanreed proxy received a destination close request from a client")
				kickErr := p.kickClient(kick.ClientId, kick.Reason)
				if kickErr != nil {
					p.log.Error("Error handling destination kick request", zap.Error(kickErr))
				}
			}
		}
	}()

	//
	// Message timeouts (kick silent or ancient connections)
	wg.Add(1)
	go func() {
		p.log.Info("Starting Spanreed goroutine for handling timeouts", zap.Duration("interval", p.connectionKickLoopTime))
		defer wg.Done()

		// TODO (sessamekesh): Move this duration to configuration too
		ticker := time.NewTicker(p.connectionKickLoopTime)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				p.log.Info("Spanreed connection timeout goroutine attempting graceful shutdown...")
				return
			case <-ticker.C:
				p.kickOldClients()
				p.checkForAuthTimeouts()
			}
		}
	}()

	wg.Wait()

	// TODO (sessamekesh): Begin graceful shutdown procedure here (notify all connected clients and destinations of shutdown)
}

func (p *proxy) forwardClientMessage(msg handlers.ClientMessage) error {
	deqTime := p.getNowTime()

	destHandlerName, destHandlerErr := p.clientStore.GetDestinationHandlerName(msg.ClientId)
	if destHandlerErr != nil {
		return destHandlerErr
	}

	p.mut_outgoingDestinationMessageSendChannels.RLock()
	sendChannels, has := p.outgoingDestinationMessageSendChannels[destHandlerName]
	if !has {
		return &MissingDestinationHandler{
			Name: destHandlerName,
		}
	}

	sendChannels.outgoingMessages <- handlers.DestinationMessage{
		ClientId:       msg.ClientId,
		RecvTimestamp:  msg.RecvTimestamp,
		RouteTimestamp: deqTime,
		Data:           msg.Data,
	}

	return nil
}

func (p *proxy) forwardDestinationMessage(msg handlers.DestinationMessage) error {
	deqTime := p.getNowTime()

	clientHandlerName, clientHandlerNameErr := p.clientStore.GetClientHandlerName(msg.ClientId)
	if clientHandlerNameErr != nil {
		return clientHandlerNameErr
	}

	p.mut_outgoingClientMessageChannels.RLock()
	sendChannels, has := p.outgoingClientMessageChannels[clientHandlerName]
	if !has {
		return &MissingClientHandler{
			Name: clientHandlerName,
		}
	}

	sendChannels.outgoingMessages <- handlers.ClientMessage{
		ClientId:       msg.ClientId,
		RecvTimestamp:  msg.RecvTimestamp,
		RouteTimestamp: deqTime,
		Data:           msg.Data,
	}

	return nil
}

func (p *proxy) kickOldClients() {
	clientMsgDeadline := p.getNowTime() - p.clientMessageTimeout.Microseconds()
	destinationMsgDeadline := p.getNowTime() - p.destinationMessageTimeout.Microseconds()
	connectionDeadline := p.getNowTime() - p.maxConnectionLength.Microseconds()

	clientsToKick := p.clientStore.GetTimeoutClientList(clientMsgDeadline, destinationMsgDeadline, connectionDeadline)
	if len(clientsToKick) > 0 {
		p.log.Info("Kicking timed out clients", zap.Int64("clientMsgDeadline", clientMsgDeadline),
			zap.Int64("destinationMsgDeadline", destinationMsgDeadline), zap.Int64("connectionDeadline", connectionDeadline),
			zap.Int("client_count", len(clientsToKick)))
	}

	for _, clientId := range clientsToKick {
		kickErr := p.kickClient(clientId, &errors.ConnectionTimeout{})
		if kickErr != nil {
			p.log.Error("Error kicking timed out client", zap.Uint32("clientId", clientId), zap.Error(kickErr))
		}
	}
}

func (p *proxy) checkForAuthTimeouts() {
	authDeadline := p.getNowTime() - p.clientAuthTimeout.Microseconds()

	clientsToTimeout := p.clientStore.GetAuthTimeoutClientList(authDeadline)

	if len(clientsToTimeout) > 0 {
		p.log.Info("Kicking clients with auth timeouts", zap.Int64("authDeadline", authDeadline),
			zap.Int("client_count", len(clientsToTimeout)))
	}

	for _, clientId := range clientsToTimeout {
		kickErr := p.kickClient(clientId, &errors.AuthTimeout{})
		if kickErr != nil {
			p.log.Error("Error kicking auth rejected client", zap.Uint32("clientId", clientId), zap.Error(kickErr))
		}
	}
}

func (p *proxy) handleClientConnectionRequestMessage(incomingMsg handlers.OpenClientConnectionCommand) error {
	p.log.Info("Handle client connection request message", zap.Uint32("clientId", incomingMsg.ClientId))
	dequeTimestamp := p.getNowTime()

	rsl, err := p.clientStore.GetDestinationHandlerName(incomingMsg.ClientId)
	if err != nil {
		return err
	}
	if err == nil && rsl != "" {
		// No-op, this is a duplicate connection request, do not re-establish
		p.log.Warn("Unexpected duplicate client ID connection request, ignoring.", zap.Uint32("clientId", incomingMsg.ClientId))
		return nil
	}

	err = func() error {
		p.mut_connectedDestinationHandlers.RLock()
		defer p.mut_connectedDestinationHandlers.RUnlock()
		for _, destinationHandler := range p.connectedDestinationHandlers {
			if destinationHandler.MatchConnectionString(incomingMsg.ConnectionString) {
				p.mut_outgoingDestinationMessageSendChannels.RLock()
				defer p.mut_outgoingDestinationMessageSendChannels.RUnlock()

				destSendChannels, has := p.outgoingDestinationMessageSendChannels[destinationHandler.Name]
				if !has {
					return &MissingDestinationHandler{Name: destinationHandler.Name}
				}

				sdhErr := p.clientStore.SetDestinationHandlerName(incomingMsg.ClientId, destinationHandler.Name)
				if sdhErr != nil {
					p.log.Error("Error setting client destination handler name", zap.Uint32("clientId", incomingMsg.ClientId), zap.Error(sdhErr))
					return sdhErr
				}

				destSendChannels.openConnectionsChannel <- handlers.OpenClientConnectionCommand{
					ClientId:         incomingMsg.ClientId,
					RecvTimestamp:    incomingMsg.RecvTimestamp,
					RouteTimestamp:   dequeTimestamp,
					ConnectionString: incomingMsg.ConnectionString,
					AppData:          incomingMsg.AppData,
				}

				return nil
			}
		}

		return &NoMatchingHandlerForConnectionString{
			ConnectionString: incomingMsg.ConnectionString,
		}
	}()

	if err != nil {
		p.log.Error("Error attaching destination handler, attempting to close", zap.Uint32("clientId", incomingMsg.ClientId), zap.Error(err))
		p.kickClient(incomingMsg.ClientId, err)
	}
	return err
}

func (p *proxy) kickClient(clientId uint32, reason error) error {
	p.log.Info("Kicking client", zap.Uint32("clientId", clientId), zap.Error(reason))
	clientHandlerName, clientHandlerNameError := p.clientStore.GetClientHandlerName(clientId)
	if clientHandlerNameError != nil {
		p.log.Warn("Client handler missing", zap.Uint32("clientId", clientId), zap.Error(clientHandlerNameError))
	}

	destinationHandlerName, destinationHandlerNameError := p.clientStore.GetDestinationHandlerName(clientId)
	if destinationHandlerNameError != nil {
		p.log.Warn("Destination handler missing", zap.Uint32("clientId", clientId), zap.Error(clientHandlerNameError))
	}

	//
	// Client disconnect notification:
	func() {
		if clientHandlerName == "" {
			return
		}

		p.mut_outgoingClientMessageChannels.Lock()
		defer p.mut_outgoingClientMessageChannels.Unlock()

		clientHandler, has := p.outgoingClientMessageChannels[clientHandlerName]
		if !has {
			p.log.Warn("No outgoing client message channels registered", zap.Uint32("clientId", clientId), zap.String("clientHandlerName", clientHandlerName))
			return
		}

		clientHandler.closeRequests <- handlers.ClientCloseCommand{
			ClientId: clientId,
			Reason:   reason,
		}
	}()

	//
	// Destination disconnect notification:
	func() {
		if destinationHandlerName == "" {
			return
		}

		p.mut_outgoingDestinationMessageSendChannels.Lock()
		defer p.mut_outgoingDestinationMessageSendChannels.Unlock()

		destinationHandler, has := p.outgoingDestinationMessageSendChannels[destinationHandlerName]
		if !has {
			p.log.Warn("No outgoing destination message channels registered", zap.Uint32("clientId", clientId), zap.String("destinationHandlerName", destinationHandlerName))
			return
		}

		destinationHandler.closeRequests <- handlers.ClientCloseCommand{
			ClientId: clientId,
			Reason:   reason,
		}
	}()

	p.clientStore.RemoveClient(clientId)

	return nil
}

func (p *proxy) forwardDestinationVerdict(msg handlers.OpenClientConnectionVerdict) error {
	p.log.Debug("Forwarding verdict", zap.Uint32("clientId", msg.ClientId), zap.Bool("verdict", msg.Verdict), zap.Error(msg.Error))
	clientName, clientNameErr := p.clientStore.GetClientHandlerName(msg.ClientId)
	if clientNameErr != nil {
		return clientNameErr
	}

	p.mut_outgoingClientMessageChannels.RLock()
	outgoingChannels, has := p.outgoingClientMessageChannels[clientName]
	if !has {
		return &MissingClientHandler{
			Name: clientName,
		}
	}

	outgoingChannels.connectionVerdicts <- msg
	return nil
}
