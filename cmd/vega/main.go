// cmd/vega/main.go
package main

import (
	"vega/api/endpoints/rest"
	"vega/api/endpoints/sse"
	"vega/blockchain"
	"vega/core"
	"vega/proto"

	"vega/api"
	"vega/datastore"
)

const sseChannelSize = 2 << 16
const storeChannelSize = 2 << 16
const marketName = "BTC/DEC18"

func main() {
	config := core.GetConfig()

	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{marketName})

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(storage.OrderStore())
	tradeService.Init(storage.TradeStore())

	// Vega core
	vega := core.New(config, storage)

	// REST server
	restServer := rest.NewRestServer(vega)
	go restServer.Start()

	// SSE server
	sseOrderChan := make(chan msg.Order, sseChannelSize)
	sseTradeChan := make(chan msg.Trade, sseChannelSize)
	sseServer := sse.NewServer(sseOrderChan, sseTradeChan)
	go sseServer.Start()

	blockchain.Start(vega)

}
