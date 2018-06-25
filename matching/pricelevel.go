package matching

import (
	"container/list"
	"log"
	"math"

	"github.com/google/btree"
)

type PriceLevel struct {
	//side              msg.Side
	price             uint64
	volume            uint64
	volumeByTimestamp map[uint64]uint64
	orders            *list.List
}

func NewPriceLevel(s *OrderBookSide, price uint64) *PriceLevel {
	return &PriceLevel{
		//side:              s.side,
		price:             price,
		orders:            list.New(),
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
	if vbt, exists := l.volumeByTimestamp[o.order.Timestamp]; exists {
		l.volumeByTimestamp[o.order.Timestamp] = vbt + o.order.Remaining
	} else {
		l.volumeByTimestamp[o.order.Timestamp] = o.order.Remaining
	}
	l.volume += o.order.Remaining
	l.orders.PushBack(o)
	o.elem = l.orders.Back()

}

func (l *PriceLevel) removeOrder(o *OrderEntry) {
	if vbt, exists := l.volumeByTimestamp[o.order.Timestamp]; exists {
		if vbt <= o.order.Remaining {
			delete(l.volumeByTimestamp, o.order.Timestamp)
		} else {
			l.volumeByTimestamp[o.order.Timestamp] = vbt - o.order.Remaining
		}
	}
	l.volume -= o.order.Remaining
	l.orders.Remove(o.elem)
	l.orders.Remove(o.elem)
	o.elem = nil
}

func (l *PriceLevel) Less(other btree.Item) bool {
	return l.price > other.(*PriceLevel).price
}

func (l PriceLevel) uncross(agg *OrderEntry, trades *[]Trade) bool {
	volumeToShare := agg.order.Remaining
	currentTimestamp := l.topTimestamp()
	initialVolumeAtTimestamp := l.volumeByTimestamp[currentTimestamp]

	el := l.orders.Front()
	for el != nil && agg.order.Remaining > 0 {

		pass := el.Value.(*OrderEntry)
		log.Println("Passing through: ", pass)

		next := el.Next()

		// See if we are at a new top time
		if currentTimestamp != pass.order.Timestamp {
			delete(l.volumeByTimestamp, currentTimestamp)
			currentTimestamp = pass.order.Timestamp
			initialVolumeAtTimestamp = l.volumeByTimestamp[currentTimestamp]
			volumeToShare = agg.order.Remaining
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, pass, volumeToShare, initialVolumeAtTimestamp)
		if size <= 0 {
			panic("Trade.size > order.remaining")
		}
		trade := newTrade(agg, pass, size)

		// Update book state
		if trade != nil {
			*trades = append(*trades, *trade)
			l.volume -= trade.size
			if vbt, exists := l.volumeByTimestamp[currentTimestamp]; exists {
				l.volumeByTimestamp[currentTimestamp] = vbt - trade.size
			}
			if pass.order.Remaining == 0 {
				l.removeOrder(pass)
			}
			log.Printf("Matched: %v\n", trade)
		}
		el = next
	}
	return agg.order.Remaining == 0
}

// Get size for a specific trade assuming aggressive order volume is allocated pro-rata among all passive trades
// with the same timestamp by their share of the total volume with the same price and timestamp. (NB: "normal"
// trading would thus *always* increment the logical timestamp between trades.)
func (l *PriceLevel) getVolumeAllocation(
	agg, pass *OrderEntry,
	volumeToShare, initialVolumeAtTimestamp uint64) uint64 {

	weight := float64(pass.order.Remaining) / float64(initialVolumeAtTimestamp)
	size := weight * float64(min(volumeToShare, initialVolumeAtTimestamp))
	if size-math.Trunc(size) > 0 {
		size++ // Otherwise we can end up allocating 1 short because of integer division rounding
	}
	return min(min(uint64(size), agg.order.Remaining), pass.order.Remaining)
}
