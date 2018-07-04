// cmd/vega/main.go
package main

import (
	"vega/api/endpoints/rest"
	"vega/api/endpoints/sse"
	"vega/blockchain"
	"vega/core"
	"vega/proto"
)

const sseChannelSize = 32

func main() {

	orderSseChan := make(chan msg.Order, sseChannelSize)
	tradeSseChan := make(chan msg.Trade, sseChannelSize)
	sseServer := sse.NewServer(orderSseChan, tradeSseChan)
	restServer := rest.NewRestServer()

	config := core.DefaultConfig()

	vega := core.New(config)
	vega.CreateMarket("BTC/DEC18")

	go restServer.Start()
	go sseServer.Start()
	blockchain.Start(vega)
}
