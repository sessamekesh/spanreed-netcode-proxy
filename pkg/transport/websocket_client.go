package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"
)

type websocketSpanreedClient struct {
	upgrader *websocket.Upgrader

	params WebsocketSpanreedClientParams

	proxyConnection        *handlers.ClientMessageHandler
	clientConnectionRouter *clientConnectionRouter

	log       *zap.Logger
	stringGen *utils.RandomStringGenerator
}

type WebsocketSpanreedClientParams struct {
	ListenAddress    string
	ListenEndpoint   string
	AllowAllHosts    bool
	AllowlistedHosts []string
	DenylistedHosts  []string

	CertPath string
	KeyPath  string

	Logger *zap.Logger

	IncomingMessageQueueLength uint32
	OutgoingMessageQueueLength uint32

	IncomingMessageReadLimit  uint32
	OutgoingMessageWriteLimit uint32

	HandshakeTimeoutMilliseconds uint32
}

type NonBinaryMessage struct{}

func (m *NonBinaryMessage) Error() string {
	return "Non binary message received"
}

func CreateWebsocketHandler(proxyConnection *handlers.ClientMessageHandler, params WebsocketSpanreedClientParams) (*websocketSpanreedClient, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	if params.IncomingMessageReadLimit == 0 {
		params.IncomingMessageReadLimit = 10240 // 10 KB
	}
	if params.OutgoingMessageWriteLimit == 0 {
		params.OutgoingMessageWriteLimit = 10240 // 10 KB
	}
	if params.HandshakeTimeoutMilliseconds == 0 {
		params.HandshakeTimeoutMilliseconds = 1500 // 1.5 seconds to establish handshake
	}

	router, err := CreateClientConnectionRouter(proxyConnection, ClientConnectionRouterParams{
		IncomingMessageQueueLength: params.IncomingMessageQueueLength,
		OutgoingMessageQueueLength: params.OutgoingMessageQueueLength,
	}, logger.With(zap.String("transport", "WebSocket")))
	if err != nil {
		return nil, err
	}

	return &websocketSpanreedClient{
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if utils.Contains(origin, params.DenylistedHosts) {
					return false
				}

				if params.AllowAllHosts {
					return true
				}

				return utils.Contains(origin, params.AllowlistedHosts)
			},
			ReadBufferSize:   int(params.IncomingMessageQueueLength),
			WriteBufferSize:  int(params.OutgoingMessageQueueLength),
			HandshakeTimeout: time.Duration(params.HandshakeTimeoutMilliseconds) * time.Millisecond,
		},
		params:          params,
		proxyConnection: proxyConnection,

		clientConnectionRouter: router,

		log:       logger.With(zap.String("handler", "WebSocket")),
		stringGen: utils.CreateRandomstringGenerator(time.Now().UnixMicro()),
	}, nil
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
	routeContext, routeCancel := context.WithCancel(ctx)
	defer routeCancel()

	clientRouter, err := ws.clientConnectionRouter.OpenConnection(routeContext)
	if err != nil {
		log.Error("Failed to establish client router for new client")
		return
	}
	defer ws.clientConnectionRouter.Remove(clientRouter.ClientId)

	log = log.With(zap.Uint32("clientId", clientRouter.ClientId))

	//
	// Now we can enter the main loop! Easy enough.
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Info("Starting WebSocket proxy listener goroutine")
		defer log.Info("Stopped WebSocket proxy listener goroutine")

		for {
			select {
			case <-ctx.Done():
			case <-clientRouter.ProxyInitiatedClose:
				log.Info("Proxy listener attempting graceful shutdown")
				c.Close()
				routeCancel()
				return
			case logicalMessage := <-clientRouter.OutgoingMessages:
				c.WriteMessage(websocket.BinaryMessage, logicalMessage)
			}
		}
	}()

	wg.Add(1)
	go func() {
		log.Info("Starting WebSocket connection listener goroutine")
		defer log.Info("Stopped WebSocket connection listener goroutine")
		defer wg.Done()
		defer func() { clientRouter.ClientInitiatedClose <- true }()
		defer routeCancel()

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
					return
				}

				if websocket.IsUnexpectedCloseError(msgErr, expectedCloseErrors...) {
					log.Warn("Received unexpected close request from client, attempting graceful shutdown", zap.Error(msgErr))
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

			clientRouter.IncomingMessages <- payload
		}
	}()

	wg.Wait()
}

func (ws *websocketSpanreedClient) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(ws.params.ListenEndpoint, func(w http.ResponseWriter, r *http.Request) {
		ws.onWsRequest(ctx, w, r)
	})

	tlsConfig, err := func() (*tls.Config, error) {
		if ws.params.CertPath == "" || ws.params.KeyPath == "" {
			return nil, nil
		}

		certs, certErr := tls.LoadX509KeyPair(ws.params.CertPath, ws.params.KeyPath)
		if certErr != nil {
			return nil, certErr
		}

		return &tls.Config{
			Certificates: []tls.Certificate{certs},
		}, nil
	}()
	if err != nil {
		ws.log.Error("Failed to load TLS cert from provided cert/key files", zap.Error(err))
		return err
	}

	server := &http.Server{
		Addr:      ws.params.ListenAddress,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		ws.log.Sugar().Infof("Starting WebSocket server at %s", ws.params.ListenAddress)
		if ws.params.KeyPath != "" && ws.params.CertPath != "" {
			if err := server.ListenAndServeTLS(ws.params.CertPath, ws.params.KeyPath); !errors.Is(err, http.ErrServerClosed) {
				ws.log.Error("Unexpected WebSocket server close!", zap.Error(err))
			}
		} else {
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				ws.log.Error("Unexpected WebSocket server close!", zap.Error(err))
			}
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

		err := ws.clientConnectionRouter.Start(ctx)
		if err != nil {
			ws.log.Error("Error on ClientConnectionRouter run", zap.Error(err))
		}
	}()

	wg.Wait()

	ws.log.Info("All WebSocket server goroutines finished. Exiting gracefully!")
	return nil
}
