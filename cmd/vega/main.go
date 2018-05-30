// cmd/vega/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"vega/api/endpoints/sse"
	"vega/api/endpoints/rest"
	"vega/blockchain"
	"vega/core"
	"vega/proto"
)

const sseChannelSize = 32

func main() {
	chain := flag.Bool("chain", false, "Start a Tendermint blockchain socket")
	flag.Parse()

	orderSseChan := make(chan msg.Order, sseChannelSize)
	tradeSseChan := make(chan msg.Trade, sseChannelSize)
	sseServer := sse.NewServer(orderSseChan, tradeSseChan)
	restServer := rest.NewRestServer()

	config := core.DefaultConfig()
	config.Matching.OrderChans = append(config.Matching.OrderChans, orderSseChan)
	config.Matching.TradeChans = append(config.Matching.TradeChans, tradeSseChan)

	vega := core.New(config)
	vega.CreateMarket("BTC/DEC18")

	if *chain {
		go restServer.Start()
		go sseServer.Start()
		blockchain.Start(*vega)
		return
	}

	vega.SubmitOrder(msg.Order{
		Market:    "BTC/DEC18",
		Party:     "A",
		Side:      msg.Side_Buy,
		Price:     100,
		Size:      50,
		Remaining: 50,
		Type:      msg.Order_GTC,
		Timestamp: 0,
	})

	vega.SubmitOrder(msg.Order{
		Market:    "BTC/DEC18",
		Party:     "B",
		Side:      msg.Side_Buy,
		Price:     102,
		Size:      44,
		Remaining: 44,
		Type:      msg.Order_GTC,
		Timestamp: 0,
	})

	vega.SubmitOrder(msg.Order{
		Market:    "BTC/DEC18",
		Party:     "C",
		Side:      msg.Side_Buy,
		Price:     99,
		Size:      42,
		Remaining: 42,
		Type:      msg.Order_GTC,
		Timestamp: 0,
	})

	res, _ :=
		vega.SubmitOrder(msg.Order{
			Market:    "BTC/DEC18",
			Party:     "D",
			Side:      msg.Side_Sell,
			Price:     110,
			Size:      100,
			Remaining: 100,
			Type:      msg.Order_GTC,
			Timestamp: 0,
		})

	start := time.Now()
	res2, _ := vega.SubmitOrder(msg.Order{
		Market:    "BTC/DEC18",
		Party:     "E",
		Side:      msg.Side_Sell,
		Price:     98,
		Size:      120,
		Remaining: 120,
		Type:      msg.Order_GTC,
		Timestamp: 0,
	})
	end := time.Now()
	log.Printf("Elapsed (add order E and match %v trades): %v\n", len(res2.Trades), end.Sub(start))

	vega.DeleteOrder(res.Order.Id)

	fmt.Println(vega.GetMarketData("BTC/DEC18"))
	fmt.Println(vega.GetMarketDepth("BTC/DEC18"))
}
