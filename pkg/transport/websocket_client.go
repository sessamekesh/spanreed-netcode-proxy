package transport

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sessamekesh/spanreed-netcode-proxy/internal"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	clientmsg "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/client"
	spanreeddestination "github.com/sessamekesh/spanreed-netcode-proxy/pkg/message/spanreed_destination"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"
)

type wsConnectionChannels struct {
	OutgoingMessages chan<- handlers.OutgoingClientMessage
	CloseRequest     chan<- handlers.ClientCloseCommand
}

type websocketSpanreedClient struct {
	upgrader *websocket.Upgrader

	params WebsocketSpanreedClientParams

	clientMessageSerializer   clientmsg.ClientMessageSerializer
	spanreedMessageSerializer spanreeddestination.SpanreedDestinationMessageSerializer

	proxyConnection *handlers.ClientMessageHandler

	mut_connections sync.RWMutex
	connections     map[uint32]*wsConnectionChannels

	log       *zap.Logger
	stringGen *utils.RandomStringGenerator
}

type WebsocketSpanreedClientParams struct {
	ListenAddress    string
	ListenEndpoint   string
	AllowAllHosts    bool
	AllowlistedHosts []string
	DenylistedHosts  []string

	MaxReadMessageSize int64

	MagicNumber uint32
	Version     uint8

	Logger *zap.Logger
}

func checkOrigin(r *http.Request, params WebsocketSpanreedClientParams) bool {
	origin := r.Header.Get("Origin")
	if utils.Contains(origin, params.DenylistedHosts) {
		return false
	}

	if params.AllowAllHosts {
		return true
	}

	return utils.Contains(origin, params.AllowlistedHosts)
}

func CreateWebsocketHandler(proxyConnection *handlers.ClientMessageHandler, params WebsocketSpanreedClientParams) (*websocketSpanreedClient, error) {
	// TODO (sessamekesh): Validation that necessary parameters exist, if any.
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	return &websocketSpanreedClient{
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return checkOrigin(r, params)
			},
			// TODO (sessamekesh): read/write buffer size params
		},
		params: params,
		clientMessageSerializer: clientmsg.ClientMessageSerializer{
			MagicNumber: params.MagicNumber,
			Version:     params.Version,
		},
		spanreedMessageSerializer: spanreeddestination.SpanreedDestinationMessageSerializer{
			MagicNumber: params.MagicNumber,
			Version:     params.Version,
		},
		proxyConnection: proxyConnection,

		mut_connections: sync.RWMutex{},
		connections:     make(map[uint32]*wsConnectionChannels),

		log:       logger.With(zap.String("handler", "WebSocket")),
		stringGen: utils.CreateRandomstringGenerator(time.Now().UnixMicro()),
	}, nil
}

func (ws *websocketSpanreedClient) sendSpanreedMessage(c *websocket.Conn, log *zap.Logger, msg *spanreeddestination.SpanreedDestinationMessage) {
	serializedMsg, serializeErr := ws.spanreedMessageSerializer.Serialize(msg)
	if serializeErr != nil {
		log.Error("Failed to serialize Spanreed client message", zap.Error(serializeErr))
	}

	sendErr := c.WriteMessage(websocket.BinaryMessage, serializedMsg)
	if sendErr != nil {
		log.Error("Failed to send Spanreed message to client", zap.Error(sendErr))
	}
}

func (ws *websocketSpanreedClient) onWsRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// TODO (sessamekesh): Logging with ID
	log := ws.log.With(
		zap.String("wsConnId", ws.stringGen.GetRandomString(6)),
	)

	log.Info("New WebSocket request")
	c, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// TODO (sessamekesh): Logging
		log.Error("Failed to upgrade HTTP request to WebSocket connection", zap.Error(err))
		return
	}

	defer c.Close()

	// TODO (sessamekesh): SetReadLimit

	clientId, err := ws.proxyConnection.GetNextClientId()
	if err != nil {
		ws.sendSpanreedMessage(c, log, &spanreeddestination.SpanreedDestinationMessage{
			MessageType: spanreeddestination.SpanreedDestinationMessageType_ConnectionResponse,
			ConnectionResponse: &spanreeddestination.ConnectionResponse{
				Verdict:      false,
				IsTimeout:    false,
				IsProxyError: true,
			},
		})

		log.Error("Failed to generate client ID for new connection", zap.Error(err))
		return
	}

	log = log.With(zap.Uint32("clientId", clientId))

	// TODO (sessamekesh): Buffer these channels based on config!
	outgoingMessages := make(chan handlers.OutgoingClientMessage, 16)
	closeRequest := make(chan handlers.ClientCloseCommand, 1)

	err = func() error {
		ws.mut_connections.Lock()
		defer ws.mut_connections.Unlock()

		if _, has := ws.connections[clientId]; has {
			return &internal.DuplicateClientIdError{
				Id: clientId,
			}
		}

		ws.connections[clientId] = &wsConnectionChannels{
			OutgoingMessages: outgoingMessages,
			CloseRequest:     closeRequest,
		}

		log.Debug("Added client to WebSocket handler connections map")

		return nil
	}()

	if err != nil {
		log.Error("Failed to establish Go channels for new client", zap.Error(err))
		ws.sendSpanreedMessage(c, log, &spanreeddestination.SpanreedDestinationMessage{
			MessageType: spanreeddestination.SpanreedDestinationMessageType_ConnectionResponse,
			ConnectionResponse: &spanreeddestination.ConnectionResponse{
				Verdict:      false,
				IsTimeout:    false,
				IsProxyError: true,
			},
		})
		return
	}

	defer func() {
		ws.mut_connections.Lock()
		defer ws.mut_connections.Unlock()
		delete(ws.connections, clientId)
		log.Debug("Removed client from WebSocket handler connections map")
	}()

	// Expect an auth message back, handle it specifically!
	select {
	case <-ctx.Done():
	case <-closeRequest:
		// TODO (sessamekesh): Handle graceful shutdown here (with auth section specific logic)
		ws.sendSpanreedMessage(c, log, &spanreeddestination.SpanreedDestinationMessage{
			MessageType: spanreeddestination.SpanreedDestinationMessageType_ConnectionResponse,
			ConnectionResponse: &spanreeddestination.ConnectionResponse{
				Verdict:      false,
				IsTimeout:    true,
				IsProxyError: false,
			},
		})
		return
	case authMsg := <-outgoingMessages:
		if authMsg.Message.MessageType != spanreeddestination.SpanreedDestinationMessageType_ConnectionResponse || authMsg.Message.ConnectionResponse == nil {
			log.Warn("Non-auth message received from Spanreed proxy, rejecting client connection")
			ws.sendSpanreedMessage(c, log, &spanreeddestination.SpanreedDestinationMessage{
				MessageType: spanreeddestination.SpanreedDestinationMessageType_ConnectionResponse,
				ConnectionResponse: &spanreeddestination.ConnectionResponse{
					Verdict:      false,
					IsTimeout:    false,
					IsProxyError: true,
				},
			})
			return
		}

		ws.sendSpanreedMessage(c, log, authMsg.Message)
		log.Info("Auth verdict received", zap.Bool("verdict", authMsg.Message.ConnectionResponse.Verdict))
		if !authMsg.Message.ConnectionResponse.Verdict {
			return
		}
	}

	//
	// Now we can enter the main loop! Easy enough.
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		log.Info("Starting Spanreed proxy listener goroutine")
		for {
			select {
			case <-ctx.Done():
			case <-closeRequest:
				log.Info("Spanreed proxy listener goroutine attempting graceful shutdown")
				// TODO (sessamekesh): Handle graceful shutdown here
				c.Close()
				return
			case logicalMessage := <-outgoingMessages:
				log.Info("Received message from proxy, forwarding to client", zap.Int("app_data_size", len(logicalMessage.Message.AppData)))
				ws.sendSpanreedMessage(c, log, logicalMessage.Message)
			}
		}
	}()

	wg.Add(1)
	go func() {
		expectedCloseErrors := []int{websocket.CloseNormalClosure, websocket.CloseMessage, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived}
		for {
			msgType, payload, msgErr := c.ReadMessage()
			if msgErr != nil {
				if websocket.IsCloseError(msgErr, expectedCloseErrors...) {
					closeError, ok := msgErr.(*websocket.CloseError)
					if ok {
						log.Info("Received close request, attempting graceful shutdown", zap.Int("closeCode", closeError.Code), zap.String("closeMsg", closeError.Text))
					} else {
						log.Info("Received close request from client, attempting graceful shutdown")
					}
					// TODO (sessamekesh): Graceful close (send to proxy)
					return
				}

				if websocket.IsUnexpectedCloseError(msgErr, expectedCloseErrors...) {
					log.Warn("Received unexpected close request from client, attempting graceful shutdown", zap.Error(msgErr))
					// TODO (sessamekesh): Logging and graceful shutdown for unexpected close
					return
				}

				// So hacky...
				if strings.Contains(msgErr.Error(), "use of closed network connection") {
					log.Info("Closing connection, probably from proxy-initiated 'close' call")
					return
				}

				log.Error("Received unexpected WebSocket error on message read", zap.Error(msgErr))
				// TODO (sessamekesh): Handling for all around unexpected error
				return
			}

			if msgType != websocket.BinaryMessage {
				log.Info("Received non-binary message, ignoring", zap.Int("size", len(payload)))
				continue
			}

			parsedMsg, parsedMsgErr := ws.clientMessageSerializer.Parse(payload)
			if parsedMsgErr != nil {
				log.Info("Failed to parse message from client", zap.Error(parsedMsgErr))
				// TODO (sessamekesh): Log error, maybe send a message back to the client?
				continue
			}

			ws.proxyConnection.IncomingClientMessageChannel <- handlers.IncomingClientMessage{
				ClientId: clientId,
				RecvTime: ws.proxyConnection.GetNowTimestamp(),
				Message:  parsedMsg,
			}
		}
	}()

	wg.Wait()
}

func (ws *websocketSpanreedClient) Start(ctx context.Context) error {
	// TODO (sessamekesh): Create mux, attach listener, listen on HTTP at given location. Start goroutines for errything.
	mux := http.NewServeMux()
	mux.HandleFunc(ws.params.ListenEndpoint, func(w http.ResponseWriter, r *http.Request) {
		ws.onWsRequest(ctx, w, r)
	})

	server := &http.Server{
		Addr:    ws.params.ListenAddress,
		Handler: mux,
		// TODO (sessamekesh): Additional HTTP server config here
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		// TODO (sessamekesh): Logging for server start
		ws.log.Sugar().Infof("Starting WebSocket server at %s", ws.params.ListenAddress)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			ws.log.Error("Unexpected WebSocket server close!", zap.Error(err))
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()
		ws.log.Info("Attempting to trigger shutdown of WebSocket server")

		if err := server.Shutdown(shutdownCtx); err != nil {
			ws.log.Error("Failed to gracefully shut down WebSocket server", zap.Error(err))
			return
		}
		ws.log.Info("Successfully shutdown WebSocket server")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		ws.log.Info("Starting WebSocket proxy message goroutine loop")

		for {
			select {
			case <-ctx.Done():
				return
			case closeRequest := <-ws.proxyConnection.CloseRequests:
				ws.handleProxyCloseRequest(closeRequest)
			case msgRequest := <-ws.proxyConnection.OutgoingClientMessageChannel:
				ws.handleOutgoingMessageRequest(msgRequest)
			}
		}

		// TODO (sessamekesh): Flush channels and gracefully exit?
	}()

	wg.Wait()

	ws.log.Info("All WebSocket server goroutines finished. Exiting gracefully!")
	return nil
}

func (ws *websocketSpanreedClient) handleProxyCloseRequest(closeRequest handlers.ClientCloseCommand) {
	ws.mut_connections.RLock()
	defer ws.mut_connections.RUnlock()

	ws.log.Info("Received ProxyCloseRequest", zap.Uint32("clientId", closeRequest.ClientId))

	route, has := ws.connections[closeRequest.ClientId]
	if !has {
		ws.log.Error("WebSocket server missing client ID", zap.Uint32("clientId", closeRequest.ClientId))
		return
	}

	route.CloseRequest <- closeRequest
}

func (ws *websocketSpanreedClient) handleOutgoingMessageRequest(msgRequest handlers.OutgoingClientMessage) {
	ws.mut_connections.RLock()
	defer ws.mut_connections.RUnlock()

	ws.log.Debug("Received outgoing message request", zap.Uint32("clientId", msgRequest.ClientId))

	route, has := ws.connections[msgRequest.ClientId]
	if !has {
		// TODO (sessamekesh): Log problem
		ws.log.Error("WebSocket server missing client ID", zap.Uint32("clientId", msgRequest.ClientId))
		return
	}

	route.OutgoingMessages <- msgRequest
}
