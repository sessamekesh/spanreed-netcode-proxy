package transport

import (
	"context"
	"fmt"
	"sync"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/sessamekesh/spanreed-netcode-proxy/internal"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/transport/SpanreedMessage"
	"go.uber.org/zap"
)

type clientConnectionChannels struct {
	IsConnected      bool
	OutgoingMessages chan<- handlers.ClientMessage
	CloseRequest     chan<- handlers.ClientCloseCommand
	Verdict          chan<- handlers.OpenClientConnectionVerdict

	TransportOutgoingData chan<- []byte
}

type ClientConnectionRouterParams struct {
	IncomingMessageQueueLength uint32
	OutgoingMessageQueueLength uint32
}

type clientConnectionRouter struct {
	proxyConnection *handlers.ClientMessageHandler

	mut_connections sync.RWMutex
	connections     map[uint32]*clientConnectionChannels
	params          ClientConnectionRouterParams

	log *zap.Logger
}

type SingleClientTransportChannels struct {
	ClientId         uint32
	OutgoingMessages <-chan []byte
	IncomingMessages chan<- []byte

	ClientInitiatedClose chan<- bool
	ProxyInitiatedClose  <-chan bool
}

func CreateClientConnectionRouter(proxyConnection *handlers.ClientMessageHandler, params ClientConnectionRouterParams, logger *zap.Logger) (*clientConnectionRouter, error) {
	log := logger
	if log == nil {
		log = zap.Must(zap.NewDevelopment())
	}

	if params.IncomingMessageQueueLength == 0 {
		params.IncomingMessageQueueLength = 16
	}
	if params.OutgoingMessageQueueLength == 0 {
		params.OutgoingMessageQueueLength = 16
	}

	return &clientConnectionRouter{
		proxyConnection: proxyConnection,
		mut_connections: sync.RWMutex{},
		connections:     make(map[uint32]*clientConnectionChannels),
		log:             log.With(zap.String("handlerBase", "ClientConnectionRouter")),
	}, nil
}

func safeParseConnectClientMessage(payload []byte) (msg *SpanreedMessage.ConnectClientMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			msg = nil
			err = fmt.Errorf("deformed message: %v", err)
		}
	}()

	// TODO (sessamekesh): Better validation, it's possible to get something invalid still
	// TODO (sessamekesh): Just... replace the message formats with custom ones, flatbuffers aren't really needed here.

	o := SpanreedMessage.GetRootAsConnectClientMessage(payload, 0)
	o.ConnectionString()
	o.AppDataBytes()
	return o, nil
}

func createVerdictMsg(verdict bool, reason string, appData []byte) []byte {
	b := flatbuffers.NewBuilder(64)
	var pAppData, pReasonString flatbuffers.UOffsetT
	if appData != nil {
		pAppData = b.CreateByteVector(appData)
	}
	if reason != "" {
		pReasonString = b.CreateString(reason)
	}

	SpanreedMessage.ConnectClientVerdictStart(b)
	SpanreedMessage.ConnectClientVerdictAddAccepted(b, verdict)
	if reason != "" {
		SpanreedMessage.ConnectClientVerdictAddErrorReason(b, pReasonString)
	}
	if appData != nil {
		SpanreedMessage.ConnectClientVerdictAddAppData(b, pAppData)
	}
	pConnectClientVerdict := SpanreedMessage.ConnectClientVerdictEnd(b)
	b.Finish(pConnectClientVerdict)

	return b.FinishedBytes()
}

func (r *clientConnectionRouter) OpenConnection(ctx context.Context) (*SingleClientTransportChannels, error) {
	clientId, err := r.proxyConnection.GetNextClientId()

	if err != nil {
		r.log.Error("Failed to generate client ID for new connection", zap.Error(err))
		return nil, err
	}

	log := r.log.With(zap.Uint32("clientId", clientId))

	log.Info("New client connection")

	outgoingMessages := make(chan handlers.ClientMessage, r.params.OutgoingMessageQueueLength)
	closeRequest := make(chan handlers.ClientCloseCommand, 1)
	verdict := make(chan handlers.OpenClientConnectionVerdict, 1)

	err = func() error {
		r.mut_connections.Lock()
		defer r.mut_connections.Unlock()

		if _, has := r.connections[clientId]; has {
			return &internal.DuplicateClientIdError{
				Id: clientId,
			}
		}

		r.connections[clientId] = &clientConnectionChannels{
			IsConnected:      false,
			OutgoingMessages: outgoingMessages,
			CloseRequest:     closeRequest,
			Verdict:          verdict,
		}

		log.Debug("Added client to handler connections map")
		return nil
	}()

	if err != nil {
		log.Error("Failed to establish Go channels for new client", zap.Error(err))
		return nil, err
	}

	outgoingPackets := make(chan []byte, r.params.OutgoingMessageQueueLength)
	incomingPackets := make(chan []byte, r.params.IncomingMessageQueueLength)
	clientCloseChannel := make(chan bool, 1)
	proxyCloseChannel := make(chan bool, 1)

	//
	// Routing
	go func() {
		//
		// Expect to receive an auth message and forward it...
		log.Info("Awaiting auth request...")
		select {
		case <-ctx.Done():
			log.Info("Cancelling auth request wait because of shutdown request")
			r.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
				ClientId: clientId,
				Reason:   "Proxy shutdown request",
			}
			return
		case <-clientCloseChannel:
			log.Info("Cancelling auth request wait because client connection closed")
			r.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
				ClientId: clientId,
				Reason:   "Proxy shutdown request",
			}
			return
		case <-closeRequest:
			log.Info("Cancelling auth request because proxy cancelled")
			r.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
				ClientId: clientId,
				Reason:   "Proxy shutdown request",
			}
			return
		case packet := <-incomingPackets:
			authMsg, err := safeParseConnectClientMessage(packet)
			if err != nil {
				log.Error("Error reading auth message", zap.Error(err))

				return
			}

			r.proxyConnection.OpenClientChannel <- handlers.OpenClientConnectionCommand{
				ClientId:         clientId,
				RecvTimestamp:    r.proxyConnection.GetNowTimestamp(),
				ConnectionString: string(authMsg.ConnectionString()),
				AppData:          authMsg.AppDataBytes(),
			}
		}

		//
		// Expect an auth message back from the proxy, handle it specifically
		select {
		case <-ctx.Done():
			log.Info("Cancelling auth response wait because of shutdown request")
			outgoingPackets <- createVerdictMsg(false, "Proxy shutdown", nil)
			proxyCloseChannel <- true
			return
		case <-clientCloseChannel:
			log.Info("Cancelling auth response wait because client connection closed")
			proxyCloseChannel <- true
			return
		case <-closeRequest:
			log.Info("Cancelling auth response because proxy cancelled")
			outgoingPackets <- createVerdictMsg(false, "Auth timeout", nil)
			proxyCloseChannel <- true
			return
		case verdictMsg := <-verdict:
			log.Info("Verdict received", zap.Bool("verdict", verdictMsg.Verdict))
			verdictString := ""
			if verdictMsg.Error != nil {
				verdictString = verdictMsg.Error.Error()
			}
			outgoingPackets <- createVerdictMsg(verdictMsg.Verdict, verdictString, verdictMsg.AppData)
			if !verdictMsg.Verdict {
				log.Warn("Client connection refused", zap.Error(verdictMsg.Error))
				proxyCloseChannel <- true
				return
			}
		}

		//
		// Mark connected
		func() {
			r.mut_connections.Lock()
			defer r.mut_connections.Unlock()
			channels, has := r.connections[clientId]
			if has {
				channels.IsConnected = true
			}
		}()

		//
		// Route proxy messages out
		go func() {
			log.Info("Starting Spanreed proxy listener goroutine")
			defer log.Info("Stopping Spanreed proxy listener goroutine")

			for {
				select {
				case <-ctx.Done():
				case <-closeRequest:
					log.Info("Spanreed proxy listener goroutine attempting graceful shutdown")
					proxyCloseChannel <- true
					return
				case logicalMessage := <-outgoingMessages:
					outgoingPackets <- logicalMessage.Data
				}
			}
		}()

		//
		// Route connection messages to proxy
		go func() {
			log.Info("Starting transport event listener goroutine")
			defer log.Info("Stopping transport event listener goroutine")
			for {
				select {
				case <-ctx.Done():
					r.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
						ClientId: clientId,
						Reason:   "Proxy shutdown request",
					}
					return
				case <-clientCloseChannel:
				case <-closeRequest:
					r.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
						ClientId: clientId,
						Reason:   "Close request received from transport",
					}
					return
				case packet := <-incomingPackets:
					r.proxyConnection.IncomingMessageChannel <- handlers.ClientMessage{
						ClientId:      clientId,
						RecvTimestamp: r.proxyConnection.GetNowTimestamp(),
						Data:          packet,
					}
				}
			}
		}()
	}()

	return &SingleClientTransportChannels{
		ClientId:             clientId,
		OutgoingMessages:     outgoingPackets,
		IncomingMessages:     incomingPackets,
		ClientInitiatedClose: clientCloseChannel,
		ProxyInitiatedClose:  proxyCloseChannel,
	}, nil
}

func (r *clientConnectionRouter) Remove(clientId uint32) {
	r.mut_connections.Lock()
	defer r.mut_connections.Unlock()
	delete(r.connections, clientId)
	r.log.Debug("Removed client from client connections map", zap.Uint32("clientId", clientId))
}

func (r *clientConnectionRouter) Start(ctx context.Context) error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		r.log.Info("Starting ClientConnectionRouter proxy listener")
		defer r.log.Info("Shutting down CilentConnectionRouter proxy listener")
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case closeRequest := <-r.proxyConnection.OutgoingCloseRequests:
				r.handleProxyCloseRequest(closeRequest)
			case msgRequest := <-r.proxyConnection.OutgoingMessageChannel:
				r.handleOutgoingMessageRequest(msgRequest)
			case incomingVerdict := <-r.proxyConnection.OpenClientVerdictChannel:
				r.handleIncomingVerdict(incomingVerdict)
			}
		}
	}()

	wg.Wait()

	// TODO (sessamekesh): Clean up connections here, graceful shutdown

	r.log.Info("All ClientConnectionRouter goroutines finished. Exiting gracefully")
	return nil
}

func (r *clientConnectionRouter) handleProxyCloseRequest(closeRequest handlers.ClientCloseCommand) {
	r.mut_connections.RLock()
	defer r.mut_connections.RUnlock()

	r.log.Info("Received ProxyCloseRequest", zap.Uint32("clientId", closeRequest.ClientId))

	route, has := r.connections[closeRequest.ClientId]
	if !has {
		r.log.Error("Missing connection with ClientID", zap.Uint32("clientId", closeRequest.ClientId))
		return
	}

	route.CloseRequest <- closeRequest
}

func (r *clientConnectionRouter) handleOutgoingMessageRequest(msgRequest handlers.ClientMessage) {
	r.mut_connections.RLock()
	defer r.mut_connections.RUnlock()

	route, has := r.connections[msgRequest.ClientId]
	if !has {
		r.log.Error("Cannot route message to missing client ID", zap.Uint32("clientId", msgRequest.ClientId))
		return
	}

	route.OutgoingMessages <- msgRequest
}

func (r *clientConnectionRouter) handleIncomingVerdict(verdictMsg handlers.OpenClientConnectionVerdict) {
	r.mut_connections.RLock()
	defer r.mut_connections.RUnlock()

	r.log.Info("Received incoming verdict", zap.Uint32("clientId", verdictMsg.ClientId), zap.Bool("verdict", verdictMsg.Verdict))

	route, has := r.connections[verdictMsg.ClientId]
	if !has {
		r.log.Error("Cannot route message to missing client ID", zap.Uint32("clientId", verdictMsg.ClientId))
		return
	}

	route.Verdict <- verdictMsg
}
