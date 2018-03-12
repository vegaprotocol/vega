package matching

import (
	"container/list"
	"fmt"
	"math"

	"proto"

	"github.com/google/btree"
)

type PriceLevel struct {
	side           msg.Side
	price          uint64
	volume         uint64
	volumeAtTop    uint64
	timestampAtTop uint64
	orders         *list.List
}

func (l *PriceLevel) firstOrder() *OrderEntry {
	return l.orders.Front().Value.(*OrderEntry)
}

func (l *PriceLevel) lastOrder() *OrderEntry {
	return l.orders.Back().Value.(*OrderEntry)
}

func (l *PriceLevel) addOrder(o *OrderEntry) {
	if o.order.Remaining > 0 {
		o.priceLevel = l
		if l.orders.Len() == 0 {
			l.timestampAtTop = o.order.Timestamp
		}
		if o.order.Timestamp == l.timestampAtTop {
			l.volumeAtTop += o.order.Remaining
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
	if o.priceLevel.volume == 0 {
		o.side.removePriceLevel(o.priceLevel.price)
	}
	o.priceLevel = nil
	return o
}

func (l *PriceLevel) recalculateVolumeAtTop() {
	volumeAtTop := uint64(0)
	timestamp := l.firstOrder().order.Timestamp
	for el := l.orders.Front();
		el != nil && el.Value.(*OrderEntry).order.Timestamp == timestamp;
		el = el.Next() {

		volumeAtTop += el.Value.(*OrderEntry).order.Remaining
	}
	l.timestampAtTop = timestamp
	l.volumeAtTop = volumeAtTop
}

func (l *PriceLevel) Less(other btree.Item) bool {
	return (l.side == msg.Side_Buy) == (l.price < other.(*PriceLevel).price)
}

func (l PriceLevel) uncross(agg *OrderEntry, trades *[]Trade) bool {
	volumeToShare := agg.order.Remaining
	el := l.orders.Front()
	for el != nil && agg.order.Remaining > 0 {

		pass := el.Value.(*OrderEntry)
		next := el.Next()

		// See if we are at a new top time
		if pass.order.Timestamp != l.timestampAtTop {
			l.recalculateVolumeAtTop()
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
			fmt.Printf("Matched: %v\n", trade)
		}
		el = next
	}

	return agg.order.Remaining == 0
}

func (l *PriceLevel) getVolumeAllocation(agg, pass *OrderEntry, volumeToShare uint64) uint64 {
	weight := float64(pass.order.Remaining) / float64(l.volumeAtTop)
	size := weight * float64(min(volumeToShare, l.volumeAtTop))
	if size-math.Trunc(size) > 0 {
		size++ // Otherwise we can end up allocating 1 short because of integer division rounding
	}
	return min(uint64(size), agg.order.Remaining)
}
