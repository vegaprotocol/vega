package restproxy

import (
	"context"
	"fmt"
	"net/http"

	"vega/api"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

type restProxyServer struct {
	*api.Config
}

func NewRestProxyServer(config *api.Config) *restProxyServer {
	return &restProxyServer{
		config,
	}
}

func (s *restProxyServer) Start() {
	logger := *s.GetLogger()
	logger.Infof("Starting REST<>GRPC based HTTP server on port %d...\n", s.RestProxyServerPort)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	restAddr := fmt.Sprintf("%s:%d", s.RestProxyIpAddress, s.RestProxyServerPort)
	grpcAddr := fmt.Sprintf("%s:%d", s.GrpcServerIpAddress, s.GrpcServerPort)
	jsonPB := &JSONPb{
		EmitDefaults: true,
		Indent:       "  ",      // formatted json output
		OrigName:     true,
	}

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonPB),
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
	)

	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := api.RegisterTradingHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		logger.Fatalf("Registering trading handler for rest proxy endpoints %+v", err)
	} else {
		// CORS support
		handler := cors.Default().Handler(mux)
		// Gzip encoding support
		handler = NewGzipHandler(logger, handler.(http.HandlerFunc))
		// Start http server on port specified
		http.ListenAndServe(restAddr, handler)
	}
}
