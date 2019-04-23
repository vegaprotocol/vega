package restproxy

import (
	"context"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

const (
	namedLogger = "api.restproxy"
)

type restProxyServer struct {
	log *logging.Logger
	api.Config
	srv *http.Server
}

func NewRestProxyServer(log *logging.Logger, config api.Config) *restProxyServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &restProxyServer{
		log:    log,
		Config: config,
		srv:    nil,
	}
}

func (s *restProxyServer) ReloadConf(cfg api.Config) {
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
		logging.String("addr", s.RestProxyIpAddress),
		logging.Int("port", s.RestProxyServerPort))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	restAddr := fmt.Sprintf("%s:%d", s.RestProxyIpAddress, s.RestProxyServerPort)
	grpcAddr := fmt.Sprintf("%s:%d", s.GrpcServerIpAddress, s.GrpcServerPort)
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
	if err := api.RegisterTradingHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Panic("Failure registering trading handler for REST proxy endpoints", logging.Error(err))
	} else {
		// CORS support
		handler := cors.Default().Handler(mux)
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
