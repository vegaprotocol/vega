// cmd/vega/main.go
package main

import (
	"log"
	"vega/api/endpoints/rest"
	"vega/api/endpoints/sse"
	"vega/blockchain"
	"vega/core"
	"vega/proto"

	"vega/api"
	"vega/datastore"
	"vega/api/endpoints/grpc"
	"vega/api/endpoints/restproxy"
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

	// REST server
	restServer := rest.NewRestServer(orderService, tradeService)
	go restServer.Start()

	// GRPC server
	grpcServer := grpc.NewGRPCServer(orderService, tradeService)
	go grpcServer.Start()

	// REST<>GRPC (reverse proxy) server
	restProxyServer := restproxy.NewRestProxyServer()
	go restProxyServer.Start()

	// SSE server
	sseOrderChan := make(chan msg.Order, sseChannelSize)
	sseTradeChan := make(chan msg.Trade, sseChannelSize)
	sseServer := sse.NewServer(sseOrderChan, sseTradeChan)
	go sseServer.Start()

	if err := blockchain.Start(vega); err != nil {
		log.Fatal(err)
	}
}
