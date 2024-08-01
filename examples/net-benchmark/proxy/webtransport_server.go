package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"github.com/sessamekesh/spanreed-netcode-proxy/internal"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	"go.uber.org/zap"
)

type clientConnectionChannels struct {
	IsConnected      bool
	OutgoingMessages chan<- handlers.ClientMessage
	CloseRequest     chan<- handlers.ClientCloseCommand
	Verdict          chan<- handlers.OpenClientConnectionVerdict
}

type WebtransportBenchmarkClientParams struct {
	CertPath   string
	KeyPath    string
	Logger     *zap.Logger
	ServerAddr string
	ListenAddr string
}

type webtransportBenchmarkClient struct {
	proxyConnection *handlers.ClientMessageHandler

	mut_connections sync.RWMutex
	connections     map[uint32]*clientConnectionChannels

	log *zap.Logger

	s      *webtransport.Server
	params WebtransportBenchmarkClientParams
}

func CreateWebtransportBenchmarkHandler(proxyConnection *handlers.ClientMessageHandler, params WebtransportBenchmarkClientParams) (*webtransportBenchmarkClient, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	if params.CertPath == "" || params.KeyPath == "" {
		return nil, errors.New("Missing cert or key path, cannot create HTTP3 server")
	}

	return &webtransportBenchmarkClient{
		proxyConnection: proxyConnection,
		mut_connections: sync.RWMutex{},
		connections:     make(map[uint32]*clientConnectionChannels),
		log:             logger.With(zap.String("handler", "WebTransport")),
		params:          params,
	}, nil
}

func (wt *webtransportBenchmarkClient) onWtRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	wt.log.Info("New WebTransport request")
	defer wt.log.Info("Closing WebTransport request")

	session, sessionError := wt.s.Upgrade(w, r)
	if sessionError != nil {
		wt.log.Error("Failed to upgrade HTTP3 request to a WebTransport session", zap.Error(sessionError))
		w.WriteHeader(500)
		return
	}

	defer session.CloseWithError(0, "Proxy requested connection close")

	routeContext, routeCancel := context.WithCancel(ctx)
	defer routeCancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	clientId, clientIdErr := wt.proxyConnection.GetNextClientId()
	if clientIdErr != nil {
		wt.log.Error("Could not allocate client ID", zap.Error(clientIdErr))
		return
	}

	log := wt.log.With(zap.Uint32("clientId", clientId))

	outgoingMessages := make(chan handlers.ClientMessage, 64)
	closeRequest := make(chan handlers.ClientCloseCommand, 1)
	verdict := make(chan handlers.OpenClientConnectionVerdict, 1)

	if openConnectionErr := func() error {
		wt.mut_connections.Lock()
		defer wt.mut_connections.Unlock()

		if _, has := wt.connections[clientId]; has {
			return &internal.DuplicateClientIdError{
				Id: clientId,
			}
		}

		wt.connections[clientId] = &clientConnectionChannels{
			IsConnected:      false,
			OutgoingMessages: outgoingMessages,
			CloseRequest:     closeRequest,
			Verdict:          verdict,
		}
		return nil
	}(); openConnectionErr != nil {
		log.Error("Failed to associate client with new message channels")
		return
	}

	//
	// Incoming Client Messages (start before verdict handling)
	go func() {
		defer wg.Done()

		log.Info("Starting WebTransport TRANSPORT listener goroutine")
		defer log.Info("Stopping WebTransport TRANSPORT listener goroutine")

		for {
			readBuffer, readErr := session.ReceiveDatagram(routeContext)
			if readErr != nil {
				log.Error("Unexpected read error", zap.Error(readErr))
				return
			}

			parsedMsgType := GetClientMessageType(readBuffer)
			switch parsedMsgType {
			case ClientMessageType_Unknown:
				log.Warn("Invalid message type from client")
				return
			case ClientMessageType_ConnectClient:
				wt.proxyConnection.OpenClientChannel <- handlers.OpenClientConnectionCommand{
					ClientId:         clientId,
					ConnectionString: wt.params.ServerAddr,
					AppData:          readBuffer,
				}
			case ClientMessageType_DisconnectClient:
				log.Info("Shutting down connection at client request")
				routeCancel()
				wt.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
					ClientId: clientId,
					AppData:  readBuffer,
				}
				return
			case ClientMessageType_Ping:
				SetPingRecvTimestamp(readBuffer, uint64(wt.proxyConnection.GetNowTimestamp()))
				fallthrough
			default:
				wt.proxyConnection.IncomingMessageChannel <- handlers.ClientMessage{
					ClientId: clientId,
					Data:     readBuffer,
				}
			}
		}
	}()

	// Expect an auth message back from proxy, handle it specifically
	select {
	case <-routeContext.Done():
		log.Info("Cancelling auth response wait because of explicit shutdown request")
		wt.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
			ClientId: clientId,
		}
		return
	case <-closeRequest:
		log.Info("Cancelling auth response wait because of proxy shutdown request")
		return
	case verdictMsg := <-verdict:
		log.Info("Verdict received", zap.Bool("verdict", verdictMsg.Verdict))
		session.SendDatagram(verdictMsg.AppData)
	}

	// Mark connected and start proxy message forwarding loop!
	func() {
		wt.mut_connections.Lock()
		defer wt.mut_connections.Unlock()

		channels, has := wt.connections[clientId]
		if has {
			channels.IsConnected = true
		}
	}()

	//
	// Proxy message forwarding loop
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Info("Starting WebTransport PROXY listener goroutine")
		defer log.Info("Stopping WebTransport PROXY listener goroutine")

		for {
			select {
			case <-routeContext.Done():
				wt.proxyConnection.IncomingCloseRequests <- handlers.ClientCloseCommand{
					ClientId: clientId,
				}
				return
			case <-closeRequest:
				return
			case msg := <-outgoingMessages:
				msgType := GetDestinationMessageType(msg.Data)
				if msgType == DestinationMessageType_Pong {
					SetPongForwardTimestamp(msg.Data, uint64(wt.proxyConnection.GetNowTimestamp()))
				}
				session.SendDatagram(msg.Data)
			}
		}
	}()

	wg.Wait()
}

func (wt *webtransportBenchmarkClient) Start(ctx context.Context) error {
	certs, err := tls.LoadX509KeyPair(wt.params.CertPath, wt.params.KeyPath)
	if err != nil {
		wt.log.Error("Failed to load certificate pair", zap.Error(err))
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certs},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		wt.onWtRequest(ctx, w, r)
	})

	// TODO (sessamekesh): Add routing goroutines

	wt.s = &webtransport.Server{
		H3: http3.Server{
			Addr:            wt.params.ListenAddr,
			TLSConfig:       tlsConfig,
			Handler:         mux,
			EnableDatagrams: true,
		},
	}

	wg := sync.WaitGroup{}

	//
	// Proxy routing goroutines
	wg.Add(1)
	go func() {
		defer wg.Done()

		wt.log.Info("Starting top-level proxy listener goroutine")
		defer wt.log.Info("Stopping top-level proxy listener goroutine")

		for {
			select {
			case <-ctx.Done():
				return
			case closeRequest := <-wt.proxyConnection.OutgoingCloseRequests:
				wt.handleProxyCloseRequest(closeRequest)
			case verdictMsg := <-wt.proxyConnection.OpenClientVerdictChannel:
				wt.handleProxyVerdict(verdictMsg)
			case msg := <-wt.proxyConnection.OutgoingMessageChannel:
				wt.handleProxyMessageRequest(msg)
			}
		}
	}()

	//
	// Server goroutines (context shutdown listener + ListenAndServe goroutine)
	wg.Add(1)
	go func() {
		defer wg.Done()

		wt.log.Info("Starting WT HTTP3 server on port 30100")
		defer wt.log.Info("Shutdown WT HTTP3 server")

		if err := wt.s.ListenAndServeTLS(wt.params.CertPath, wt.params.KeyPath); err != nil {
			wt.log.Error("Unexpected WT server close", zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		if closeErr := wt.s.Close(); closeErr != nil {
			wt.log.Error("Error shutting down WT HTTP3 server", zap.Error(closeErr))
		}
	}()

	wg.Wait()

	wt.log.Info("All WT server goroutines finished, exiting gracefully")
	return nil
}

func (wt *webtransportBenchmarkClient) handleProxyCloseRequest(closeRequest handlers.ClientCloseCommand) {
	wt.log.Info("Received ProxyCloseRequset", zap.Uint32("clientId", closeRequest.ClientId))

	wt.mut_connections.Lock()
	defer wt.mut_connections.Unlock()

	route, has := wt.connections[closeRequest.ClientId]
	if !has {
		wt.log.Warn("Missing connection with ClientID", zap.Uint32("clientId", closeRequest.ClientId))
		return
	}

	route.CloseRequest <- closeRequest
}

func (wt *webtransportBenchmarkClient) handleProxyMessageRequest(msgRequest handlers.ClientMessage) {
	wt.log.Info("Received ProxyCloseRequset", zap.Uint32("clientId", msgRequest.ClientId))

	wt.mut_connections.Lock()
	defer wt.mut_connections.Unlock()

	route, has := wt.connections[msgRequest.ClientId]
	if !has {
		wt.log.Warn("Missing connection with ClientID", zap.Uint32("clientId", msgRequest.ClientId))
		return
	}

	route.OutgoingMessages <- msgRequest
}

func (wt *webtransportBenchmarkClient) handleProxyVerdict(verdictMsg handlers.OpenClientConnectionVerdict) {
	wt.log.Info("Received ProxyCloseRequset", zap.Uint32("clientId", verdictMsg.ClientId))

	wt.mut_connections.Lock()
	defer wt.mut_connections.Unlock()

	route, has := wt.connections[verdictMsg.ClientId]
	if !has {
		wt.log.Warn("Missing connection with ClientID", zap.Uint32("clientId", verdictMsg.ClientId))
		return
	}

	route.Verdict <- verdictMsg
}
