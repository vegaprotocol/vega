package matching

import (
	"container/list"
	"fmt"
	"math"

	"vega/src/proto"

	"github.com/google/btree"
)

type PriceLevel struct {
	book              *OrderBook
	side              msg.Side
	price             uint64
	volume            uint64
	volumeByTimestamp map[uint64]uint64
	orders            *list.List
}

func NewPriceLevel(s *OrderBookSide, price uint64) *PriceLevel {
	return &PriceLevel{
		book: s.book,
		side: s.side,
		price: price,
		orders: list.New(),
		volumeByTimestamp: make(map[uint64]uint64),
	}
}

func (l *PriceLevel) firstOrder() *OrderEntry {
	if l.orders.Front() != nil {
		return l.orders.Front().Value.(*OrderEntry)
	} else {
		return nil
	}
}

func (l *PriceLevel) lastOrder() *OrderEntry {
	if l.orders.Back() != nil {
		return l.orders.Back().Value.(*OrderEntry)
	} else {
		return nil
	}
}

func (l *PriceLevel) topTimestamp() uint64 {
	if l.firstOrder() != nil {
		return l.firstOrder().order.Timestamp
	} else {
		return 0
	}
}


func (l *PriceLevel) addOrder(o *OrderEntry) {
	if o.order.Remaining > 0 {
		o.priceLevel = l
		if vbt, exists := l.volumeByTimestamp[o.order.Timestamp]; exists {
			l.volumeByTimestamp[o.order.Timestamp] = vbt + o.order.Remaining
		} else {
			l.volumeByTimestamp[o.order.Timestamp] = o.order.Remaining
		}
		o.elem = l.orders.PushBack(o)
		l.volume += o.order.Remaining
	}
}

func (l *PriceLevel) removeOrder(o *OrderEntry) *OrderEntry {
	if l != o.priceLevel || l.price != o.order.Price {
		panic("removeOrder called on wrong price level for order/price")
	}
	o.priceLevel.volume -= o.order.Remaining
	o.priceLevel.orders.Remove(o.elem)
	o.elem = nil
	if vbt, exists := o.priceLevel.volumeByTimestamp[o.order.Timestamp]; exists {
		o.priceLevel.volumeByTimestamp[o.order.Timestamp] = vbt - o.order.Remaining
	}
	if o.priceLevel.volume == 0 {
		o.side.removePriceLevel(o.priceLevel.price)
	}
	o.priceLevel = nil
	return o
}

func (l *PriceLevel) Less(other btree.Item) bool {
	return (l.side == msg.Side_Buy) == (l.price < other.(*PriceLevel).price)
}

func (l PriceLevel) uncross(agg *OrderEntry, trades *[]Trade) bool {
	volumeToShare := agg.order.Remaining
	currentTimestamp := l.topTimestamp()
	el := l.orders.Front()
	for el != nil && agg.order.Remaining > 0 {

		pass := el.Value.(*OrderEntry)
		next := el.Next()

		// See if we are at a new top time
		if currentTimestamp != pass.order.Timestamp {
			currentTimestamp = pass.order.Timestamp
			volumeToShare = agg.order.Remaining
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, pass, volumeToShare)
		trade := newTrade(agg, pass, size)

		// Update book state
		if trade != nil {
			l.volume -= trade.size
			if pass.order.Remaining == 0 {
				pass.remove()
			}
			*trades = append(*trades, *trade)
			if !l.book.config.Quiet {
				fmt.Printf("Matched: %v\n", trade)
			}
		}
		el = next
	}
	return agg.order.Remaining == 0
}

func (l *PriceLevel) getVolumeAllocation(agg, pass *OrderEntry, volumeToShare uint64) uint64 {
	volumeAtPassiveTimestamp := l.volumeByTimestamp[pass.order.Timestamp]
	weight := float64(pass.order.Remaining) / float64(volumeAtPassiveTimestamp)
	size := weight * float64(min(volumeToShare, volumeAtPassiveTimestamp))
	if size-math.Trunc(size) > 0 {
		size++ // Otherwise we can end up allocating 1 short because of integer division rounding
	}
	return min(min(uint64(size), agg.order.Remaining), pass.order.Remaining)
}
