// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package matching

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var (
	// ErrWashTrade signals an attempt to a wash trade from a party.
	ErrWashTrade    = errors.New("party attempted to submit wash trade")
	ErrFOKNotFilled = errors.New("FOK order could not be fully filled")
)

// PriceLevel represents all the Orders placed at a given price.
type PriceLevel struct {
	price  *num.Uint
	orders []*types.Order
	volume uint64
}

// NewPriceLevel instantiate a new PriceLevel.
func NewPriceLevel(price *num.Uint) *PriceLevel {
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
		if o.Party == partyID {
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

// uncrossIcebergs when a large aggressive order consumes the peak of iceberg orders, we trade with the hidden portion of
// the icebergs such that when they are refreshed the book does not cross.
func (l *PriceLevel) uncrossIcebergs(agg *types.Order, icebergs []*types.Order, trades []*types.Trade, fake bool) {
	var totalReserved uint64
	for _, b := range icebergs {
		totalReserved += b.IcebergOrder.ReservedRemaining
	}

	if totalReserved == 0 {
		// nothing to do
		return
	}

	// either the amount left of the aggressive order, or the rest of all the iceberg orders
	totalCrossed := num.MinV(agg.Remaining, totalReserved)

	// let do it with decimals
	totalCrossedDec := num.DecimalFromInt64(int64(totalCrossed))
	totalReservedDec := num.DecimalFromInt64(int64(totalReserved))

	// divide up between icebergs
	var sum uint64
	extraTraded := []uint64{}
	for _, b := range icebergs {
		rr := num.DecimalFromInt64(int64(b.IcebergOrder.ReservedRemaining))
		extra := uint64(rr.Mul(totalCrossedDec).Div(totalReservedDec).IntPart())
		sum += extra
		extraTraded = append(extraTraded, extra)
	}

	// if there is some left over due to the rounding when dividing then
	// it is traded against the iceberg with the highest time priority
	if rem := totalCrossed - sum; rem > 0 {
		for i := range icebergs {
			max := icebergs[i].IcebergOrder.ReservedRemaining - extraTraded[i]
			dd := num.MinV(max, rem) // can allocate the smallest of the remainder and whats left in the berg

			extraTraded[i] += dd
			rem -= dd

			if rem == 0 {
				break
			}
		}
		if rem != 0 {
			panic("unable to distribute rounding crumbs between iceberg orders")
		}
	}

	// increase traded sizes based on consumed hidden iceberg volume
	for i := range icebergs {
		extra := extraTraded[i]
		agg.Remaining -= extra
		trades[i].Size += extra
		if !fake {
			// only change values in passive orders if uncrossing is for real and not just to see potential trades.
			icebergs[i].IcebergOrder.ReservedRemaining -= extra
			l.volume -= extra
		}
	}
}

// fakeUncross - this updates a copy of the order passed to it, the copied order is returned.
func (l *PriceLevel) fakeUncross(o *types.Order, checkWashTrades bool) (agg *types.Order, trades []*types.Trade, err error) {
	// work on a copy of the order, so we can submit it a second time
	// after we've done the price monitoring and fees checks
	cpy := *o
	agg = &cpy
	if len(l.orders) == 0 {
		return
	}

	icebergs := []*types.Order{}
	icebergTrades := []*types.Trade{}
	for _, order := range l.orders {
		if checkWashTrades {
			if order.Party == agg.Party {
				err = ErrWashTrade
				return
			}
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, order)
		if size <= 0 {
			panic("Trade.size > order.remaining")
		}

		// New Trade
		trade := newTrade(agg, order, size)

		// Update Remaining for aggressive only
		agg.Remaining -= size

		// Update trades
		trades = append(trades, trade)

		// if the passive order is an iceberg with a hidden quantity make a note of it and
		// its trade incase we need to uncross further
		if order.IcebergOrder != nil && order.IcebergOrder.ReservedRemaining > 0 {
			icebergs = append(icebergs, order)
			icebergTrades = append(icebergTrades, trade)
		}

		// Exit when done
		if agg.Remaining == 0 {
			break
		}
	}

	// if the aggressive trade is not filled uncross with iceberg hidden quantity
	if agg.Remaining != 0 && len(icebergs) > 0 {
		l.uncrossIcebergs(agg, icebergs, icebergTrades, true)
	}

	return agg, trades, err
}

func (l *PriceLevel) uncross(agg *types.Order, checkWashTrades bool) (filled bool, trades []*types.Trade, impactedOrders []*types.Order, err error) {
	// for some reason sometimes it seems the pricelevels are not deleted when getting empty
	// no big deal, just return early
	if len(l.orders) <= 0 {
		return
	}

	var (
		icebergs      []*types.Order
		icebergTrades []*types.Trade
		toRemove      []int
		removed       int
	)

	// l.orders is always sorted by timestamps, that is why when iterating we always start from the beginning
	for i, order := range l.orders {
		// prevent wash trade
		if checkWashTrades {
			if order.Party == agg.Party {
				err = ErrWashTrade
				break
			}
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

		// if the passive order is an iceberg with a hidden quantity make a note of it and
		// its trade incase we need to uncross further
		if order.IcebergOrder != nil && order.IcebergOrder.ReservedRemaining > 0 {
			icebergs = append(icebergs, order)
			icebergTrades = append(icebergTrades, trade)
		}

		// Exit when done
		if agg.Remaining == 0 {
			break
		}
	}

	// if the aggressive trade is not filled uncross with iceberg hidden quantity
	if agg.Remaining > 0 && len(icebergs) > 0 {
		l.uncrossIcebergs(agg, icebergs, icebergTrades, false)
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

// Returns the min of 2 uint64s.
func min(x, y uint64) uint64 {
	if y < x {
		return y
	}
	return x
}

// Returns the max of 2 uint64s.
func max(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

// Creates a trade of a given size between two orders and updates the order details.
func newTrade(agg, pass *types.Order, size uint64) *types.Trade {
	var buyer, seller *types.Order
	if agg.Side == types.SideBuy {
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
		Type:        types.TradeTypeDefault,
		MarketID:    agg.MarketID,
		Price:       pass.Price.Clone(),
		MarketPrice: pass.OriginalPrice.Clone(),
		Size:        size,
		Aggressor:   agg.Side,
		Buyer:       buyer.Party,
		Seller:      seller.Party,
		Timestamp:   agg.CreatedAt,
	}
}

func (l PriceLevel) print(log *logging.Logger) {
	log.Debug(fmt.Sprintf("priceLevel: %d\n", l.price))
	for _, o := range l.orders {
		var side string
		if o.Side == types.SideBuy {
			side = "BUY"
		} else {
			side = "SELL"
		}

		log.Debug(fmt.Sprintf("    %s %s @%d size=%d R=%d Type=%d T=%d %s\n",
			o.Party, side, o.Price, o.Size, o.Remaining, o.TimeInForce, o.CreatedAt, o.ID))
	}
}
