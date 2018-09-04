package matching

import (
	"fmt"
	"math"

	"vega/log"
	"vega/msg"
)

type PriceLevel struct {
	price             uint64
	orders            []*msg.Order
	volumeAtTimestamp map[uint64]uint64
	volume uint64
}

func NewPriceLevel(price uint64) *PriceLevel {
	return &PriceLevel{
		price:             price,
		orders:            []*msg.Order{},
		volumeAtTimestamp: make(map[uint64]uint64),
	}
}

func (l *PriceLevel) addOrder(o *msg.Order) {
	// adjust volume by timestamp map for correct pro-rata calculation
	l.increaseVolumeByTimestamp(o)
	// add orders to slice of orders on this price level
	l.orders = append(l.orders, o)

	l.volume += o.Remaining
}

func (l *PriceLevel) removeOrder(index int) {
	l.volume -= l.orders[index].Remaining
	copy(l.orders[index:], l.orders[index+1:])
	l.orders = l.orders[:len(l.orders)-1]
}

func (l *PriceLevel) increaseVolumeByTimestamp(o *msg.Order) {
	if vbt, exists := l.volumeAtTimestamp[o.Timestamp]; exists {
		l.volumeAtTimestamp[o.Timestamp] = vbt + o.Remaining
	} else {
		l.volumeAtTimestamp[o.Timestamp] = o.Remaining
	}
}

func (l *PriceLevel) decreaseVolumeByTimestamp(o *msg.Order) {
	if vbt, exists := l.volumeAtTimestamp[o.Timestamp]; exists {
		if vbt <= o.Remaining {
			delete(l.volumeAtTimestamp, o.Timestamp)
		} else {
			l.volumeAtTimestamp[o.Timestamp] = vbt - o.Remaining
		}
	}
}

func (l *PriceLevel) adjustVolumeByTimestamp(currentTimestamp uint64, trade *msg.Trade) {
	if vbt, exists := l.volumeAtTimestamp[currentTimestamp]; exists {
		l.volumeAtTimestamp[currentTimestamp] = vbt - trade.Size
	}
}

func (l *PriceLevel) uncross(agg *msg.Order) (filled bool, trades []*msg.Trade, impactedOrders []*msg.Order) {

	var (
		toRemove []int
		removed  int
	)

	// start from earliest timestamp
	currentTimestamp := l.earliestTimestamp()
	totalVolumeAtTimestamp := l.volumeAtTimestamp[currentTimestamp]
	volumeToShare := agg.Remaining

	// l.orders is always sorted by timestamps, that is why when iterating we always start from the beginning
	for i, order := range l.orders {

		// See if we are at a new top timestamp
		if currentTimestamp != order.Timestamp {
			// if consumed all orders on the current timestamp, delete exhausted timestamp and proceed to the next one
			delete(l.volumeAtTimestamp, currentTimestamp)
			// assign new timestamp
			currentTimestamp = order.Timestamp
			// assign new volume at timestamp
			totalVolumeAtTimestamp = l.volumeAtTimestamp[currentTimestamp]
			volumeToShare = agg.Remaining
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, order, volumeToShare, totalVolumeAtTimestamp)
		if size <= 0 {
			panic("Trade.size > order.remaining")
		}

		// New Trade
		trade := newTrade(agg, order, size)

		// Update Remaining for both aggressive and passive
		agg.Remaining -= size
		order.Remaining -= size
		l.volume -= size

		// Schedule order for deletion
		if order.Remaining == 0 {
			toRemove = append(toRemove, i)
			l.decreaseVolumeByTimestamp(order)
		}

		// Update Volumes for the price level
		l.adjustVolumeByTimestamp(currentTimestamp, trade)

		// Update trades
		trades = append(trades, trade)
		impactedOrders = append(impactedOrders, order)

		// Exit when done
		if agg.Remaining == 0 {
			break
		}
	}

	if len(toRemove) > 0 {
		for _, idx := range toRemove {
			copy(l.orders[idx-removed:], l.orders[idx-removed+1:])
			removed++
		}
		l.orders = l.orders[:len(l.orders)-removed]
	}

	return agg.Remaining == 0, trades, impactedOrders
}

func (l *PriceLevel) earliestTimestamp() uint64 {
	if len(l.orders) != 0 {
		return l.orders[0].Timestamp
	}
	return 0
}

// Get size for a specific trade assuming aggressive order volume is allocated pro-rata among all passive trades
// with the same timestamp by their share of the total volume with the same price and timestamp. (NB: "normal"
// trading would thus *always* increment the logical timestamp between trades.)
func (l *PriceLevel) getVolumeAllocation(
	agg, pass *msg.Order,
	volumeToShare, initialVolumeAtTimestamp uint64) uint64 {

	weight := float64(pass.Remaining) / float64(initialVolumeAtTimestamp)
	size := weight * float64(min(volumeToShare, initialVolumeAtTimestamp))
	if size-math.Trunc(size) > 0 {
		size++ // Otherwise we can end up allocating 1 short because of integer division rounding
	}
	return min(min(uint64(size), agg.Remaining), pass.Remaining)

	// return min(agg.Remaining, pass.Remaining)
}

// Returns the min of 2 uint64s
func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}

// Creates a trade of a given size between two orders and updates the order details
func newTrade(agg, pass *msg.Order, size uint64) *msg.Trade {
	var buyer, seller *msg.Order
	if agg.Side == msg.Side_Buy {
		buyer = agg
		seller = pass
	} else {
		buyer = pass
		seller = agg
	}

	if agg.Side == pass.Side {
		panic(fmt.Sprintf("agg.side == pass.side (agg: %v, pass: %v)", agg, pass))
	}

	trade := msg.TradePool.Get().(*msg.Trade)
	trade.Market = agg.Market
	trade.Price = pass.Price
	trade.Size = size
	trade.Aggressor = agg.Side
	trade.Buyer = buyer.Party
	trade.Seller = seller.Party
	trade.Timestamp = agg.Timestamp
	return trade
}

func (l PriceLevel) print() {
	//log.Infof("priceLevel: %d\n", l.price)
	for _, o := range l.orders {
		var side string
		if o.Side == msg.Side_Buy {
			side = "BUY"
		} else {
			side = "SELL"
		}

		line := fmt.Sprintf(" %s %s @%d size=%d R=%d Type=%d T=%d %s",
			o.Party, side, o.Price, o.Size, o.Remaining, o.Type, o.Timestamp, o.Id)
		log.Infof("%s", line)
	}
}
