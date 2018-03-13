package main

import (
	"flag"
	"fmt"
	"time"

	"vega/src/core"
	"vega/src/proto"
	"vega/src/tests"
)

func main() {

	benchmark := flag.Bool("bench", false, "Run benchmarks")
	numberOfOrders := flag.Int("orders", 50000, "Number of orders to benchmark")
	blockSize := flag.Int("block", 1, "Block size for timestamp increment")
	uniform := flag.Bool("uniform", false, "Use the same size for all orders")
	flag.Parse()

	if *benchmark {
		tests.BenchmarkMatching(*numberOfOrders, nil, false, *blockSize, *uniform)
		return
	}


	vega := core.New(core.DefaultConfig())
	vega.CreateMarket("BTC/DEC18")

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
}
