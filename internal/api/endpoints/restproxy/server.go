package restproxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/logging"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

type restProxyServer struct {
	*api.Config
	srv *http.Server
}

func NewRestProxyServer(config *api.Config) *restProxyServer {
	return &restProxyServer{
		config, nil,
	}
}

func (s *restProxyServer) Start() {
	logger := *s.GetLogger()

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
		handler = NewGzipHandler(logger, handler.(http.HandlerFunc))
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

func (s *restProxyServer) Stop() error {
	if s.srv != nil {
		return s.srv.Shutdown(context.Background())
	}
	return errors.New("Rest proxy server not started")
}
