package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sessamekesh/spanreed-netcode-proxy/internal"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/errors"
	clientmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/client"
	destmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/destination"
	spanclientmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/spanreed_client"
)

type MissingDestinationHandler struct {
	Name string
}

func (e *MissingDestinationHandler) Error() string {
	return fmt.Sprintf("Missing destination handler with name %s", e.Name)
}

type NoMatchingHandlerForConnectionString struct {
	ConnectionString string
}

func (e *NoMatchingHandlerForConnectionString) Error() string {
	return fmt.Sprintf("No destination handler can match connection string %s", e.ConnectionString[0:32])
}

type ClientMessageHandler struct {
	Name            string
	GetNextClientId func() uint32
	GetNowTimestamp func() int64

	IncomingClientMessageChannel chan<- IncomingClientMessage
	OutgoingClientMessageChannel <-chan OutgoingClientMessage

	ConnectionResponses <-chan ClientConnectionResponse
	CloseRequests       <-chan ClientCloseConnectionRequest
}

type DestinationMessageHandler struct {
	Name                              string
	GetNowTimestamp                   func() int64
	IncomingDestinationMessageChannel chan<- IncomingDestinationMessage
	OutgoingDestinationMessageChannel <-chan OutgoingDestinationMessage

	matchConnectionString func(connStr string) bool
}

type clientHandlerChannels struct {
	outgoingMessages    chan<- OutgoingClientMessage
	connectionResponses chan<- ClientConnectionResponse
	closeRequests       chan<- ClientCloseConnectionRequest
}

type destinationHandlerChannels struct {
	outgoingMessages chan<- OutgoingDestinationMessage
}

type proxy struct {
	config ProxyConfig

	clientStore *internal.ClientStore
	startTime   time.Time

	clientMessageSerializer         clientmsg.ClientMessageSerializer
	spanreedClientMessageSerializer spanclientmsg.SpanreedClientMessageSerializer
	destinationMessageSerializer    destmsg.DestinationMessageSerializer

	incomingClientMessageSendChannel chan<- IncomingClientMessage
	incomingClientMessageRecvChannel <-chan IncomingClientMessage

	incomingDestinationMessageSendChannel chan<- IncomingDestinationMessage
	incomingDestinationMessageRecvChannel <-chan IncomingDestinationMessage

	mut_outgoingClientMessageChannels sync.RWMutex
	outgoingClientMessageChannels     map[string]clientHandlerChannels

	mut_outgoingDestinationMessageSendChannels sync.RWMutex
	outgoingDestinationMessageSendChannels     map[string]destinationHandlerChannels

	mut_connectedDestinationHandlers sync.RWMutex
	connectedDestinationHandlers     []*DestinationMessageHandler
}

type ProxyConfig struct {
	MagicNumber uint32
	Version     uint8

	IncomingClientConnectionBufferLength int

	IncomingClientMessageBufferLength      int
	IncomingDestinationMessageBufferLength int

	OutgoingClientMessageBufferLength      int
	OutgoingDestinationMessageBufferLength int
}

func CreateProxy(config ProxyConfig) *proxy {
	incomingClientMessageBufferLength := 256
	incomingDestinationMessageBufferLength := 256

	if config.IncomingClientMessageBufferLength > 0 {
		incomingClientMessageBufferLength = config.IncomingClientMessageBufferLength
	}
	if config.IncomingDestinationMessageBufferLength > 0 {
		incomingDestinationMessageBufferLength = config.IncomingDestinationMessageBufferLength
	}

	incomingClientMessagesChannel := make(chan IncomingClientMessage, incomingClientMessageBufferLength)
	incomingDestinationMessagesChannel := make(chan IncomingDestinationMessage, incomingDestinationMessageBufferLength)

	var magicNumber uint32 = 0x52554259
	var version uint8 = 0
	if config.MagicNumber != 0 {
		magicNumber = config.MagicNumber
	}
	if config.Version > 0 {
		version = config.Version
	}

	return &proxy{
		config: config,

		clientStore: &internal.ClientStore{},
		startTime:   time.Now(),

		clientMessageSerializer: clientmsg.ClientMessageSerializer{
			MagicNumber: magicNumber,
			Version:     version,
		},
		spanreedClientMessageSerializer: spanclientmsg.SpanreedClientMessageSerializer{
			MagicNumber: magicNumber,
			Version:     version,
		},
		destinationMessageSerializer: destmsg.DestinationMessageSerializer{
			MagicNumber: magicNumber,
			Version:     version,
		},

		incomingClientMessageSendChannel: incomingClientMessagesChannel,
		incomingClientMessageRecvChannel: incomingClientMessagesChannel,

		incomingDestinationMessageSendChannel: incomingDestinationMessagesChannel,
		incomingDestinationMessageRecvChannel: incomingDestinationMessagesChannel,

		mut_outgoingClientMessageChannels: sync.RWMutex{},
		outgoingClientMessageChannels:     make(map[string]clientHandlerChannels),

		mut_outgoingDestinationMessageSendChannels: sync.RWMutex{},
		outgoingDestinationMessageSendChannels:     make(map[string]destinationHandlerChannels),

		mut_connectedDestinationHandlers: sync.RWMutex{},
		connectedDestinationHandlers:     []*DestinationMessageHandler{},
	}
}

func (p *proxy) getNowTime() int64 {
	return time.Since(p.startTime).Microseconds()
}

func (p *proxy) CreateClientMessageHandler(name string) (*ClientMessageHandler, error) {
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

	outgoingMessageChannel := make(chan OutgoingClientMessage, outgoingMessageChannelLength)
	connectionResponses := make(chan ClientConnectionResponse, incomingClientConnectionBufferLength)
	closeRequests := make(chan ClientCloseConnectionRequest, incomingClientConnectionBufferLength)

	p.outgoingClientMessageChannels[name] = clientHandlerChannels{
		outgoingMessages:    outgoingMessageChannel,
		connectionResponses: connectionResponses,
		closeRequests:       closeRequests,
	}

	return &ClientMessageHandler{
		Name:                         name,
		GetNextClientId:              p.getClientIdCbForMessageHandler(name),
		GetNowTimestamp:              p.getNowTime,
		IncomingClientMessageChannel: p.incomingClientMessageSendChannel,
		OutgoingClientMessageChannel: outgoingMessageChannel,
		ConnectionResponses:          connectionResponses,
		CloseRequests:                closeRequests,
	}, nil
}

func (p *proxy) getClientIdCbForMessageHandler(name string) func() uint32 {
	return func() uint32 {
		clientId := p.clientStore.GetNewClientId()

		// TODO (sessamekesh): Log this error
		p.clientStore.CreateClient(clientId, name, p.getNowTime())

		return clientId
	}
}

func (p *proxy) CreateDestinationMessageHandler(name string, matchConnectionStringFn func(string) bool) (*DestinationMessageHandler, error) {
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

	outgoingMessageChannel := make(chan OutgoingDestinationMessage, outgoingMessageChannelLength)

	p.mut_connectedDestinationHandlers.Lock()
	defer p.mut_connectedDestinationHandlers.Unlock()

	newHandler := &DestinationMessageHandler{
		matchConnectionString:             matchConnectionStringFn,
		Name:                              name,
		GetNowTimestamp:                   p.getNowTime,
		IncomingDestinationMessageChannel: p.incomingDestinationMessageSendChannel,
		OutgoingDestinationMessageChannel: outgoingMessageChannel,
	}

	destinationHandlerChannels := destinationHandlerChannels{
		outgoingMessages: outgoingMessageChannel,
	}

	p.outgoingDestinationMessageSendChannels[name] = destinationHandlerChannels
	p.connectedDestinationHandlers = append(p.connectedDestinationHandlers, newHandler)

	return newHandler, nil
}

func (p *proxy) Start(ctx context.Context) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	//
	// Client message handling goroutine...
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				// TODO (sessamekesh): Logging
				return
			case clientMsg := <-p.incomingClientMessageRecvChannel:
				// TODO (sessamekesh): Logging
				err := p.handleClientMessage(clientMsg)
				if err == nil {
					p.clientStore.SetClientRecvTimestamp(clientMsg.ClientId, p.getNowTime())
				}
			}
		}
	}()

	//
	// Server message handling goroutine...
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				// TODO (sessamekesh): Logging
				return
			case destinationMsg := <-p.incomingDestinationMessageRecvChannel:
				// TODO (sessamekesh): Logging
				err := p.handleDestinationMessage(destinationMsg)
				if err == nil {
					p.clientStore.SetDestinationRecvTimestamp(destinationMsg.ClientId, p.getNowTime())
				}
			}
		}
	}()

	wg.Wait()
}

func (p *proxy) handleClientMessage(incomingMsg IncomingClientMessage) error {
	// TODO (sessamekesh)
	parsedMsg, parsedMsgErr := p.clientMessageSerializer.Parse(incomingMsg.RawMessage)
	if parsedMsgErr != nil {
		return parsedMsgErr
	}

	if !p.clientStore.HasClient(incomingMsg.ClientId) {
		return &internal.MissingClientIdError{
			Id: incomingMsg.ClientId,
		}
	}

	switch parsedMsg.MessageType {
	case clientmsg.ClientMessageType_ConnectionRequest:
		return p.handleClientConnectionRequestMessage(incomingMsg, parsedMsg)
	case clientmsg.ClientMessageType_AppMessage:
		return p.handleClientAppMessage(incomingMsg, parsedMsg)
	case clientmsg.ClientMessageType_NONE:
	default:
		// fallthrough
	}

	return &errors.InvalidEnumValue{
		EnumName: "IncomingClientMessage::Message::MessageType",
		IntValue: 0xff,
	}
}

func (p *proxy) handleClientConnectionRequestMessage(incomingMsg IncomingClientMessage, parsedMsg *clientmsg.ClientMessage) error {
	dequeTimestamp := p.getNowTime()

	if parsedMsg.ConnectionRequest == nil {
		return &errors.MissingFieldError{
			MessageName: "IncomingClientMessage::Message",
			FieldName:   "ConnectionRequest",
		}
	}

	err := func() error {
		p.mut_connectedDestinationHandlers.RLock()
		defer p.mut_connectedDestinationHandlers.RUnlock()
		for _, destinationHandler := range p.connectedDestinationHandlers {
			if destinationHandler.matchConnectionString(parsedMsg.ConnectionRequest.ConnectionString) {
				p.mut_outgoingDestinationMessageSendChannels.RLock()
				defer p.mut_outgoingDestinationMessageSendChannels.RUnlock()

				destSendChannels, has := p.outgoingDestinationMessageSendChannels[destinationHandler.Name]
				if !has {
					return &MissingDestinationHandler{Name: destinationHandler.Name}
				}

				spanreedMessage := spanclientmsg.SpanreedClientMessage{
					MessageType: spanclientmsg.SpanreedClientMessageType_ConnectionRequest,
					ConnectionRequest: &spanclientmsg.ConnectionRequest{
						ClientId: incomingMsg.ClientId,
					},
					AppData: parsedMsg.AppData,
				}
				rawSpanreedMessage, serializeError := p.spanreedClientMessageSerializer.SerializeMessage(&spanreedMessage)
				if serializeError != nil {
					return serializeError
				}

				msg := OutgoingDestinationMessage{
					ClientId:   incomingMsg.ClientId,
					RecvTime:   incomingMsg.RecvTime,
					DeqTime:    dequeTimestamp,
					RawMessage: rawSpanreedMessage,
				}
				// TODO (sessamekesh): Log error
				p.clientStore.SetDestinationHandlerName(incomingMsg.ClientId, destinationHandler.Name)
				destSendChannels.outgoingMessages <- msg

				return nil
			}
		}

		return &NoMatchingHandlerForConnectionString{
			ConnectionString: parsedMsg.ConnectionRequest.ConnectionString,
		}
	}()

	if err != nil {
		clientName, clientNameErr := p.clientStore.GetClientHandlerName(incomingMsg.ClientId)
		if clientNameErr != nil {
			// TODO (sessamekesh): clientNameErr here!
			return err
		}

		p.mut_outgoingClientMessageChannels.RLock()
		defer p.mut_outgoingClientMessageChannels.RUnlock()
		clientChannels, has := p.outgoingClientMessageChannels[clientName]
		if !has {
			// TODO (sessamekesh): Log the error here too...
			return err
		}

		clientChannels.closeRequests <- ClientCloseConnectionRequest{
			ClientId:        incomingMsg.ClientId,
			RecvTime:        incomingMsg.RecvTime,
			DeqTime:         dequeTimestamp,
			UnderlyingError: err,
		}
	}
	return err
}

func (p *proxy) handleClientAppMessage(incomingMsg IncomingClientMessage, parsedMsg *clientmsg.ClientMessage) error {
	return p.sendDestinationMessage(incomingMsg.ClientId, incomingMsg.RecvTime, p.getNowTime(), &spanclientmsg.SpanreedClientMessage{
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

	rawSpanreedMessage, serializeError := p.spanreedClientMessageSerializer.SerializeMessage(msg)
	if serializeError != nil {
		return serializeError
	}

	p.mut_outgoingDestinationMessageSendChannels.RLock()
	defer p.mut_outgoingDestinationMessageSendChannels.RUnlock()

	destSendChannels, has := p.outgoingDestinationMessageSendChannels[destName]
	if !has {
		return &MissingDestinationHandler{Name: destName}
	}

	destSendChannels.outgoingMessages <- OutgoingDestinationMessage{
		ClientId:   clientId,
		RawMessage: rawSpanreedMessage,
		RecvTime:   recvTime,
		DeqTime:    deqTime,
	}

	return nil
}

func (p *proxy) handleDestinationMessage(_ IncomingDestinationMessage) error {
	// TODO (sessamekesh): Handle them
	return nil
}
