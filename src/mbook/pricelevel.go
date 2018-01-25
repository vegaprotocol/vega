package mbook

import (
	"container/list"
	"fmt"

	"github.com/google/btree"
)

type PriceLevel struct {
	side   BuySell
	price  uint64
	volume uint64
	orders *list.List
}

func (l *PriceLevel) addOrder(o *Order) {
	if o.remaining > 0 {
		o.priceLevel = l
		l.orders.PushBack(o)
		l.volume += o.remaining
	}
}

func (l *PriceLevel) Less(other btree.Item) bool {
	return (l.side == Buy) == (l.price < other.(*PriceLevel).price)
}

func (l PriceLevel) uncross(agg *Order, trades *list.List) bool {
	for agg.remaining > 0 && l.orders.Len() > 0 {
		pass := l.orders.Front().Value.(*Order)
		trade := trade(agg, pass)
		l.volume -= trade.size
		if pass.remaining == 0 {
			l.orders.Remove(l.orders.Front())
		}
		trades.PushBack(trade)
		fmt.Printf("Trade: %v\n", trade)
	}
	return agg.remaining == 0
}