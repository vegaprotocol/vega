// cmd/vega/main.go
package main

import (
	"vega/api/endpoints/rest"
	"vega/api/endpoints/sse"
	"vega/blockchain"
	"vega/core"
	"vega/proto"

)

const sseChannelSize = 2 << 16
const marketName = "BTC/DEC18"

func main() {
	config := core.GetConfig()

	// Vega core
	vega := core.New(config)

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
