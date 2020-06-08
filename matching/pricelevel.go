package matching

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrWashTrade signals an attempt to a wash trade from a party
	ErrWashTrade = errors.New("party attempted to submit wash trade")
)

// PriceLevel represents all the Orders placed at a given price.
type PriceLevel struct {
	price  uint64
	orders []*types.Order
	volume uint64
}

// NewPriceLevel instantiate a new PriceLevel
func NewPriceLevel(price uint64) *PriceLevel {
	return &PriceLevel{
		price:  price,
		orders: []*types.Order{},
	}
}

func (l *PriceLevel) reduceVolume(reduceBy uint64) {
	l.volume -= reduceBy
}

func (l *PriceLevel) getOrdersByParty(partyID string) []*types.Order {
	ret := []*types.Order{}
	for _, o := range l.orders {
		if o.PartyID == partyID {
			ret = append(ret, o)
		}
	}
	return ret
}

func (l *PriceLevel) addOrder(o *types.Order) {
	// add orders to slice of orders on this price level
	l.orders = append(l.orders, o)
	l.volume += o.Remaining
}

func (l *PriceLevel) removeOrder(index int) {
	// decrease total volume
	l.volume -= l.orders[index].Remaining
	// remove the orders at index
	copy(l.orders[index:], l.orders[index+1:])
	l.orders = l.orders[:len(l.orders)-1]
}

func (l *PriceLevel) uncross(agg *types.Order) (filled bool, trades []*types.Trade, impactedOrders []*types.Order, err error) {
	// for some reason sometimes it seems the pricelevels are not deleted when getting empty
	// no big deal, just return early
	if len(l.orders) <= 0 {
		return
	}

	var (
		toRemove []int
		removed  int
	)

	// l.orders is always sorted by timestamps, that is why when iterating we always start from the beginning
	for i, order := range l.orders {
		// prevent wash trade
		if order.PartyID == agg.PartyID {
			err = ErrWashTrade
			break
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, order)
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

	return agg.Remaining == 0, trades, impactedOrders, err
}
func (l *PriceLevel) getVolumeAllocation(agg, pass *types.Order) uint64 {
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
	if agg.Side == types.Side_SIDE_BUY {
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
		Type:      types.Trade_TYPE_DEFAULT,
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
		if o.Side == types.Side_SIDE_BUY {
			side = "BUY"
		} else {
			side = "SELL"
		}

		log.Debug(fmt.Sprintf("    %s %s @%d size=%d R=%d Type=%d T=%d %s\n",
			o.PartyID, side, o.Price, o.Size, o.Remaining, o.TimeInForce, o.CreatedAt, o.Id))
	}
}
