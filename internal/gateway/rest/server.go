package rest

import (
	"context"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/internal/gateway"
	"code.vegaprotocol.io/vega/internal/logging"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

const (
	namedLogger = "api.restproxy"
)

type restProxyServer struct {
	log *logging.Logger
	gateway.Config
	srv *http.Server
}

func NewRestProxyServer(log *logging.Logger, config gateway.Config) *restProxyServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &restProxyServer{
		log:    log,
		Config: config,
		srv:    nil,
	}
}

func (s *restProxyServer) ReloadConf(cfg gateway.Config) {
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

func (s *restProxyServer) Start() {
	logger := s.log

	logger.Info("Starting REST<>GRPC based API",
		logging.String("addr", s.Rest.IP),
		logging.Int("port", s.Rest.Port))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	restAddr := fmt.Sprintf("%s:%d", s.Rest.IP, s.Rest.Port)
	grpcAddr := fmt.Sprintf("%s:%d", s.Node.IP, s.Node.Port)
	jsonPB := &JSONPb{
		EmitDefaults: true,
		Indent:       "  ", // formatted json output
		OrigName:     true,
	}

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonPB),
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
	)

	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := protoapi.RegisterTradingHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}
	if err := protoapi.RegisterTradingDataHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	}

	// CORS support
	handler := cors.Default().Handler(mux)
	handler = gateway.RemoteAddrMiddleware(logger, handler)
	// Gzip encoding support
	handler = NewGzipHandler(*logger, handler.(http.HandlerFunc))

	s.srv = &http.Server{
		Addr:    restAddr,
		Handler: handler,
	}
	// Start http server on port specified
	err = s.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Panic("Failure serving REST proxy API", logging.Error(err))
	}

}

func (s *restProxyServer) Stop() {
	if s.srv != nil {
		s.log.Info("Stopping REST<>GRPC based API")

		if err := s.srv.Shutdown(context.Background()); err != nil {
			s.log.Error("Failed to stop REST<>GRPC based API cleanly",
				logging.Error(err))
		}
	}
}
