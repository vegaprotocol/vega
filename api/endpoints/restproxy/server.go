package restproxy

import (
	"context"
	"fmt"
	"net/http"
	"log"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"vega/api"
	"github.com/rs/cors"
)

type restProxyServer struct {}

func NewRestProxyServer() *restProxyServer {
	return &restProxyServer{}
}

func (s *restProxyServer) Start() {
	var port = 3003
	var addr = fmt.Sprintf(":%d", port)
	fmt.Printf("Starting REST<>GRPC based HTTP server on port %d...\n", port)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	endpoint := "localhost:3002"
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := api.RegisterTradingHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		log.Fatal(err)
	} else {
		// CORS support
		handler := cors.Default().Handler(mux)
		// Gzip encoding support
		handler = NewGzipHandler(handler.(http.HandlerFunc))
		// Start http server on port specified
		http.ListenAndServe(addr, handler)
	}
}
