package matching

import (
	"fmt"
	"math"
	"sort"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type tsVolPair struct {
	ts  int64
	vol uint64
}

// PriceLevel represents all the Orders placed at a given price.
type PriceLevel struct {
	price             uint64
	proRataMode       bool
	orders            []*types.Order
	volumeAtTimestamp []tsVolPair
	volume            uint64
}

// NewPriceLevel instanciate a new PriceLevel
func NewPriceLevel(price uint64, proRataMode bool) *PriceLevel {
	return &PriceLevel{
		price:             price,
		proRataMode:       proRataMode,
		orders:            []*types.Order{},
		volumeAtTimestamp: []tsVolPair{},
	}
}

func (l *PriceLevel) getOrdersByTrader(trader string) []*types.Order {
	ret := []*types.Order{}
	for _, o := range l.orders {
		if o.PartyID == trader {
			ret = append(ret, o)
		}
	}
	return ret
}

func (l *PriceLevel) addOrder(o *types.Order) {
	// adjust volume by timestamp map for correct pro-rata calculation
	l.increaseVolumeByTimestamp(o)
	// add orders to slice of orders on this price level
	l.orders = append(l.orders, o)
	l.volume += o.Remaining
}

func (l *PriceLevel) removeOrder(index int) {
	// decrease total volume
	l.volume -= l.orders[index].Remaining

	// search the volumeAtTimestamp for this index
	ts := l.orders[index].CreatedAt
	i := sort.Search(len(l.volumeAtTimestamp), func(i int) bool {
		return l.volumeAtTimestamp[i].ts >= ts
	})
	// if we found it, we decrease the volume at timestamp
	if i < len(l.volumeAtTimestamp) && l.volumeAtTimestamp[i].ts == ts {
		if l.volumeAtTimestamp[i].vol > l.orders[index].Remaining {
			l.volumeAtTimestamp[i].vol -= l.orders[index].Remaining
		} else {
			// volume == 0, remove it from the list
			// also this is not a  typo:
			// https://github.com/golang/go/wiki/SliceTricks#delete
			l.volumeAtTimestamp = l.volumeAtTimestamp[:i+copy(l.volumeAtTimestamp[i:], l.volumeAtTimestamp[i+1:])]
		}
	}

	// remove the orders at index
	copy(l.orders[index:], l.orders[index+1:])
	l.orders = l.orders[:len(l.orders)-1]
}

func (l *PriceLevel) increaseVolumeByTimestamp(o *types.Order) {
	// if no volume, or last timestamp is different than the current timestamp
	// which means there's no volume for a given timestamp at the moment
	if len(l.volumeAtTimestamp) <= 0 ||
		l.volumeAtTimestamp[len(l.volumeAtTimestamp)-1].ts != o.CreatedAt {
		l.volumeAtTimestamp = append(l.volumeAtTimestamp, tsVolPair{ts: o.CreatedAt, vol: o.Remaining})
		return
	}

	// then the last one have the same timestamp, which can be possible
	// if other orders with the same price have been placed in the block
	l.volumeAtTimestamp[len(l.volumeAtTimestamp)-1].vol += o.Remaining
}

// in this function it is very much likely that we want to decrease the volume in the first
// time stamp or maybe one of the first as while uncrossing the first few timestamps may
// end up being at a 0 volume before being removed from the map.
// once we found the first time stamp not being == 0, then if it is not the expected
// timestamp, then we will use a binary search to find the correct timestamp as we
// most likely are in the use case where we remove an order which can be any timestamp
func (l *PriceLevel) decreaseVolumeByTimestamp(o *types.Order) {
	var idx int

	// return if with have an empty slice
	if len(l.volumeAtTimestamp) <= 0 {
		// that should never happend as we never call this with no volume bust stilll ...
		return
	}

	// figure out where is the first valid timestamp in there
	// most likely we'll do 0 iteration in here
	for ; idx < len(l.volumeAtTimestamp) &&
		l.volumeAtTimestamp[idx].vol == 0; idx++ {
	}
	if idx >= len(l.volumeAtTimestamp) {
		// this should never happen as we should always have enough volume when trying to decrease
		// , that's weird and should most likely not happen, but let's make sure we do not go out of bound ...
		return
	}

	// so now we check the timestamp
	if l.volumeAtTimestamp[idx].ts == o.CreatedAt {
		if l.volumeAtTimestamp[idx].vol <= o.Remaining {
			// FIXME(jeremy): need to make sure we remove the field if it goes < 0
			l.volumeAtTimestamp[idx].vol = 0
		} else {
			l.volumeAtTimestamp[idx].vol -= o.Remaining
		}
		return
	}

	// last case to handle, we did not find the timestamp first
	// this means we try to delete an order not uncrossing, so let's just remove
	i := sort.Search(len(l.volumeAtTimestamp), func(i int) bool {
		return l.volumeAtTimestamp[i].ts >= o.CreatedAt
	})

	// make sure we found it
	if i >= len(l.volumeAtTimestamp) &&
		l.volumeAtTimestamp[i].ts != o.CreatedAt {
		// we did not find the timestamp, that must be a problem
		// but is never supposed to happen
		return
	}

	// update the pair now as we found it
	if l.volumeAtTimestamp[i].vol <= o.Remaining {
		// FIXME(jeremy): need to make sure we remove the field if it goes < 0
		l.volumeAtTimestamp[i].vol = 0
		return
	}
	l.volumeAtTimestamp[i].vol -= o.Remaining
}

func (l *PriceLevel) adjustVolumeByTimestamp(currentTimestamp int64, trade *types.Trade) {
	var idx int

	// return if with have an empty slice
	if len(l.volumeAtTimestamp) <= 0 {
		// that should never happen as we never call this with no volume, but still ...
		return
	}

	// figure out where is the first valid timestamp in there
	// most likely we'll do 0 iteration in here
	for ; idx < len(l.volumeAtTimestamp) &&
		l.volumeAtTimestamp[idx].vol == 0; idx++ {
	}
	if idx >= len(l.volumeAtTimestamp) {
		// this should never happen as we should always have enough volume when trying to decrease
		// , that's weird and should most likely not happen, but let's make sure we do not go out of bound ...
		return
	}

	// so now we check the timestamp
	if l.volumeAtTimestamp[idx].ts == currentTimestamp {
		if l.volumeAtTimestamp[idx].vol <= trade.Size {
			// FIXME(jeremy): need to make sure we remove the field if it goes < 0
			l.volumeAtTimestamp[idx].vol = 0
		} else {
			l.volumeAtTimestamp[idx].vol -= trade.Size
		}
		return
	}

	// last case to handle, we did not find the timestamp first
	// this means we try to delete an order not uncrossing, so let's just remove
	i := sort.Search(len(l.volumeAtTimestamp), func(i int) bool {
		return l.volumeAtTimestamp[i].ts >= currentTimestamp
	})

	// make sure we found it
	if i >= len(l.volumeAtTimestamp) &&
		l.volumeAtTimestamp[i].ts != currentTimestamp {
		// ok we did not find the actual timestamp, that must be a problem
		// but is never supposed to happen
		return
	}

	// update the pair now as we found it
	if l.volumeAtTimestamp[i].vol <= trade.Size {
		// FIXME(jeremy): need to make sure we remove the field if it goes < 0
		l.volumeAtTimestamp[i].vol = 0
		return
	}

	l.volumeAtTimestamp[i].vol -= trade.Size
}

func (l *PriceLevel) uncross(agg *types.Order) (filled bool, trades []*types.Trade, impactedOrders []*types.Order) {
	// for some reason sometimes it seems the pricelevels are not deleted when getting empty
	// no big deal, just return early
	if len(l.orders) <= 0 {
		return
	}

	var (
		toRemove []int
		removed  int
	)

	// start from earliest timestamp
	tsIdx := 0
	totalVolumeAtTimestamp := l.volumeAtTimestamp[tsIdx].vol
	currentTimestamp := l.volumeAtTimestamp[tsIdx].ts
	volumeToShare := agg.Remaining

	// l.orders is always sorted by timestamps, that is why when iterating we always start from the beginning
	for i, order := range l.orders {

		// See if we are at a new top timestamp
		if currentTimestamp != order.CreatedAt {
			// if consumed all orders on the current timestamp, delete exhausted timestamp and proceed to the next one
			// assign new timestamp
			currentTimestamp = order.CreatedAt
			// increase the volumeAtTimestamp index
			tsIdx += 1
			// assign new volume at timestamp
			totalVolumeAtTimestamp = l.volumeAtTimestamp[tsIdx].vol
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

	// FIXME(jeremy): these need to be optimized, we can make a single copy
	// just by keep the index of the last order which is to remove as they
	// are all order, then just copy the second part of the slice in the actual s[0]
	if len(toRemove) > 0 {
		for _, idx := range toRemove {
			copy(l.orders[idx-removed:], l.orders[idx-removed+1:])
			removed++
		}
		l.orders = l.orders[:len(l.orders)-removed]
	}

	// remove the unused timestamps now
	// first check if the last tsIdx was actually with 0 volume, if not
	// we need to include it from the remaining stuff
	if l.volumeAtTimestamp[tsIdx].vol == 0 {
		tsIdx++
	}
	l.volumeAtTimestamp = l.volumeAtTimestamp[:copy(l.volumeAtTimestamp[0:], l.volumeAtTimestamp[tsIdx:])]
	return agg.Remaining == 0, trades, impactedOrders
}

func (l *PriceLevel) earliestTimestamp() int64 {
	if len(l.orders) != 0 {
		return l.orders[0].CreatedAt
	}
	return 0
}

// Get size for a specific trade assuming aggressive order volume is allocated pro-rata among all passive trades
// with the same timestamp by their share of the total volume with the same price and timestamp. (NB: "normal"
// trading would thus *always* increment the logical timestamp between trades.)
func (l *PriceLevel) getVolumeAllocation(
	agg, pass *types.Order,
	volumeToShare, initialVolumeAtTimestamp uint64) uint64 {

	if l.proRataMode {
		weight := float64(pass.Remaining) / float64(initialVolumeAtTimestamp)
		size := weight * float64(min(volumeToShare, initialVolumeAtTimestamp))
		if size-math.Trunc(size) > 0 {
			size++ // Otherwise we can end up allocating 1 short because of integer division rounding
		}
		return min(min(uint64(size), agg.Remaining), pass.Remaining)
	}

	return min(agg.Remaining, pass.Remaining)
}

// Returns the min of 2 uint64s
func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}

// Creates a trade of a given size between two orders and updates the order details
func newTrade(agg, pass *types.Order, size uint64) *types.Trade {
	var buyer, seller *types.Order
	if agg.Side == types.Side_Buy {
		buyer = agg
		seller = pass
	} else {
		buyer = pass
		seller = agg
	}

	if agg.Side == pass.Side {
		panic(fmt.Sprintf("agg.side == pass.side (agg: %v, pass: %v)", agg, pass))
	}

	return &types.Trade{
		MarketID:  agg.MarketID,
		Price:     pass.Price,
		Size:      size,
		Aggressor: agg.Side,
		Buyer:     buyer.PartyID,
		Seller:    seller.PartyID,
		Timestamp: agg.CreatedAt,
	}
}

func (l PriceLevel) print(log *logging.Logger) {
	log.Debug(fmt.Sprintf("priceLevel: %d\n", l.price))
	for _, o := range l.orders {
		var side string
		if o.Side == types.Side_Buy {
			side = "BUY"
		} else {
			side = "SELL"
		}

		log.Debug(fmt.Sprintf("    %s %s @%d size=%d R=%d Type=%d T=%d %s\n",
			o.PartyID, side, o.Price, o.Size, o.Remaining, o.TimeInForce, o.CreatedAt, o.Id))
	}
}
