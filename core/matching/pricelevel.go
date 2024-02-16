// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package matching

import (
	"errors"
	"fmt"
	"sort"

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

// trackIceberg holds together information about iceberg orders while we are uncrossing
// so we can trade against them all again but distributed evenly.
type trackIceberg struct {
	// the iceberg order
	order *types.Order
	// the trade that occurred with the icebergs visible peak
	trade *types.Trade
	// the index of the iceberg order in the price-level slice
	idx int
}

func (t *trackIceberg) reservedRemaining() uint64 {
	return t.order.IcebergOrder.ReservedRemaining
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
	l.volume += o.TrueRemaining()
}

func (l *PriceLevel) removeOrder(index int) {
	// decrease total volume
	l.volume -= l.orders[index].TrueRemaining()
	// remove the orders at index
	copy(l.orders[index:], l.orders[index+1:])
	l.orders = l.orders[:len(l.orders)-1]
}

// uncrossIcebergs when a large aggressive order consumes the peak of iceberg orders, we trade with the hidden portion of
// the icebergs such that when they are refreshed the book does not cross.
func (l *PriceLevel) uncrossIcebergs(agg *types.Order, tracked []*trackIceberg, fake bool) ([]*types.Trade, []*types.Order) {
	var totalReserved uint64
	for _, t := range tracked {
		totalReserved += t.reservedRemaining()
	}

	if totalReserved == 0 {
		// nothing to do
		return nil, nil
	}

	// either the amount left of the aggressive order, or the rest of all the iceberg orders
	totalCrossed := num.MinV(agg.Remaining, totalReserved)

	// let do it with decimals
	totalCrossedDec := num.DecimalFromInt64(int64(totalCrossed))
	totalReservedDec := num.DecimalFromInt64(int64(totalReserved))

	// divide up between icebergs
	var sum uint64
	extraTraded := []uint64{}
	for _, t := range tracked {
		rr := num.DecimalFromInt64(int64(t.reservedRemaining()))
		extra := uint64(rr.Mul(totalCrossedDec).Div(totalReservedDec).IntPart())
		sum += extra
		extraTraded = append(extraTraded, extra)
	}

	// if there is some left over due to the rounding when dividing then
	// it is traded against the iceberg with the highest time priority
	if rem := totalCrossed - sum; rem > 0 {
		for i, t := range tracked {
			max := t.reservedRemaining() - extraTraded[i]
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
	newTrades := []*types.Trade{}
	newImpacted := []*types.Order{}
	for i, t := range tracked {
		extra := extraTraded[i]
		agg.Remaining -= extra

		// if there was not a previous trade with the iceberg's peak, make a fresh one
		if t.trade == nil {
			t.trade = newTrade(agg, t.order, 0)
			newTrades = append(newTrades, t.trade)
			newImpacted = append(newImpacted, t.order)
		}
		t.trade.Size += extra

		if !fake {
			// only change values in passive orders if uncrossing is for real and not just to see potential trades.
			t.order.IcebergOrder.ReservedRemaining -= extra
			l.volume -= extra
		}
	}
	return newTrades, newImpacted
}

// fakeUncross - this updates a copy of the order passed to it, the copied order is returned.
func (l *PriceLevel) fakeUncross(o *types.Order, checkWashTrades bool) (agg *types.Order, trades []*types.Trade, err error) {
	// work on a copy of the order, so we can submit it a second time
	// after we've done the price monitoring and fees checks
	agg = o.Clone()
	if len(l.orders) == 0 {
		return
	}

	icebergs := []*trackIceberg{}
	for i, order := range l.orders {
		if checkWashTrades {
			if order.Party == agg.Party {
				err = ErrWashTrade
				return
			}
		}

		// Get size and make newTrade
		size := l.getVolumeAllocation(agg, order)
		if size <= 0 {
			// this is only fine if it is an iceberg order with only reserve and in that case
			// we need to trade with it later in uncrossIcebergs
			if order.IcebergOrder != nil &&
				order.Remaining == 0 &&
				order.IcebergOrder.ReservedRemaining != 0 {
				icebergs = append(icebergs, &trackIceberg{order, nil, i})
				continue
			}

			panic("Trade.size > order.remaining")
		}

		// New Trade
		trade := newTrade(agg, order, size)
		trade.SellOrder = agg.ID
		trade.BuyOrder = order.ID
		if agg.Side == types.SideBuy {
			trade.SellOrder, trade.BuyOrder = trade.BuyOrder, trade.SellOrder
		}

		// Update Remaining for aggressive only
		agg.Remaining -= size

		// Update trades
		trades = append(trades, trade)

		// if the passive order is an iceberg with a hidden quantity make a note of it and
		// its trade incase we need to uncross further
		if order.IcebergOrder != nil && order.IcebergOrder.ReservedRemaining > 0 {
			icebergs = append(icebergs, &trackIceberg{order, trade, i})
		}

		// Exit when done
		if agg.Remaining == 0 {
			break
		}
	}

	// if the aggressive trade is not filled uncross with iceberg hidden quantity
	if agg.Remaining != 0 && len(icebergs) > 0 {
		newTrades, _ := l.uncrossIcebergs(agg, icebergs, true)
		trades = append(trades, newTrades...)
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
		icebergs []*trackIceberg
		toRemove []int
		removed  int
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
			// this is only fine if it is an iceberg order with only reserve and in that case
			// we need to trade with it later in uncrossIcebergs
			if order.IcebergOrder != nil &&
				order.Remaining == 0 &&
				order.IcebergOrder.ReservedRemaining != 0 {
				icebergs = append(icebergs, &trackIceberg{order, nil, i})
				continue
			}
			panic("Trade.size > order.remaining")
		}

		// New Trade
		trade := newTrade(agg, order, size)
		trade.SellOrder, trade.BuyOrder = agg.ID, order.ID
		if agg.Side == types.SideBuy {
			trade.SellOrder, trade.BuyOrder = trade.BuyOrder, trade.SellOrder
		}

		// Update Remaining for both aggressive and passive
		agg.Remaining -= size
		order.Remaining -= size
		l.volume -= size

		if order.TrueRemaining() == 0 {
			toRemove = append(toRemove, i)
		}

		// Update trades
		trades = append(trades, trade)
		impactedOrders = append(impactedOrders, order)

		// if the passive order is an iceberg with a hidden quantity make a note of it and
		// its trade incase we need to uncross further
		if order.IcebergOrder != nil && order.IcebergOrder.ReservedRemaining > 0 {
			icebergs = append(icebergs, &trackIceberg{order, trade, i})
		}

		// Exit when done
		if agg.Remaining == 0 {
			break
		}
	}

	// if the aggressive trade is not filled uncross with iceberg hidden reserves
	if agg.Remaining > 0 && len(icebergs) > 0 {
		newTrades, newImpacted := l.uncrossIcebergs(agg, icebergs, false)
		trades = append(trades, newTrades...)
		impactedOrders = append(impactedOrders, newImpacted...)

		// only remove fully depleted icebergs, icebergs with 0 remaining but some in reserve
		// stay at the pricelevel until they refresh at the end of execution, or the end of auction uncrossing
		for _, t := range icebergs {
			if t.order.TrueRemaining() == 0 {
				toRemove = append(toRemove, t.idx)
			}
		}
		sort.Ints(toRemove)
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
