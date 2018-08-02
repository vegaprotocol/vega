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
	"vega/api/endpoints/gql"
	"vega/msg"
)

func main() {
	// Configuration and logging
	config := core.GetConfig()
	log.InitConsoleLogger(log.DebugLevel)

	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{"BTC/DEC18"}, []string{"partyA", "partyB", "TEST"})

	// VEGA core
	vega := core.New(config, storage)
	vega.InitialiseMarkets()
	vega.RiskEngine.AddNewMarket(&msg.Market{Name: "BTC/DEC18"})

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(vega, storage.OrderStore())
	tradeService.Init(vega, storage.TradeStore(), vega.RiskEngine)

	// GRPC server
	// Port 3002
	grpcServer := grpc.NewGRPCServer(orderService, tradeService)
	go grpcServer.Start()

	// REST<>GRPC (gRPC proxy) server
	// Port 3003
	restServer := restproxy.NewRestProxyServer()
	go restServer.Start()

	// GraphQL server (using new production quality gQL)
	// Port 3004
	graphServer := gql.NewGraphQLServer(orderService, tradeService)
	go graphServer.Start()

	// ABCI socket server
	// Port 46658
	if err := blockchain.Start(vega); err != nil {
		log.Fatalf("%s", err)
	}
}
