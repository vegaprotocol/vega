// cmd/vega/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"vega/blockchain"
	"vega/core"
	"vega/services/msg"

	"vega/api"
	"vega/datastore"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

const sseChannelSize = 2 << 16
const marketName = "BTC/DEC18"

func main() {
	config := core.GetConfig()

	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{marketName})

	// Vega core
	vega := core.New(config, storage)
	vega.InitialiseMarkets()

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(vega, storage.OrderStore())
	tradeService.Init(vega, storage.TradeStore())

	// Listen for GRPC requests
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 5678))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	msg.RegisterTradingServer(grpcServer, orderService)
	go grpcServer.Serve(lis)

	// Listen for JSON requests
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err2 := msg.RegisterTradingHandlerFromEndpoint(ctx, mux, "localhost:5678", opts)
	if err2 != nil {
		fmt.Println(err2)
	}

	go http.ListenAndServe(":8080", mux)

	if err := blockchain.Start(vega); err != nil {
		log.Fatal(err)
	}
}
