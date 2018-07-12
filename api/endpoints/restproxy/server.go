package restproxy

import (
	"context"
	"fmt"
	"net/http"
	
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"vega/api"
	"log"
)

type restProxyServer struct {}

func NewRestProxyServer() *restProxyServer {
	return &restProxyServer{}
}

func (s *restProxyServer) Start() {
	var port = 3005
	var addr = fmt.Sprintf(":%d", port)
	fmt.Printf("Starting REST<>GRPC reverse proxy based HTTP server on port %d...\n", port)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	endpoint := "localhost:3004"
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := api.RegisterTradingHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		log.Fatal(err)
	} else {
		http.ListenAndServe(addr, mux)
	}
}
