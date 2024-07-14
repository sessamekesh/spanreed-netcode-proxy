package transport

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/adriancable/webtransport-go"
	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/handlers"
	utils "github.com/sessamekesh/spanreed-netcode-proxy/pkg/util"
	"go.uber.org/zap"
)

type wtConnectionChannels struct {
	IsConnected      bool
	OutgoingMessages chan<- handlers.ClientMessage
	CloseRequest     chan<- handlers.ClientCloseCommand
	Verdict          chan<- handlers.OpenClientConnectionVerdict
}

type WebTransportSpanreedClientParams struct {
	TLSCertPath string
	TLSKeyPath  string

	ListenAddress    string
	ListenEndpoint   string
	AllowAllHosts    bool
	AllowlistedHosts []string
	DenylistedHosts  []string

	MaxReadMessageSize int64

	Logger *zap.Logger
}

type wtSpanreedClient struct {
	params WebTransportSpanreedClientParams

	log       *zap.Logger
	stringGen *utils.RandomStringGenerator

	proxyConnection *handlers.ClientMessageHandler
}

func checkWtOrigin(r *http.Request, params WebTransportSpanreedClientParams) bool {
	origin := r.Header.Get("Origin")
	if utils.Contains(origin, params.DenylistedHosts) {
		return false
	}

	if params.AllowAllHosts {
		return true
	}

	return utils.Contains(origin, params.AllowlistedHosts)
}

func CreateWebTransportHandler(proxyConnection *handlers.ClientMessageHandler, params WebTransportSpanreedClientParams) (*wtSpanreedClient, error) {
	logger := params.Logger
	if logger == nil {
		logger = zap.Must(zap.NewDevelopment())
	}

	return nil, nil
}

func (wt *wtSpanreedClient) onWtRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log := wt.log.With(zap.String("wtConnId", wt.stringGen.GetRandomString(6)))

	if !checkWtOrigin(r, wt.params) {
		log.Info("Denied request because of denied origin")
		w.WriteHeader(500)
		return
	}

	session := r.Body.(*webtransport.Session)
	if session == nil {
		log.Warn("No WebTransport session could be extracted, rejecting")
		w.WriteHeader(500)
		return
	}

	session.AcceptSession()
	defer session.CloseSession()

	// TODO (sessamekesh): Set deadline here! If WT lib doesn't support it, do it yourself.
	// session.SetDeadline()

	bidiStream, err := session.AcceptStream()
	if err != nil {
		log.Error("Error accepting bidirectional stream", zap.Error(err))
		return
	}

	// TODO (sessamekesh): Set read deadline

	for {
		buf := make([]byte, 1024) // TODO (sessamekesh): Configurable buffer size
		n, err := bidiStream.Read(buf)
		if err != nil {
			log.Error("Unexpected WebTransport read error", zap.Error(err))
			break
		}

		log.Info("Read from bidi stream!", zap.Int("size", n))
		// TODO (sessamekesh): Handle message
	}

	log.Info("New WebTransport request")
}

func (wt *wtSpanreedClient) Start(ctx context.Context) error {
	if wt.params.TLSCertPath == "" || wt.params.TLSKeyPath == "" {
		return errors.New("Missing TLS certification, cannot start HTTP3 server")
	}

	mux := http.NewServeMux()
	mux.HandleFunc(wt.params.ListenEndpoint, func(w http.ResponseWriter, r *http.Request) {
		wt.onWtRequest(ctx, w, r)
	})

	server := &webtransport.Server{
		ListenAddr: wt.params.ListenAddress,
		Handler:    mux,
		TLSCert:    webtransport.CertFile{Path: wt.params.TLSCertPath},
		TLSKey:     webtransport.CertFile{Path: wt.params.TLSKeyPath},
	}

	wg := sync.WaitGroup{}

	//
	// HTTP3 Server....
	wg.Add(1)
	go func() {
		defer wg.Done()

		wt.log.Sugar().Infof("Starting WebTransport server at %s", wt.params.ListenAddress)
		if err := server.Run(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			wt.log.Error("Unexpected WebTransport server close!", zap.Error(err))
		}
	}()

	//
	// Proxy goroutine loop...
	wg.Add(1)
	go func() {
		defer wg.Done()

		wt.log.Info("Starting WebTransport proxy message goroutine loop")

		for {
			select {
			case <-ctx.Done():
				return
				// TODO (sessamekesh): Handle different outgoing proxy messages here!
			}
		}
	}()

	wt.log.Info("All WebTransport server goroutines finished. Exiting gracefully!")
	return nil
}
