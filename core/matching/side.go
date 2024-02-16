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
	"encoding/binary"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/pkg/errors"
)

// ErrPriceNotFound signals that a price was not found on the book side.
var ErrPriceNotFound = errors.New("price-volume pair not found")

// OrderBookSide represent a side of the book, either Sell or Buy.
type OrderBookSide struct {
	side   types.Side
	log    *logging.Logger
	levels []*PriceLevel
}

func (s *OrderBookSide) Hash() []byte {
	// 32 num.Uint.Bytes() for price + 8 for volume
	output := make([]byte, len(s.levels)*40)
	var i int
	for _, l := range s.levels {
		// Data is already coming as big endian out of
		// Uint.Bytes()
		price := l.price.Bytes()
		copy(output[i:], price[:])
		i += 32
		binary.BigEndian.PutUint64(output[i:], l.volume)
		i += 8
	}
	return crypto.Hash(output)
}

func (s *OrderBookSide) cleanup() {
	s.levels = nil
}

// When we leave an auction we need to remove any orders marked as GFA.
func (s *OrderBookSide) getOrdersToCancel(auction bool) []*types.Order {
	ordersToCancel := make([]*types.Order, 0)
	for _, pricelevel := range s.levels {
		for _, order := range pricelevel.orders {
			// Find orders to cancel
			if (order.TimeInForce == types.OrderTimeInForceGFA && !auction) ||
				(order.TimeInForce == types.OrderTimeInForceGFN && auction) {
				// Save order to send back to client
				ordersToCancel = append(ordersToCancel, order)
			}
		}
	}
	return ordersToCancel
}

func (s *OrderBookSide) addOrder(o *types.Order) {
	// update the price-volume map
	s.getPriceLevel(o.Price).addOrder(o)
}

// BestPriceAndVolume returns the top of book price and volume
// returns an error if the book is empty.
func (s *OrderBookSide) BestPriceAndVolume() (*num.Uint, uint64, error) {
	if len(s.levels) <= 0 {
		return num.UintZero(), 0, errors.New("no orders on the book")
	}
	last := len(s.levels) - 1
	return s.levels[last].price.Clone(), s.levels[last].volume, nil
}

// BestStaticPrice returns the top of book price for non pegged orders
// We do not keep count of the volume which makes this slightly quicker
// returns an error if the book is empty.
func (s *OrderBookSide) BestStaticPrice() (*num.Uint, error) {
	if len(s.levels) <= 0 {
		return num.UintZero(), errors.New("no orders on the book")
	}

	for i := len(s.levels) - 1; i >= 0; i-- {
		pricelevel := s.levels[i]
		for _, order := range pricelevel.orders {
			if order.PeggedOrder == nil {
				return pricelevel.price.Clone(), nil
			}
		}
	}
	return num.UintZero(), errors.New("no non pegged orders found on the book")
}

// BestStaticPriceAndVolume returns the top of book price for non pegged orders
// returns an error if the book is empty.
func (s *OrderBookSide) BestStaticPriceAndVolume() (*num.Uint, uint64, error) {
	if len(s.levels) <= 0 {
		return num.UintZero(), 0, errors.New("no orders on the book")
	}

	var (
		bestPrice  = num.UintZero()
		bestVolume uint64
	)
	for i := len(s.levels) - 1; i >= 0; i-- {
		pricelevel := s.levels[i]
		for _, order := range pricelevel.orders {
			if order.PeggedOrder == nil {
				bestPrice = pricelevel.price
				bestVolume += order.Remaining
			}
		}
		// If we found a price, return it
		if bestPrice.GT(num.UintZero()) {
			return bestPrice.Clone(), bestVolume, nil
		}
	}
	return num.UintZero(), 0, errors.New("no non pegged orders found on the book")
}

func (s *OrderBookSide) amendIcebergOrder(amendOrder *types.Order, oldOrder *types.Order, priceLevelIndex int, orderIndex int) (int64, error) {
	if amendOrder.Remaining > oldOrder.Remaining {
		// iceberg amend should never increase the visible remaining
		return 0, types.ErrOrderAmendFailure
	}

	// set the new order in the level
	s.levels[priceLevelIndex].orders[orderIndex] = amendOrder

	// iceberg orders are a little different because they can be increased or decreased in size but
	// amended in place. This is because on increase only the reserve amount it changed.
	oldReserved := oldOrder.IcebergOrder.ReservedRemaining
	amendReserved := amendOrder.IcebergOrder.ReservedRemaining
	if amendReserved > oldReserved {
		// only increased volume diff is easy
		inc := amendReserved - oldReserved
		s.levels[priceLevelIndex].volume += inc
		return int64(inc), nil
	}

	if amendReserved < oldReserved {
		dec := oldOrder.Remaining - amendOrder.Remaining
		dec += oldReserved - amendReserved
		s.levels[priceLevelIndex].reduceVolume(dec)
		return -int64(dec), nil
	}

	// this is the case where we have an iceberg with no reserve, and reducing its visible peak
	if oldOrder.Remaining < amendOrder.Remaining {
		panic("we should not be increasing iceberg visble size in-place")
	}
	return -int64(oldOrder.Remaining - amendOrder.Remaining), nil
}

func (s *OrderBookSide) amendOrder(orderAmend *types.Order) (int64, error) {
	priceLevelIndex := -1
	orderIndex := -1
	var oldOrder *types.Order

	for idx, priceLevel := range s.levels {
		if priceLevel.price.EQ(orderAmend.Price) {
			priceLevelIndex = idx
			for j, order := range priceLevel.orders {
				if order.ID == orderAmend.ID {
					orderIndex = j
					oldOrder = order
					break
				}
			}
			break
		}
	}

	if oldOrder == nil || priceLevelIndex == -1 || orderIndex == -1 {
		return 0, types.ErrOrderNotFound
	}

	if oldOrder.Party != orderAmend.Party {
		return 0, types.ErrOrderAmendFailure
	}

	if oldOrder.Reference != orderAmend.Reference {
		return 0, types.ErrOrderAmendFailure
	}

	if oldOrder.IcebergOrder != nil {
		return s.amendIcebergOrder(orderAmend, oldOrder, priceLevelIndex, orderIndex)
	}

	if oldOrder.Size < orderAmend.Size &&
		oldOrder.Remaining < orderAmend.Size {
		return 0, types.ErrOrderAmendFailure
	}

	reduceBy := oldOrder.Remaining - orderAmend.Remaining
	s.levels[priceLevelIndex].orders[orderIndex] = orderAmend
	s.levels[priceLevelIndex].reduceVolume(reduceBy)
	return -int64(reduceBy), nil
}

// ExtractOrders extracts the orders from the top of the book until the volume amount is hit,
// if removeOrders is set to True then the relevant orders also get removed.
func (s *OrderBookSide) ExtractOrders(price *num.Uint, volume uint64, removeOrders bool) []*types.Order {
	extractedOrders := []*types.Order{}
	var (
		totalVolume uint64
		checkPrice  func(*num.Uint) bool
	)
	if s.side == types.SideBuy {
		checkPrice = func(orderPrice *num.Uint) bool { return orderPrice.GTE(price) }
	} else {
		checkPrice = func(orderPrice *num.Uint) bool { return orderPrice.LTE(price) }
	}

	for i := len(s.levels) - 1; i >= 0; i-- {
		pricelevel := s.levels[i]
		var toRemove int
		for _, order := range pricelevel.orders {
			// Check the price is good and the total volume will not be exceeded
			if checkPrice(order.Price) && totalVolume+order.TrueRemaining() <= volume {
				// Remove this order
				extractedOrders = append(extractedOrders, order.Clone())
				totalVolume += order.TrueRemaining()
				// Remove the order from the price level
				toRemove++
			} else {
				// We should never get to here unless the passed in price
				// and volume are not correct
				s.log.Panic("Failed to extract orders as not enough volume within price limits",
					logging.BigUint("price", price),
					logging.Uint64("required-volume", volume),
					logging.Uint64("found-volume", totalVolume),
					logging.Bool("remove-orders", removeOrders))
			}

			// If we have the right amount, stop processing
			if totalVolume == volume {
				break
			}
		}

		if removeOrders {
			for ; toRemove > 0; toRemove-- {
				pricelevel.removeOrder(0)
			}
			// Erase this price level which will be at the end of the slice
			if len(pricelevel.orders) == 0 {
				s.levels[i] = nil
				s.levels = s.levels[:len(s.levels)-1]
			}
		}

		// Check if we have done enough
		if totalVolume == volume {
			break
		}
	}
	// If we get here and don't have the full amount of volume
	// something has gone wrong
	if totalVolume != volume {
		s.log.Panic("Failed to extract orders as not enough volume on the book",
			logging.BigUint("Price", price), logging.Uint64("volume", volume))
	}

	return extractedOrders
}

// RemoveOrder will remove an order from the book.
func (s *OrderBookSide) RemoveOrder(o *types.Order) (*types.Order, error) {
	// first  we try to find the pricelevel of the order
	var i int
	if o.Side == types.SideBuy {
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price.GTE(o.Price) })
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price.LTE(o.Price) })
	}
	// we did not found the level
	// then the order do not exists in the price level
	if i >= len(s.levels) {
		return nil, types.ErrOrderNotFound
	}

	// now we may have a few orders with the same timestamp
	// lets iterate over them in order to find the right one
	finaloidx := -1
	for index, order := range s.levels[i].orders {
		if order.ID == o.ID {
			finaloidx = index
			break
		}
	}

	var order *types.Order
	// remove the order from the
	if finaloidx != -1 {
		order = s.levels[i].orders[finaloidx]
		s.levels[i].removeOrder(finaloidx)
	} else {
		// We could not find the matching order, return an error
		return nil, types.ErrOrderNotFound
	}

	if len(s.levels[i].orders) <= 0 {
		s.levels = s.levels[:i+copy(s.levels[i:], s.levels[i+1:])]
	}

	return order, nil
}

func (s *OrderBookSide) getPriceLevelIfExists(price *num.Uint) *PriceLevel {
	var i int
	if s.side == types.SideBuy {
		// buy side levels should be ordered in ascending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price.GTE(price) })
	} else {
		// sell side levels should be ordered in descending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price.LTE(price) })
	}

	// we found the level just return it.
	if i < len(s.levels) && s.levels[i].price.EQ(price) {
		return s.levels[i]
	}
	return nil
}

func (s *OrderBookSide) getPriceLevel(price *num.Uint) *PriceLevel {
	var i int
	if s.side == types.SideBuy {
		// buy side levels should be ordered in ascending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price.GTE(price) })
	} else {
		// sell side levels should be ordered in descending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price.LTE(price) })
	}

	// we found the level just return it.
	if i < len(s.levels) && s.levels[i].price.EQ(price) {
		return s.levels[i]
	}

	// append new elem first to make sure we have enough place
	// this would reallocate sufficiently then
	// no risk of this being a empty order, as it's overwritten just next with
	// the slice insert
	level := NewPriceLevel(price.Clone())
	s.levels = append(s.levels, nil)
	copy(s.levels[i+1:], s.levels[i:])
	s.levels[i] = level
	return level
}

func (s *OrderBookSide) getLevelsForPrice(price *num.Uint) []*PriceLevel {
	ret := make([]*PriceLevel, 0, len(s.levels))
	// buy is ASCENDING, start at the highest buy price until we find a buy order that will not trade
	// at the given price (ie given price is > buy order price).
	cmpF := price.GT
	if s.side == types.SideSell {
		// sell is DESCENDING, start at the lowest sell order price, until we find a sell order that won't trade
		// at the given price (ie given price < sell order price).
		cmpF = price.LT
	}
	for i := len(s.levels) - 1; i >= 0; i-- {
		if cmpF(s.levels[i].price) {
			return ret
		}
		ret = append(ret, s.levels[i])
	}
	return ret
}

// GetVolume returns the volume at the given pricelevel.
func (s *OrderBookSide) GetVolume(price *num.Uint) (uint64, error) {
	priceLevel := s.getPriceLevelIfExists(price)

	if priceLevel == nil {
		return 0, ErrPriceNotFound
	}

	return priceLevel.volume, nil
}

// fakeUncross returns hypothetical trades if the order book side were to be uncrossed with the agg order supplied,
// checkWashTrades checks non-FOK orders for wash trades if set to true (FOK orders are always checked for wash trades).
func (s *OrderBookSide) fakeUncross(agg *types.Order, checkWashTrades bool) ([]*types.Trade, error) {
	var (
		trades            []*types.Trade
		totalVolumeToFill uint64
	)
	if agg.TimeInForce == types.OrderTimeInForceFOK {
		var checkPrice func(*num.Uint) bool
		if agg.Side == types.SideBuy {
			checkPrice = func(levelPrice *num.Uint) bool { return levelPrice.LTE(agg.Price) }
		} else {
			checkPrice = func(levelPrice *num.Uint) bool { return levelPrice.GTE(agg.Price) }
		}

		for i := len(s.levels) - 1; i >= 0; i-- {
			level := s.levels[i]
			// we don't have to account for network orders, they don't apply in price monitoring
			// nor do fees apply
			if checkPrice(level.price) || agg.Type == types.OrderTypeMarket {
				for _, order := range level.orders {
					if agg.Party == order.Party {
						return nil, ErrWashTrade
					}
					totalVolumeToFill += order.Remaining
					if totalVolumeToFill >= agg.Remaining {
						break
					}
				}
			}
			if totalVolumeToFill >= agg.Remaining {
				break
			}
		}

		// FOK order could not be filled
		if totalVolumeToFill < agg.Remaining {
			return nil, nil
		}
	}

	// get a copy of the order passed in, so we can rely on fakeUncross to do its job
	fake := agg.Clone()

	var (
		idx        = len(s.levels) - 1
		ntrades    []*types.Trade
		err        error
		checkPrice func(*num.Uint) bool
	)

	if fake.Side == types.SideBuy {
		checkPrice = func(levelPrice *num.Uint) bool { return levelPrice.GT(agg.Price) }
	} else {
		checkPrice = func(levelPrice *num.Uint) bool { return levelPrice.LT(agg.Price) }
	}

	// in here we iterate from the end, as it's easier to remove the
	// price levels from the back of the slice instead of from the front
	// also it will allow us to reduce allocations
	for idx >= 0 && fake.Remaining > 0 {
		// not a market order && buy side price is too low => break
		if agg.Type != types.OrderTypeMarket && checkPrice(s.levels[idx].price) {
			break
		}
		fake, ntrades, err = s.levels[idx].fakeUncross(fake, checkWashTrades)
		trades = append(trades, ntrades...)
		// break if a wash trade is detected
		if err != nil && err == ErrWashTrade {
			break
		}

		// the orders are still part of the levels, so we just have to move on anyway
		idx--
	}

	return trades, err
}

// fakeUncrossAuction returns hypothetical trades if the order book side were to be uncrossed with the agg orders supplied, wash trades are allowed.
func (s *OrderBookSide) fakeUncrossAuction(orders []*types.Order) ([]*types.Trade, error) {
	// in here we iterate from the end, as it's easier to remove the
	// price levels from the back of the slice instead of from the front
	// also it will allow us to reduce allocations
	nOrders := len(orders)
	if nOrders == 0 {
		return []*types.Trade{}, nil
	}

	checkPrice := func(levelPrice *num.Uint, order *types.Order) bool {
		if order.Side == types.SideBuy {
			return levelPrice.GT(order.Price)
		}
		return levelPrice.LT(order.Price)
	}

	var (
		ntrades []*types.Trade
		iOrder  = 0
		trades  []*types.Trade
		lvl     *PriceLevel
		err     error
	)

	fake := orders[iOrder].Clone()
	for idx := len(s.levels) - 1; idx >= 0; idx-- {
		// since all of uncrossOrders will be traded away and at the same uncrossing price
		// iceberg orders are sent in as their full value instead of refreshing at each step
		if fake.IcebergOrder != nil {
			fake.Remaining += fake.IcebergOrder.ReservedRemaining
			fake.IcebergOrder.ReservedRemaining = 0
		}

		// clone price level
		lvl = clonePriceLevel(s.levels[idx])
		for lvl.volume > 0 {
			// not a market order && buy side price is too low => continue
			if fake.Type != types.OrderTypeMarket && checkPrice(lvl.price, fake) {
				continue
			}

			_, ntrades, _, err = lvl.uncross(fake, false)
			if err != nil {
				return nil, err
			}
			trades = append(trades, ntrades...)
			if fake.Remaining == 0 {
				iOrder++
				if iOrder >= nOrders {
					return trades, nil
				}
				fake = orders[iOrder].Clone()
			}
		}
	}
	return trades, nil
}

func clonePriceLevel(lvl *PriceLevel) *PriceLevel {
	orders := make([]*types.Order, 0, len(lvl.orders))
	for _, o := range lvl.orders {
		orders = append(orders, o.Clone())
	}
	return &PriceLevel{
		price:  lvl.price.Clone(),
		orders: orders,
		volume: lvl.volume,
	}
}

// uncross returns trades after order book side gets uncrossed with the agg order supplied,
// checkWashTrades checks non-FOK orders for wash trades if set to true (FOK orders are always checked for wash trades).
func (s *OrderBookSide) uncross(agg *types.Order, checkWashTrades bool) ([]*types.Trade, []*types.Order, *num.Uint, error) {
	var (
		trades            []*types.Trade
		impactedOrders    []*types.Order
		lastTradedPrice   = num.UintZero()
		totalVolumeToFill uint64
		checkPrice        func(*num.Uint) bool
	)

	if agg.Side == types.SideSell {
		checkPrice = func(levelPrice *num.Uint) bool { return levelPrice.GTE(agg.Price) }
	} else {
		checkPrice = func(levelPrice *num.Uint) bool { return levelPrice.LTE(agg.Price) }
	}

	if agg.TimeInForce == types.OrderTimeInForceFOK {
		// Process these backwards
		for i := len(s.levels) - 1; i >= 0; i-- {
			level := s.levels[i]
			if checkPrice(level.price) || agg.Type == types.OrderTypeMarket || agg.Type == types.OrderTypeNetwork {
				// We have to process every order to check for wash trades
				for _, order := range level.orders {
					// Check for wash trading
					if agg.Party == order.Party {
						// Stop the order and return
						agg.Status = types.OrderStatusStopped
						return nil, nil, lastTradedPrice, ErrWashTrade
					}
					// in case of network trades, we want to calculate an accurate average price to return
					totalVolumeToFill += order.Remaining

					if totalVolumeToFill >= agg.Remaining {
						break
					}
				}
			}
			if totalVolumeToFill >= agg.Remaining {
				break
			}
		}

		if s.log.GetLevel() == logging.DebugLevel {
			s.log.Debug(fmt.Sprintf("totalVolumeToFill %d until price %d, remaining %d\n", totalVolumeToFill, agg.Price, agg.Remaining))
		}

		if totalVolumeToFill < agg.Remaining {
			return trades, impactedOrders, lastTradedPrice, nil
		}
	}

	var (
		idx     = len(s.levels) - 1
		filled  bool
		ntrades []*types.Trade
		nimpact []*types.Order
		err     error
	)

	// in here we iterate from the end, as it's easier to remove the
	// price levels from the back of the slice instead of from the front
	// also it will allow us to reduce allocations
	for !filled && idx >= 0 {
		if checkPrice(s.levels[idx].price) || agg.Type == types.OrderTypeMarket || agg.Type == types.OrderTypeNetwork {
			filled, ntrades, nimpact, err = s.levels[idx].uncross(agg, checkWashTrades)
			trades = append(trades, ntrades...)
			impactedOrders = append(impactedOrders, nimpact...)
			// break if a wash trade is detected
			if err != nil && err == ErrWashTrade {
				break
			}
			if len(s.levels[idx].orders) <= 0 {
				idx--
			}
		} else {
			break
		}
	}

	// now we nil the price levels that have been completely emptied out
	// then we resize the slice
	if idx < 0 || len(s.levels[idx].orders) > 0 {
		// do not remove this one as it's not emptied already
		idx++
	}
	if idx < len(s.levels) {
		// nil out the pricelevels so they get collected at some point
		for i := idx; i < len(s.levels); i++ {
			s.levels[i] = nil
		}
		s.levels = s.levels[:idx]
	}

	if agg.Type == types.OrderTypeNetwork {
		totalPrice := num.UintZero()
		for _, t := range trades {
			// totalPrice += t.Price * t.Size
			totalPrice.Add(
				totalPrice,
				num.UintZero().Mul(t.Price, num.NewUint(t.Size)),
			)
		}
		// now we are done with uncrossing,
		// we can set back the price of the netorder to the average
		// price over the whole volume
		// agg.Price = totalPrice / agg.Size
		agg.Price.Div(totalPrice, num.NewUint(agg.Size))
	}

	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].Price.Clone()
	}
	return trades, impactedOrders, lastTradedPrice, err
}

func (s *OrderBookSide) getLevels() []*PriceLevel {
	return s.levels
}

func (s *OrderBookSide) getOrderCount() int64 {
	var orderCount int64
	for _, level := range s.levels {
		orderCount = orderCount + int64(len(level.orders))
	}
	return orderCount
}

func (s *OrderBookSide) getTotalVolume() int64 {
	var volume int64
	for _, level := range s.levels {
		volume = volume + int64(level.volume)
	}
	return volume
}
