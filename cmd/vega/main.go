// cmd/vega/main.go
package main

import (
	"log"
	"vega/api"
	"vega/api/endpoints/grpc"
	"vega/api/endpoints/restproxy"
	"vega/blockchain"
	"vega/core"
	"vega/datastore"
)

func main() {
	config := core.GetConfig()

	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{"BTC/DEC18"}, []string{"partyA", "partyB", "TEST"})

	// Vega core
	vega := core.New(config, storage)
	vega.InitialiseMarkets()

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(vega, storage.OrderStore())
	tradeService.Init(vega, storage.TradeStore())

	// GRPC server
	grpcServer := grpc.NewGRPCServer(orderService, tradeService)
	go grpcServer.Start()

	// REST<>GRPC (reverse proxy) server
	restServer := restproxy.NewRestProxyServer()
	go restServer.Start()

	if err := blockchain.Start(vega); err != nil {
		log.Fatal(err)
	}
}
