package main

import (
	"exchange"
	"proto"
)

func main() {
	vega := exchange.New()
	vega.NewMarket("BTC/DEC18")

	vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "A",
		Side:      pb.Order_Buy,
		Price:     100,
		Size:      50,
		Remaining: 50,
		Type:      pb.Order_GTC,
		Sequence:  0,
	})

	vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "B",
		Side:      pb.Order_Buy,
		Price:     102,
		Size:      125,
		Remaining: 125,
		Type:      pb.Order_GTC,
	})

	vega.AddOrder(&pb.Order{
		Market:    "BTC/DEC18",
		Party:     "C",
		Side:      pb.Order_Sell,
		Price:     100,
		Size:      700,
		Remaining: 700,
		Type:      pb.Order_GTC,
	})
}
