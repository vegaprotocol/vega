// cmd/vega/main.go
package main

import (
	"flag"
	"vega/api/endpoints/rest"
	"vega/api/endpoints/sse"
	"vega/blockchain"
	"vega/core"
	"vega/datastore"
	"vega/proto"
	"vega/api"
)

const sseChannelSize = 2 << 16
const storeChannelSize = 2 << 16
const marketName = "BTC/DEC18"

func main() {
	chain := flag.Bool("chain", false, "Start a Tendermint blockchain socket")
	flag.Parse()

	config := core.DefaultConfig()

	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	storeOrderChan := make(chan msg.Order, storeChannelSize)
	storeTradeChan := make(chan msg.Trade, storeChannelSize)
	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{marketName}, storeOrderChan, storeTradeChan)

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(storage.OrderStore())
	tradeService.Init(storage.TradeStore())
	// REST server
	restServer := rest.NewRestServer(orderService, tradeService)

	sseOrderChan := make(chan msg.Order, sseChannelSize)
	sseTradeChan := make(chan msg.Trade, sseChannelSize)
	sseServer := sse.NewServer(sseOrderChan, sseTradeChan)

	config.Matching.OrderChans = append(config.Matching.OrderChans, sseOrderChan)
	config.Matching.TradeChans = append(config.Matching.TradeChans, sseTradeChan)

	config.Matching.OrderChans = append(config.Matching.OrderChans, storeOrderChan)
	config.Matching.TradeChans = append(config.Matching.TradeChans, storeTradeChan)

	vega := core.New(config)
	vega.CreateMarket(marketName)

	if *chain {
		go restServer.Start()
		go sseServer.Start()
		blockchain.Start(*vega)
		return
	}
}
