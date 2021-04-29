package matching

import (
	"encoding/binary"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrPriceNotFound signals that a price was not found on the book side
	ErrPriceNotFound = errors.New("price-volume pair not found")
)

// OrderBookSide represent a side of the book, either Sell or Buy
type OrderBookSide struct {
	side types.Side
	log  *logging.Logger
	// Config
	levels []*PriceLevel
}

func (s *OrderBookSide) Hash() []byte {
	output := make([]byte, len(s.levels)*16)
	var i int
	for _, l := range s.levels {
		binary.BigEndian.PutUint64(output[i:], l.price)
		i += 8
		binary.BigEndian.PutUint64(output[i:], l.volume)
		i += 8
	}
	return crypto.Hash(output)
}

// When we leave an auction we need to remove any orders marked as GFA
func (s *OrderBookSide) getOrdersToCancel(auction bool) ([]*types.Order, error) {
	ordersToCancel := make([]*types.Order, 0)
	for _, pricelevel := range s.levels {
		for _, order := range pricelevel.orders {
			// Find orders to cancel
			if (order.TimeInForce == types.Order_TIME_IN_FORCE_GFA && !auction) ||
				(order.TimeInForce == types.Order_TIME_IN_FORCE_GFN && auction) {
				// Save order to send back to client
				ordersToCancel = append(ordersToCancel, order)
			}
		}
	}
	return ordersToCancel, nil
}

func (s *OrderBookSide) addOrder(o *types.Order) {
	// update the price-volume map
	s.getPriceLevel(o.Price).addOrder(o)
}

// BestPriceAndVolume returns the top of book price and volume
// returns an error if the book is empty
func (s *OrderBookSide) BestPriceAndVolume() (uint64, uint64, error) {
	if len(s.levels) <= 0 {
		return 0, 0, errors.New("no orders on the book")
	}
	last := len(s.levels) - 1
	return s.levels[last].price, s.levels[last].volume, nil
}

// BestStaticPrice returns the top of book price for non pegged orders
// We do not keep count of the volume which makes this slightly quicker
// returns an error if the book is empty
func (s *OrderBookSide) BestStaticPrice() (uint64, error) {
	if len(s.levels) <= 0 {
		return 0, errors.New("no orders on the book")
	}

	for i := len(s.levels) - 1; i >= 0; i-- {
		pricelevel := s.levels[i]
		for _, order := range pricelevel.orders {
			if order.PeggedOrder == nil {
				return pricelevel.price, nil
			}
		}
	}
	return 0, errors.New("no non pegged orders found on the book")
}

// BestStaticPriceAndVolume returns the top of book price for non pegged orders
// returns an error if the book is empty
func (s *OrderBookSide) BestStaticPriceAndVolume() (uint64, uint64, error) {
	if len(s.levels) <= 0 {
		return 0, 0, errors.New("no orders on the book")
	}

	var bestPrice uint64
	var bestVolume uint64
	for i := len(s.levels) - 1; i >= 0; i-- {
		pricelevel := s.levels[i]
		for _, order := range pricelevel.orders {
			if order.PeggedOrder == nil {
				bestPrice = pricelevel.price
				bestVolume += order.Remaining
			}
		}
		// If we found a price, return it
		if bestPrice > 0 {
			return bestPrice, bestVolume, nil
		}
	}
	return 0, 0, errors.New("no non pegged orders found on the book")
}

func (s *OrderBookSide) amendOrder(orderAmend *types.Order) (uint64, error) {
	priceLevelIndex := -1
	orderIndex := -1
	var oldOrder *types.Order

	for idx, priceLevel := range s.levels {
		if priceLevel.price == orderAmend.Price {
			priceLevelIndex = idx
			for j, order := range priceLevel.orders {
				if order.Id == orderAmend.Id {
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

	if oldOrder.PartyId != orderAmend.PartyId {
		return 0, types.ErrOrderAmendFailure
	}

	if oldOrder.Remaining < orderAmend.Size {
		return 0, types.ErrOrderAmendFailure
	}

	if oldOrder.Reference != orderAmend.Reference {
		return 0, types.ErrOrderAmendFailure
	}

	reduceBy := oldOrder.Remaining - orderAmend.Size
	*s.levels[priceLevelIndex].orders[orderIndex] = *orderAmend
	s.levels[priceLevelIndex].reduceVolume(reduceBy)
	return reduceBy, nil
}

// ExtractOrders removes the orders from the top of the book until the volume amount is hit
func (s *OrderBookSide) ExtractOrders(price, volume uint64) ([]*types.Order, error) {
	extractedOrders := []*types.Order{}

	var (
		totalVolume uint64
		checkPrice  func(uint64) bool
	)
	if s.side == types.Side_SIDE_BUY {
		checkPrice = func(orderPrice uint64) bool { return orderPrice >= price }
	} else {
		checkPrice = func(orderPrice uint64) bool { return orderPrice <= price }
	}

	for i := len(s.levels) - 1; i >= 0; i-- {
		pricelevel := s.levels[i]
		var toRemove int
		for _, order := range pricelevel.orders {
			// Check the price is good and the total volume will not be exceeded
			if checkPrice(order.Price) && totalVolume+order.Remaining <= volume {
				// Remove this order
				extractedOrders = append(extractedOrders, order)
				totalVolume += order.Remaining
				// Remove the order from the price level
				toRemove++

			} else {
				// We should never get to here unless the passed in price
				// and volume are not correct
				return nil, ErrInvalidVolume
			}

			// If we have the right amount, stop processing
			if totalVolume == volume {
				break
			}

		}
		for toRemove > 0 {
			toRemove--
			pricelevel.removeOrder(0)
		}
		// Erase this price level which will be at the end of the slice
		if len(pricelevel.orders) == 0 {
			s.levels[i] = nil
			s.levels = s.levels[:len(s.levels)-1]
		}

		// Check if we have done enough
		if totalVolume == volume {
			break
		}
	}
	// If we get here and don't have the full amount of volume
	// something has gone wrong
	if totalVolume != volume {
		return nil, ErrInvalidVolume
	}

	return extractedOrders, nil
}

// RemoveOrder will remove an order from the book
func (s *OrderBookSide) RemoveOrder(o *types.Order) (*types.Order, error) {
	// first  we try to find the pricelevel of the order
	var i int
	if o.Side == types.Side_SIDE_BUY {
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price >= o.Price })
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= o.Price })
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
		if order.Id == o.Id {
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

func (s *OrderBookSide) getPriceLevelIfExists(price uint64) *PriceLevel {
	var i int
	if s.side == types.Side_SIDE_BUY {
		// buy side levels should be ordered in ascending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price >= price })
	} else {
		// sell side levels should be ordered in descending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= price })
	}

	// we found the level just return it.
	if i < len(s.levels) && s.levels[i].price == price {
		return s.levels[i]
	}
	return nil
}

func (s *OrderBookSide) getPriceLevel(price uint64) *PriceLevel {
	var i int
	if s.side == types.Side_SIDE_BUY {
		// buy side levels should be ordered in descending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price >= price })
	} else {
		// sell side levels should be ordered in ascending
		i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= price })
	}

	// we found the level just return it.
	if i < len(s.levels) && s.levels[i].price == price {
		return s.levels[i]
	}

	// append new elem first to make sure we have enough place
	// this would reallocate sufficiently then
	// no risk of this being a empty order, as it's overwritten just next with
	// the slice insert
	level := NewPriceLevel(price)
	s.levels = append(s.levels, nil)
	copy(s.levels[i+1:], s.levels[i:])
	s.levels[i] = level
	return level
}

// GetVolume returns the volume at the given pricelevel
func (s *OrderBookSide) GetVolume(price uint64) (uint64, error) {
	priceLevel := s.getPriceLevelIfExists(price)

	if priceLevel == nil {
		return 0, ErrPriceNotFound
	}

	return priceLevel.volume, nil
}

func (s *OrderBookSide) fakeUncross(agg *types.Order) ([]*types.Trade, error) {
	var (
		trades            []*types.Trade
		totalVolumeToFill uint64
	)
	if agg.TimeInForce == types.Order_TIME_IN_FORCE_FOK {
		var checkPrice func(uint64) bool
		if agg.Side == types.Side_SIDE_BUY {
			checkPrice = func(levelPrice uint64) bool { return levelPrice <= agg.Price }
		} else {
			checkPrice = func(levelPrice uint64) bool { return levelPrice >= agg.Price }
		}

		for i := len(s.levels) - 1; i >= 0; i-- {
			level := s.levels[i]
			// we don't have to account for network orders, they don't apply in price monitoring
			// nor do fees apply
			if checkPrice(level.price) || agg.Type == types.Order_TYPE_MARKET {
				for _, order := range level.orders {
					if agg.PartyId == order.PartyId {
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
			return nil, ErrFOKNotFilled
		}
	}

	// get a copy of the order passed in, so we can rely on fakeUncross to do its job
	cpy := *agg
	fake := &cpy

	var (
		idx        = len(s.levels) - 1
		ntrades    []*types.Trade
		err        error
		checkPrice func(uint64) bool
	)

	if fake.Side == types.Side_SIDE_BUY {
		checkPrice = func(levelPrice uint64) bool { return levelPrice > agg.Price }
	} else {
		checkPrice = func(levelPrice uint64) bool { return levelPrice < agg.Price }
	}

	// in here we iterate from the end, as it's easier to remove the
	// price levels from the back of the slice instead of from the front
	// also it will allow us to reduce allocations
	for idx >= 0 && fake.Remaining > 0 {
		// not a market order && buy side price is too low => break
		if agg.Type != types.Order_TYPE_MARKET && checkPrice(s.levels[idx].price) {
			break
		}
		fake, ntrades, err = s.levels[idx].fakeUncross(fake)
		trades = append(trades, ntrades...)
		// break if a wash trade is detected
		if err != nil && err == ErrWashTrade {
			break
		}
		// the orders are still part of the levels, so we just have to move on anyway
		idx--
	}

	return trades, nil
}

func (s *OrderBookSide) uncross(agg *types.Order, checkWashTrades bool) ([]*types.Trade, []*types.Order, uint64, error) {
	timer := metrics.NewTimeCounter("-", "matching", "OrderBookSide.uncross")

	var (
		trades            []*types.Trade
		impactedOrders    []*types.Order
		lastTradedPrice   uint64
		totalVolumeToFill uint64
		checkPrice        func(uint64) bool
	)

	if agg.Side == types.Side_SIDE_SELL {
		checkPrice = func(levelPrice uint64) bool { return levelPrice >= agg.Price }
	} else {
		checkPrice = func(levelPrice uint64) bool { return levelPrice <= agg.Price }
	}

	if agg.TimeInForce == types.Order_TIME_IN_FORCE_FOK {
		// Process these backwards
		for i := len(s.levels) - 1; i >= 0; i-- {
			level := s.levels[i]
			if checkPrice(level.price) || agg.Type == types.Order_TYPE_MARKET || agg.Type == types.Order_TYPE_NETWORK {
				// We have to process every order to check for wash trades
				for _, order := range level.orders {
					// Check for wash trading
					if agg.PartyId == order.PartyId {
						// Stop the order and return
						agg.Status = types.Order_STATUS_STOPPED
						return nil, nil, 0, ErrWashTrade
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
			timer.EngineTimeCounterAdd()
			return trades, impactedOrders, 0, nil
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
		if checkPrice(s.levels[idx].price) || agg.Type == types.Order_TYPE_MARKET || agg.Type == types.Order_TYPE_NETWORK {
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

	if agg.Type == types.Order_TYPE_NETWORK {
		var totalPrice uint64
		for _, t := range trades {
			totalPrice += t.Price * t.Size
		}
		// now we are done with uncrossing,
		// we can set back the price of the netorder to the average
		// price over the whole volume
		agg.Price = totalPrice / agg.Size
	}

	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].Price
	}
	timer.EngineTimeCounterAdd()
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
