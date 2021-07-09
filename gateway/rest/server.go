package rest

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"go.elastic.co/apm/module/apmhttp"
	"google.golang.org/grpc"
)

const (
	namedLogger = "gateway.restproxy"
)

// ProxyServer implement a rest server acting as a proxy to the grpc api
type ProxyServer struct {
	log *logging.Logger
	gateway.Config
	srv *http.Server
}

// NewProxyServer returns a new instance of the rest proxy server
func NewProxyServer(log *logging.Logger, config gateway.Config) *ProxyServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &ProxyServer{
		log:    log,
		Config: config,
		srv:    nil,
	}
}

// ReloadConf update the internal configuration of the server
func (s *ProxyServer) ReloadConf(cfg gateway.Config) {
	s.log.Info("reloading confioguration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	s.Config = cfg
}

// Start start the server
func (s *ProxyServer) Start() {
	logger := s.log

	logger.Info("Starting REST<>GRPC based API",
		logging.String("addr", s.REST.IP),
		logging.Int("port", s.REST.Port))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	restAddr := net.JoinHostPort(s.REST.IP, strconv.Itoa(s.REST.Port))
	grpcAddr := net.JoinHostPort(s.Node.IP, strconv.Itoa(s.Node.Port))
	jsonPB := &JSONPb{
		EmitDefaults: true,
		Indent:       "  ", // formatted json output
		OrigName:     false,
	}

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonPB),
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
	)

	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := protoapi.RegisterTradingServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}
	if err := protoapi.RegisterTradingDataServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}

	// CORS support
	handler := cors.Default().Handler(mux)
	handler = healthCheckMiddleware(handler)
	handler = gateway.RemoteAddrMiddleware(logger, handler)
	// Gzip encoding support
	handler = newGzipHandler(*logger, handler.(http.HandlerFunc))
	// Metric support
	handler = gateway.MetricCollectionMiddleware(logger, handler)

	// APM
	if s.Config.REST.APMEnabled {
		handler = apmhttp.Wrap(handler)
	}

	s.srv = &http.Server{
		Addr:    restAddr,
		Handler: handler,
	}

	// Start http server on port specified
	err := s.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Panic("Failure serving REST proxy API", logging.Error(err))
	}

}

// Stop stops the server
func (s *ProxyServer) Stop() {
	if s.srv != nil {
		s.log.Info("Stopping REST<>GRPC based API")

		if err := s.srv.Shutdown(context.Background()); err != nil {
			s.log.Error("Failed to stop REST<>GRPC based API cleanly",
				logging.Error(err))
		}
	}
}

func healthCheckMiddleware(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Write([]byte("ok"))
			w.WriteHeader(200)
			return
		}
		f.ServeHTTP(w, r)
	}
}
