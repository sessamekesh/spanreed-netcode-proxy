package transport

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"
)

type webtransportSpanreedClient struct {
	proxyConnection        *handlers.ClientMessageHandler
	clientConnectionRouter *clientConnectionRouter

	log       *zap.Logger
	stringGen *utils.RandomStringGenerator

	params WebtransportSpanreedClientParams

	s *webtransport.Server
}

type WebtransportSpanreedClientParams struct {
	ListenAddress  string
	ListenEndpoint string

	Logger *zap.Logger

	CertPath string
	KeyPath  string

	AllowAllHosts    bool
	AllowlistedHosts []string
	DenylistedHosts  []string

	IncomingMessageQueueLength uint32
	OutgoingMessageQueueLength uint32
}

func CreateWebtransportHandler(proxyConnection *handlers.ClientMessageHandler, params WebtransportSpanreedClientParams) (*webtransportSpanreedClient, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	router, err := CreateClientConnectionRouter(proxyConnection, ClientConnectionRouterParams{
		IncomingMessageQueueLength: params.IncomingMessageQueueLength,
		OutgoingMessageQueueLength: params.OutgoingMessageQueueLength,
	}, logger.With(zap.String("transport", "WebTransport")))
	if err != nil {
		return nil, err
	}

	return &webtransportSpanreedClient{
		proxyConnection:        proxyConnection,
		clientConnectionRouter: router,
		log:                    logger.With(zap.String("handler", "WebTransport")),
		params:                 params,
	}, nil
}

func (wt *webtransportSpanreedClient) onWtRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log := wt.log.With(zap.String("wtConnId", wt.stringGen.GetRandomString(6)))

	log.Info("New WebTransport request")

	session, sessionError := wt.s.Upgrade(w, r)
	if sessionError != nil {
		log.Warn("Failed to upgrade HTTP3 request to a WebTransport session", zap.Error(sessionError))
		w.WriteHeader(500)
		return
	}

	defer session.CloseWithError(0, "Proxy requested connection close")

	routeContext, routeCancel := context.WithCancel(ctx)
	defer routeCancel()

	clientRouter, crErr := wt.clientConnectionRouter.OpenConnection(routeContext)
	if crErr != nil {
		log.Error("Failed to establish client router for new client")
		return
	}
	defer wt.clientConnectionRouter.Remove(clientRouter.ClientId)

	log = log.With(zap.Uint32("clientId", clientRouter.ClientId))

	//
	// Main loop
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		log.Info("Starting WebTransport proxy listener goroutine")
		defer log.Info("Stopped WebSocket proxy listener goroutine")
		defer wg.Done()

		for {
			select {
			case <-session.Context().Done():
				routeCancel()
			case <-routeContext.Done():
			case <-clientRouter.ProxyInitiatedClose:
				log.Info("Proxy listener attempting graceful shutdown")
				session.CloseWithError(0, "Attempting graceful shutdown at request of proxy or destination")
				routeCancel()
				return
			case logicalMessage := <-clientRouter.OutgoingMessages:
				writeErr := session.SendDatagram(logicalMessage)
				if writeErr != nil {
					if cerr := session.Context().Err(); cerr != nil {
						routeCancel()
						return
					} else {
						log.Warn("Error writing to bidi stream", zap.Error(writeErr))
					}
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		log.Info("Starting WebTransport connection listener goroutine")
		defer log.Info("Stopped WebTransport connection listener goroutine")
		defer wg.Done()
		defer func() { clientRouter.ClientInitiatedClose <- true }()
		defer routeCancel()

		for {
			readBuffer, readErr := session.ReceiveDatagram(routeContext)
			if readErr != nil {
				log.Error("Unexpected read error", zap.Error(readErr))
				return
			}

			clientRouter.IncomingMessages <- readBuffer
		}
	}()

	wg.Wait()
}

func (wt *webtransportSpanreedClient) Start(ctx context.Context) error {
	certs, err := tls.LoadX509KeyPair(wt.params.CertPath, wt.params.KeyPath)
	if err != nil {
		wt.log.Error("Failed to load certificate pair", zap.Error(err))
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certs},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(wt.params.ListenEndpoint, func(w http.ResponseWriter, r *http.Request) {
		wt.log.Info("Request!")
		wt.onWtRequest(ctx, w, r)
	})

	wt.s = &webtransport.Server{
		H3: http3.Server{
			Addr:            wt.params.ListenAddress,
			TLSConfig:       tlsConfig,
			Handler:         mux,
			EnableDatagrams: true,
		},
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if utils.Contains(origin, wt.params.DenylistedHosts) {
				return false
			}

			if wt.params.AllowAllHosts {
				return true
			}

			return utils.Contains(origin, wt.params.AllowlistedHosts)
		},
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		wt.log.Info("Starting WebTransport HTTP3 server!", zap.String("path", wt.params.ListenAddress))
		defer wt.log.Info("Shutdown WebTransport HTTP3 server")
		defer wg.Done()

		if err := wt.s.ListenAndServeTLS(wt.params.CertPath, wt.params.KeyPath); err != nil {
			wt.log.Error("Unexpected WebTransport server close!", zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := wt.clientConnectionRouter.Start(ctx); err != nil {
			wt.log.Error("Error on ClientConnectionRouter run", zap.Error(err))
		}
	}()

	wg.Wait()

	wt.log.Info("All WebTransport server goroutines finished. Exiting gracefully.")
	return nil
}
