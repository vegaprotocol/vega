// cmd/vega/main.go
package main

import (
	"vega/api"
	"vega/api/endpoints/grpc"
	"vega/api/endpoints/restproxy"
	"vega/blockchain"
	"vega/core"
	"vega/datastore"
	"vega/log"
)

func main() {
	config := core.GetConfig()

	log.InitConsoleLogger(log.DebugLevel)

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
	tradeService.Init(vega, storage.TradeStore(), vega.RiskEngine)

	// GRPC server
	grpcServer := grpc.NewGRPCServer(orderService, tradeService)
	go grpcServer.Start()

	// REST<>GRPC (reverse proxy) server
	restServer := restproxy.NewRestProxyServer()
	go restServer.Start()

	if err := blockchain.Start(vega); err != nil {
		log.Fatalf("%s", err)
	}
}
