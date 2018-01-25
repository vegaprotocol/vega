package mbook

import "fmt"

type BuySell string

const (
	Buy  BuySell = "Buy"
	Sell BuySell = "Sell"
)

type Order struct {
	party      string
	side       BuySell
	size       uint64
	remaining  uint64
	price      uint64
	priceLevel *PriceLevel
}

func (o *Order) crossedWith(side BuySell, price uint64) bool {
	return o.side != side &&
		price > 0 &&
		o.price > 0 &&
		((side == Buy && price >= o.price) || (side == Sell && price <= o.price))
}

func (o *Order) update(t *Trade) {
	if t.size >= o.remaining {
		o.remaining = 0
	} else {
		o.remaining -= t.size
	}
}

func (o *Order) remove() bool {
	//TODO: way remove order
	return false
}

func (o *Order) String() string {
	return fmt.Sprintf(
		"%v %v/%v @%v (%v)",
		o.side,
		o.remaining,
		o.size,
		o.price,
		o.party)
}

func (bs BuySell) getOrder(agg, pass *Order) *Order {
	if agg.side == pass.side {
		panic("order: aggressor and passive orders can't have same side")
	} else if agg.side == bs {
		return agg
	} else { // pass.side == bs
		return pass
	}
}

func (bs BuySell) String() string {
	if bs == Buy {
		return "Buy"
	} else { // bs == Sell
		return "Sell"
	}
}
