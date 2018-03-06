package main

import (
	"book"
	"proto"
)

func main() {
	vega := book.NewBook("test vega")
	A := &book.Party{Name: "A"}
	B := &book.Party{Name: "B"}
	C := &book.Party{Name: "C"}
	vega.AddOrder(&pb.Order{
		Market:    vega.GetId(),
		Party:     A.GetId(),
		Side:      pb.Order_Buy,
		Price:     100,
		Size:      50,
		Remaining: 50,
		Type:      pb.Order_GTC,
		Sequence:  0,
	})
	vega.AddOrder(&pb.Order{
		Market:    vega.GetId(),
		Party:     B.GetId(),
		Side:      pb.Order_Buy,
		Price:     102,
		Size:      125,
		Remaining: 125,
		Type:      pb.Order_GTC,
	})
	vega.AddOrder(&pb.Order{
		Market:    vega.GetId(),
		Party:     C.GetId(),
		Side:      pb.Order_Sell,
		Price:     100,
		Size:      700,
		Remaining: 700,
		Type:      pb.Order_GTC,
	})
}
