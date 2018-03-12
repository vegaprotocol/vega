package main

import (
	"fmt"

	"exchange"
	"proto"
)

func main() {
	vega := exchange.New()
	vega.NewMarket("BTC/DEC18")

	vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "A",
		Side:      pb.Side_Buy,
		Price:     100,
		Size:      50,
		Remaining: 50,
		Type:      pb.Order_GTC,
		Timestamp: 0,
	})

	vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "B",
		Side:      pb.Side_Buy,
		Price:     102,
		Size:      44,
		Remaining: 44,
		Type:      pb.Order_GTC,
		Timestamp: 0,
	})

	vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "B",
		Side:      pb.Side_Buy,
		Price:     99,
		Size:      42,
		Remaining: 42,
		Type:      pb.Order_GTC,
		Timestamp: 0,
	})

	res, _ :=
		vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "D",
		Side:      pb.Side_Sell,
		Price:     110,
		Size:      100,
		Remaining: 100,
		Type:      pb.Order_GTC,
		Timestamp: 0,
	})

	vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "C",
		Side:      pb.Side_Sell,
		Price:     98,
		Size:      120,
		Remaining: 120,
		Type:      pb.Order_GTC,
		Timestamp: 0,
	})

	vega.RemoveOrder("BTC/DEC18", res.OrderId)

	fmt.Println(vega.GetMarketData("BTC/DEC18"))
}
