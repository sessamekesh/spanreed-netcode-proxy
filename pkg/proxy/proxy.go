package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sessamekesh/spanreed-netcode-proxy/internal"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/errors"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	clientmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/client"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/destination"
	spanclientmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/spanreed_client"
	spandestintationmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/spanreed_destination"
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
	outgoingMessages chan<- handlers.OutgoingClientMessage
	closeRequests    chan<- handlers.ClientCloseCommand
}

type destinationHandlerChannels struct {
	outgoingMessages       chan<- handlers.OutgoingDestinationMessage
	openConnectionsChannel chan<- handlers.OpenDestinationConnectionCommand
	closeRequests          chan<- handlers.DestinationCloseCommand
}

type proxy struct {
	config      ProxyConfig
	magicNumber uint32
	version     uint8

	clientStore *internal.ClientStore
	startTime   time.Time

	incomingClientMessageSendChannel chan<- handlers.IncomingClientMessage
	incomingClientMessageRecvChannel <-chan handlers.IncomingClientMessage

	incomingDestinationMessageSendChannel chan<- handlers.IncomingDestinationMessage
	incomingDestinationMessageRecvChannel <-chan handlers.IncomingDestinationMessage

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

	incomingClientMessagesChannel := make(chan handlers.IncomingClientMessage, incomingClientMessageBufferLength)
	incomingDestinationMessagesChannel := make(chan handlers.IncomingDestinationMessage, incomingDestinationMessageBufferLength)

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

		incomingClientMessageSendChannel: incomingClientMessagesChannel,
		incomingClientMessageRecvChannel: incomingClientMessagesChannel,

		incomingDestinationMessageSendChannel: incomingDestinationMessagesChannel,
		incomingDestinationMessageRecvChannel: incomingDestinationMessagesChannel,

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

	outgoingMessageChannel := make(chan handlers.OutgoingClientMessage, outgoingMessageChannelLength)
	closeRequests := make(chan handlers.ClientCloseCommand, incomingClientConnectionBufferLength)

	p.outgoingClientMessageChannels[name] = clientHandlerChannels{
		outgoingMessages: outgoingMessageChannel,
		closeRequests:    closeRequests,
	}

	return &handlers.ClientMessageHandler{
		Name:                         name,
		GetNextClientId:              p.getClientIdCbForMessageHandler(name),
		GetNowTimestamp:              p.getNowTime,
		IncomingClientMessageChannel: p.incomingClientMessageSendChannel,
		OutgoingClientMessageChannel: outgoingMessageChannel,
		CloseRequests:                closeRequests,
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

	outgoingMessageChannel := make(chan handlers.OutgoingDestinationMessage, outgoingMessageChannelLength)
	openConnectionsChannel := make(chan handlers.OpenDestinationConnectionCommand, incomingClientConnectionBufferLength)
	closeRequestsChannel := make(chan handlers.DestinationCloseCommand, incomingClientConnectionBufferLength)

	p.mut_connectedDestinationHandlers.Lock()
	defer p.mut_connectedDestinationHandlers.Unlock()

	newHandler := &handlers.DestinationMessageHandler{
		MatchConnectionString:             matchConnectionStringFn,
		Name:                              name,
		GetNowTimestamp:                   p.getNowTime,
		IncomingDestinationMessageChannel: p.incomingDestinationMessageSendChannel,
		OutgoingDestinationMessageChannel: outgoingMessageChannel,
		ConnectionOpenRequests:            openConnectionsChannel,
		CloseRequests:                     closeRequestsChannel,
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
				err := p.handleClientMessage(clientMsg)
				if err == nil {
					p.clientStore.SetClientRecvTimestamp(clientMsg.ClientId, p.getNowTime())
				} else {
					p.log.Error("Error handling client message", zap.Uint32("clientId", clientMsg.ClientId), zap.Error(err))
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
				err := p.handleDestinationMessage(destinationMsg)
				if err == nil {
					p.clientStore.SetDestinationRecvTimestamp(destinationMsg.ClientId, p.getNowTime())
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
		cmErr := p.sendClientMessage(clientId, p.getNowTime(), p.getNowTime(), &spandestintationmsg.SpanreedDestinationMessage{
			MagicNumber: p.magicNumber,
			Version:     p.version,
			MessageType: spandestintationmsg.SpanreedDestinationMessageType_ConnectionResponse,
			ConnectionResponse: &spandestintationmsg.ConnectionResponse{
				Verdict:   false,
				IsTimeout: true,
			},
		})
		if cmErr != nil {
			p.log.Error("Error sending auth timeout message to client", zap.Uint32("clientId", clientId), zap.Error(cmErr))
		}
		kickErr := p.kickClient(clientId, &errors.AuthTimeout{})
		if kickErr != nil {
			p.log.Error("Error kicking auth rejected client", zap.Uint32("clientId", clientId), zap.Error(kickErr))
		}
	}
}

func (p *proxy) handleClientMessage(incomingMsg handlers.IncomingClientMessage) error {
	p.log.Info("Handling client message", zap.Uint32("clientId", incomingMsg.ClientId))
	if !p.clientStore.HasClient(incomingMsg.ClientId) {
		return &internal.MissingClientIdError{
			Id: incomingMsg.ClientId,
		}
	}

	if incomingMsg.Message.MagicNumber != p.magicNumber || incomingMsg.Message.Version != p.version {
		return &errors.InvalidHeaderVersion{
			ExpectedMagicNumber: p.magicNumber,
			ExpectedVersion:     p.version,
			ActualMagicNumber:   incomingMsg.Message.MagicNumber,
			ActualVersion:       incomingMsg.Message.Version,
		}
	}

	switch incomingMsg.Message.MessageType {
	case clientmsg.ClientMessageType_ConnectionRequest:
		return p.handleClientConnectionRequestMessage(incomingMsg, incomingMsg.Message)
	case clientmsg.ClientMessageType_AppMessage:
		return p.handleClientAppMessage(incomingMsg, incomingMsg.Message)
	case clientmsg.ClientMessageType_NONE:
	default:
		// fallthrough
	}

	return &errors.InvalidEnumValue{
		EnumName: "IncomingClientMessage::Message::MessageType",
		IntValue: 0xff,
	}
}

func (p *proxy) handleClientConnectionRequestMessage(incomingMsg handlers.IncomingClientMessage, parsedMsg *clientmsg.ClientMessage) error {
	dequeTimestamp := p.getNowTime()

	if parsedMsg.ConnectionRequest == nil {
		return &errors.MissingFieldError{
			MessageName: "IncomingClientMessage::Message",
			FieldName:   "ConnectionRequest",
		}
	}

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
			if destinationHandler.MatchConnectionString(parsedMsg.ConnectionRequest.ConnectionString) {
				p.mut_outgoingDestinationMessageSendChannels.RLock()
				defer p.mut_outgoingDestinationMessageSendChannels.RUnlock()

				destSendChannels, has := p.outgoingDestinationMessageSendChannels[destinationHandler.Name]
				if !has {
					return &MissingDestinationHandler{Name: destinationHandler.Name}
				}

				spanreedMessage := &spanclientmsg.SpanreedClientMessage{
					MagicNumber: p.magicNumber,
					Version:     p.version,
					MessageType: spanclientmsg.SpanreedClientMessageType_ConnectionRequest,
					ConnectionRequest: &spanclientmsg.ConnectionRequest{
						ClientId: incomingMsg.ClientId,
					},
					AppData: parsedMsg.AppData,
				}

				sdhErr := p.clientStore.SetDestinationHandlerName(incomingMsg.ClientId, destinationHandler.Name)
				if sdhErr != nil {
					p.log.Error("Error setting client destination handler name", zap.Uint32("clientId", incomingMsg.ClientId), zap.Error(sdhErr))
					return sdhErr
				}
				destSendChannels.openConnectionsChannel <- handlers.OpenDestinationConnectionCommand{
					ClientId:             incomingMsg.ClientId,
					ConnectionRequestMsg: spanreedMessage,
					ConnectionString:     parsedMsg.ConnectionRequest.ConnectionString,
					RecvTime:             incomingMsg.RecvTime,
					ProcessTime:          dequeTimestamp,
				}

				return nil
			}
		}

		return &NoMatchingHandlerForConnectionString{
			ConnectionString: parsedMsg.ConnectionRequest.ConnectionString,
		}
	}()

	if err != nil {
		p.log.Error("Error attaching destination handler, attempting to close", zap.Uint32("clientId", incomingMsg.ClientId), zap.Error(err))
		p.kickClient(incomingMsg.ClientId, err)
	}
	return err
}

func (p *proxy) handleClientAppMessage(incomingMsg handlers.IncomingClientMessage, parsedMsg *clientmsg.ClientMessage) error {
	return p.sendDestinationMessage(incomingMsg.ClientId, incomingMsg.RecvTime, p.getNowTime(), &spanclientmsg.SpanreedClientMessage{
		MagicNumber: p.magicNumber,
		Version:     p.version,
		MessageType: spanclientmsg.SpanreedClientMessageType_AppMessage,
		AppData:     parsedMsg.AppData,
	})
}

func (p *proxy) sendDestinationMessage(clientId uint32, recvTime int64, deqTime int64, msg *spanclientmsg.SpanreedClientMessage) error {
	if !p.clientStore.HasClient(clientId) {
		return &internal.MissingClientIdError{
			Id: clientId,
		}
	}

	destName, destNameError := p.clientStore.GetDestinationHandlerName(clientId)
	if destNameError != nil {
		return destNameError
	}

	p.mut_outgoingDestinationMessageSendChannels.RLock()
	defer p.mut_outgoingDestinationMessageSendChannels.RUnlock()

	destSendChannels, has := p.outgoingDestinationMessageSendChannels[destName]
	if !has {
		return &MissingDestinationHandler{Name: destName}
	}

	destSendChannels.outgoingMessages <- handlers.OutgoingDestinationMessage{
		ClientId:    clientId,
		Message:     msg,
		RecvTime:    recvTime,
		ProcessTime: deqTime,
	}

	return nil
}

func (p *proxy) sendClientMessage(clientId uint32, recvTime int64, deqTime int64, msg *spandestintationmsg.SpanreedDestinationMessage) error {
	clientHandlerName, clientHandlerNameError := p.clientStore.GetClientHandlerName(clientId)
	if clientHandlerNameError != nil {
		return clientHandlerNameError
	}

	p.mut_outgoingClientMessageChannels.RLock()
	defer p.mut_outgoingClientMessageChannels.RUnlock()

	clientSendChannels, has := p.outgoingClientMessageChannels[clientHandlerName]
	if !has {
		return &MissingClientHandler{Name: clientHandlerName}
	}

	clientSendChannels.outgoingMessages <- handlers.OutgoingClientMessage{
		ClientId:    clientId,
		Message:     msg,
		RecvTime:    recvTime,
		ProcessTime: deqTime,
	}

	return nil
}

func (p *proxy) handleDestinationMessage(incomingMsg handlers.IncomingDestinationMessage) error {
	if !p.clientStore.HasClient(incomingMsg.ClientId) {
		return &internal.MissingClientIdError{
			Id: incomingMsg.ClientId,
		}
	}

	if incomingMsg.Message.MagicNumber != p.magicNumber || incomingMsg.Message.Version != p.version {
		return &errors.InvalidHeaderVersion{
			ExpectedMagicNumber: p.magicNumber,
			ExpectedVersion:     p.version,
			ActualMagicNumber:   incomingMsg.Message.MagicNumber,
			ActualVersion:       incomingMsg.Message.Version,
		}
	}

	switch incomingMsg.Message.MessageType {
	case destination.DestinationMessageType_ConnectionResponse:
		return p.handleDestinationConnectionResponseMessage(incomingMsg)
	case destination.DestinationMessageType_AppMessage:
		return p.handleDestinationAppMessage(incomingMsg)
	case destination.DestinationMessageType_NONE:
	default:
		// fallthrough
	}

	// TODO (sessamekesh): Handle them
	return nil
}

func (p *proxy) handleDestinationConnectionResponseMessage(incomingMsg handlers.IncomingDestinationMessage) error {
	processStartTimestamp := p.getNowTime()

	if incomingMsg.Message.ConnectionResponse == nil {
		return &errors.MissingFieldError{
			MessageName: "IncomingDestinationMessage::Message",
			FieldName:   "ConnectionResponse",
		}
	}

	if !incomingMsg.Message.ConnectionResponse.Verdict {
		// Kick the client after sending the message...
		p.kickClient(incomingMsg.ClientId, &errors.ConnectionRefusedByDestination{})
	}

	p.sendClientMessage(incomingMsg.ClientId, incomingMsg.RecvTime, processStartTimestamp,
		&spandestintationmsg.SpanreedDestinationMessage{
			MagicNumber: p.magicNumber,
			Version:     p.version,
			MessageType: spandestintationmsg.SpanreedDestinationMessageType_ConnectionResponse,
			ConnectionResponse: &spandestintationmsg.ConnectionResponse{
				Verdict:   incomingMsg.Message.ConnectionResponse.Verdict,
				IsTimeout: false,
			},
			AppData: incomingMsg.Message.AppData,
		})

	return nil
}

func (p *proxy) handleDestinationAppMessage(incomingMsg handlers.IncomingDestinationMessage) error {
	// TODO (sessamekesh)
	return nil
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

		destinationHandler.closeRequests <- handlers.DestinationCloseCommand{
			ClientId: clientId,
			Reason:   reason,
		}
	}()

	p.clientStore.RemoveClient(clientId)

	return nil
}
