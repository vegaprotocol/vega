package market

import (
	"fmt"

	"proto"
)

type Trade struct {
	price     uint64
	size      uint64
	buy       *pb.Order
	sell      *pb.Order
	aggressor *pb.Order
}

func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}

func trade(agg, pass *wrappedOrder) *Trade {
	trade := &Trade{
		price:     pass.order.Price,
		size:      min(agg.order.Remaining, pass.order.Remaining),
		buy:       getOrderForSide(pb.Order_Buy, agg, pass).order,
		sell:      getOrderForSide(pb.Order_Sell, agg, pass).order,
		aggressor: agg.order,
	}
	pass.update(trade)
	agg.update(trade)
	return trade
}

func (t Trade) String() string {
	var buyAgg, sellAgg string
	if t.buy == t.aggressor {
		buyAgg = "*"
	} else if t.sell == t.aggressor {
		sellAgg = "*"
	}
	return fmt.Sprintf(
		"%v%v -> %v%v %v @%v",
		t.sell.Party,
		sellAgg,
		t.buy.Party,
		buyAgg,
		t.size,
		t.price)
}
