package mbook

import "fmt"

type Trade struct {
	price     uint64
	size      uint64
	buy       *Order
	sell      *Order
	aggressor *Order
}

func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}

func trade(agg, pass *Order) *Trade {
	trade := &Trade{
		price:     pass.price,
		size:      min(agg.remaining, pass.remaining),
		buy:       Buy.getOrder(agg, pass),
		sell:      Sell.getOrder(agg, pass),
		aggressor: agg,
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
		t.sell.party,
		sellAgg,
		t.buy.party,
		buyAgg,
		t.size,
		t.price)
}
