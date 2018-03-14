package main

import (
	"flag"
	"fmt"
	"time"

	"vega/api"
	"vega/blockchain"
	"vega/core"
	"vega/proto"
	"vega/tests"
)

func main() {
	benchmark := flag.Bool("bench", false, "Run benchmarks")
	blockSize := flag.Int("block", 1, "Block size for timestamp increment")
	chain := flag.Bool("chain", false, "Start a Tendermint blockchain socket")
	numberOfOrders := flag.Int("orders", 50000, "Number of orders to benchmark")
	// restapi := flag.Bool("restapi", false, "Run a REST/JSON HTTP API")
	uniform := flag.Bool("uniform", false, "Use the same size for all orders")
	reportInterval := flag.Int("reportEvery", 0, "Report stats every n orders")
	flag.Parse()

	if *benchmark {
		tests.BenchmarkMatching(*numberOfOrders, nil, false, *blockSize, *uniform, *reportInterval)
		return
	}

	vega := core.New(core.DefaultConfig())
	vega.CreateMarket("BTC/DEC18")

	go api.NewServer()

	if *chain {
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
	fmt.Printf("Elapsed (add order E and match %v trades): %v\n", len(res2.Trades), end.Sub(start))

	vega.DeleteOrder(res.Order.Id)

	fmt.Println(vega.GetMarketData("BTC/DEC18"))
	fmt.Println(vega.GetMarketDepth("BTC/DEC18"))
}
