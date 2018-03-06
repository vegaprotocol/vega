package book

import (
	"container/list"
	"fmt"

	"proto"

	"github.com/google/btree"
)

type PriceLevel struct {
	side        pb.Order_Side
	price       uint64
	volume      uint64
	volumeAtTop uint64
	orders      *list.List
}

func (l *PriceLevel) addOrder(o *wrappedOrder) {
	if o.order.Remaining > 0 {
		o.priceLevel = l
		o.elem = l.orders.PushBack(o)
		l.volume += o.order.Remaining
	}
}

func (l *PriceLevel) Less(other btree.Item) bool {
	return (l.side == pb.Order_Buy) == (l.price < other.(*PriceLevel).price)
}

func (l PriceLevel) uncross(agg *wrappedOrder, trades *[]Trade) bool {
	for agg.order.Remaining > 0 && l.orders.Len() > 0 {
		pass := l.orders.Front().Value.(*wrappedOrder)
		trade := trade(agg, pass)
		if trade != nil {
			l.volume -= trade.size
			if pass.order.Remaining == 0 {
				l.orders.Remove(l.orders.Front())
			}
			*trades = append(*trades, *trade)
			fmt.Printf("Trade: %v\n", trade)
		}
	}
	return agg.order.Remaining == 0
}
