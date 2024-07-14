package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/gorilla/websocket"
	"github.com/sessamekesh/spanreed-netcode-proxy/internal"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/transport/SpanreedMessage"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"
)

type wsConnectionChannels struct {
	IsConnected      bool
	OutgoingMessages chan<- handlers.ClientMessage
	CloseRequest     chan<- handlers.ClientCloseCommand
	Verdict          chan<- handlers.OpenClientConnectionVerdict
}

type websocketSpanreedClient struct {
	upgrader *websocket.Upgrader

	params WebsocketSpanreedClientParams

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

type NonBinaryMessage struct{}

func (m *NonBinaryMessage) Error() string {
	return "Non binary message received"
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
		params:          params,
		proxyConnection: proxyConnection,

		mut_connections: sync.RWMutex{},
		connections:     make(map[uint32]*wsConnectionChannels),

		log:       logger.With(zap.String("handler", "WebSocket")),
		stringGen: utils.CreateRandomstringGenerator(time.Now().UnixMicro()),
	}, nil
}

func safeParseConnectClientMessage(payload []byte) (msg *SpanreedMessage.ConnectClientMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			msg = nil
			err = fmt.Errorf("deformed message: %v", err)
		}
	}()

	return SpanreedMessage.GetRootAsConnectClientMessage(payload, 0), nil
}

func (ws *websocketSpanreedClient) readAuthMessage(c *websocket.Conn) (*SpanreedMessage.ConnectClientMessage, error) {
	msgType, payload, msgErr := c.ReadMessage()
	if msgErr != nil {
		return nil, msgErr
	}

	if msgType != websocket.BinaryMessage {
		return nil, &NonBinaryMessage{}
	}

	return safeParseConnectClientMessage(payload)
}

func (ws *websocketSpanreedClient) onWsRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log := ws.log.With(
		zap.String("wsConnId", ws.stringGen.GetRandomString(6)),
	)

	log.Info("New WebSocket request")
	c, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Failed to upgrade HTTP request to WebSocket connection", zap.Error(err))
		return
	}

	defer c.Close()

	// TODO (sessamekesh): SetReadLimit

	clientId, err := ws.proxyConnection.GetNextClientId()
	if err != nil {
		// TODO (sessamekesh): Send error response
		log.Error("Failed to generate client ID for new connection", zap.Error(err))
		return
	}

	log = log.With(zap.Uint32("clientId", clientId))

	// TODO (sessamekesh): Buffer these channels based on config!
	outgoingMessages := make(chan handlers.ClientMessage, 16)
	closeRequest := make(chan handlers.ClientCloseCommand, 1)
	verdict := make(chan handlers.OpenClientConnectionVerdict, 1)

	err = func() error {
		ws.mut_connections.Lock()
		defer ws.mut_connections.Unlock()

		if _, has := ws.connections[clientId]; has {
			return &internal.DuplicateClientIdError{
				Id: clientId,
			}
		}

		ws.connections[clientId] = &wsConnectionChannels{
			IsConnected:      false,
			OutgoingMessages: outgoingMessages,
			CloseRequest:     closeRequest,
			Verdict:          verdict,
		}

		log.Debug("Added client to WebSocket handler connections map")

		return nil
	}()

	if err != nil {
		log.Error("Failed to establish Go channels for new client", zap.Error(err))
		// TODO (sessamekesh): Send error back over to client
		return
	}

	defer func() {
		ws.mut_connections.Lock()
		defer ws.mut_connections.Unlock()
		delete(ws.connections, clientId)
		log.Debug("Removed client from WebSocket handler connections map")
	}()

	// TODO (sessamekesh): Read a single message from the client, expect it to be a connection request message
	authMsg, err := ws.readAuthMessage(c)
	if err != nil {
		log.Error("Error reading auth message", zap.Error(err))
		return
	}

	// TODO (sessamekesh): Make reading flatbuffer data safe! It could panic!!!
	ws.proxyConnection.OpenClientChannel <- handlers.OpenClientConnectionCommand{
		ClientId:         clientId,
		RecvTimestamp:    ws.proxyConnection.GetNowTimestamp(),
		ConnectionString: string(authMsg.ConnectionString()),
		AppData:          authMsg.AppDataBytes(),
	}

	// Expect an auth message back, handle it specifically!
	select {
	case <-ctx.Done():
		return
	case <-closeRequest:
		log.Warn("Auth timeout")
		b := flatbuffers.NewBuilder(64)
		pAuthTimeoutMsg := b.CreateString("Authorization timed out")
		SpanreedMessage.ConnectClientVerdictStart(b)
		SpanreedMessage.ConnectClientVerdictAddAccepted(b, false)
		SpanreedMessage.ConnectClientVerdictAddErrorReason(b, pAuthTimeoutMsg)
		msg := SpanreedMessage.ConnectClientVerdictEnd(b)
		b.Finish(msg)
		buf := b.FinishedBytes()
		c.WriteMessage(websocket.BinaryMessage, buf)
		// TODO (sessamekesh): Handle graceful shutdown here (with auth section specific logic)
		return
	case verdictMsg := <-verdict:
		b := flatbuffers.NewBuilder(64)
		SpanreedMessage.ConnectClientVerdictStart(b)
		SpanreedMessage.ConnectClientVerdictAddAccepted(b, verdictMsg.Verdict)
		if verdictMsg.Error != nil {
			SpanreedMessage.ConnectClientVerdictAddErrorReason(b, b.CreateString("Proxy error"))
		}
		if verdictMsg.AppData != nil {
			SpanreedMessage.ConnectClientVerdictAddAppData(b, b.CreateByteVector(verdictMsg.AppData))
		}
		msg := SpanreedMessage.ConnectClientVerdictEnd(b)
		b.Finish(msg)
		buf := b.FinishedBytes()
		c.WriteMessage(websocket.BinaryMessage, buf)

		if !verdictMsg.Verdict {
			log.Warn("Client connection refused", zap.Error(verdictMsg.Error))
			return
		}
	}

	// Mark connected
	func() {
		ws.mut_connections.RLock()
		defer ws.mut_connections.RUnlock()
		channels, has := ws.connections[clientId]
		if has {
			channels.IsConnected = true
		}
	}()

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
				// TODO (sessamekesh): Handle graceful shutdown here, send message back to client
				c.Close()
				return
			case logicalMessage := <-outgoingMessages:
				c.WriteMessage(websocket.BinaryMessage, logicalMessage.Data)
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
					ws.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
						ClientId: clientId,
						Reason:   "Close request received from websocket",
					}
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

			ws.proxyConnection.IncomingMessageChannel <- handlers.ClientMessage{
				ClientId:      clientId,
				RecvTimestamp: ws.proxyConnection.GetNowTimestamp(),
				Data:          payload,
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
			case closeRequest := <-ws.proxyConnection.OutgoingCloseRequests:
				ws.handleProxyCloseRequest(closeRequest)
			case msgRequest := <-ws.proxyConnection.OutgoingMessageChannel:
				ws.handleOutgoingMessageRequest(msgRequest)
			case incomingVerdict := <-ws.proxyConnection.OpenClientVerdictChannel:
				ws.handleIncomingVerdict(incomingVerdict)
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

func (ws *websocketSpanreedClient) handleOutgoingMessageRequest(msgRequest handlers.ClientMessage) {
	ws.mut_connections.RLock()
	defer ws.mut_connections.RUnlock()

	route, has := ws.connections[msgRequest.ClientId]
	if !has {
		ws.log.Error("WebSocket server missing client ID", zap.Uint32("clientId", msgRequest.ClientId))
		return
	}

	route.OutgoingMessages <- msgRequest
}

func (ws *websocketSpanreedClient) handleIncomingVerdict(verdictMsg handlers.OpenClientConnectionVerdict) {
	ws.mut_connections.RLock()
	defer ws.mut_connections.RUnlock()

	ws.log.Info("Received incoming verdict", zap.Uint32("clientId", verdictMsg.ClientId), zap.Bool("verdict", verdictMsg.Verdict))

	route, has := ws.connections[verdictMsg.ClientId]
	if !has {
		ws.log.Error("WebSocket server missing client ID", zap.Uint32("clientId", verdictMsg.ClientId))
		return
	}

	route.Verdict <- verdictMsg
}
