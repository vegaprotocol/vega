package matching

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	// ErrPriceNotFound signals that a price was not found on the book side
	ErrPriceNotFound = errors.New("price-volume pair not found")
	// ErrNoOrder signals that there's no orders on the book side.
	ErrNoOrder = errors.New("no orders in the book side")
)

type baseSide struct {
	log    *logging.Logger
	levels []*PriceLevel
}

func (s *baseSide) getLevels() []*PriceLevel {
	return s.levels
}

func (s baseSide) BestPriceAndVolume(side types.Side) (uint64, uint64) {
	if len(s.levels) <= 0 {
		return 0, 0
	}
	return s.levels[len(s.levels)-1].price, s.levels[len(s.levels)-1].volume
}

func (s *baseSide) amendOrder(orderAmended *types.Order) error {
	priceLevelIndex := -1
	orderIndex := -1
	var oldOrder *types.Order

	for idx, priceLevel := range s.levels {
		if priceLevel.price == orderAmended.Price {
			priceLevelIndex = idx
			for j, order := range priceLevel.orders {
				if order.Id == orderAmended.Id {
					orderIndex = j
					oldOrder = order
					break
				}
			}
			break
		}
	}

	if oldOrder == nil || priceLevelIndex == -1 || orderIndex == -1 {
		return types.ErrOrderNotFound
	}

	if oldOrder.PartyID != orderAmended.PartyID {
		return types.ErrOrderAmendFailure
	}

	if oldOrder.Size < orderAmended.Size {
		return types.ErrOrderAmendFailure
	}

	if oldOrder.Reference != orderAmended.Reference {
		return types.ErrOrderAmendFailure
	}

	s.levels[priceLevelIndex].orders[orderIndex] = orderAmended
	return nil
}

// RemoveOrder will remove an order from the book
func (s *baseSide) removeOrderAtPriceLevelIndex(i int, o *types.Order) (*types.Order, error) {
	// we did not found the level
	// then the order do not exists in the price level
	if i >= len(s.levels) {
		return nil, types.ErrOrderNotFound
	}

	// orders are order by timestamp (CreatedAt)
	oidx := sort.Search(len(s.levels[i].orders), func(j int) bool {
		return s.levels[i].orders[j].CreatedAt >= o.CreatedAt
	})
	// we did not find the order
	if oidx >= len(s.levels[i].orders) {
		return nil, types.ErrOrderNotFound
	}

	// now we may have a few orders with the same timestamp
	// lets iterate over them in order to find the right one
	finaloidx := -1
	for oidx < len(s.levels[i].orders) && s.levels[i].orders[oidx].CreatedAt == o.CreatedAt {
		if s.levels[i].orders[oidx].Id == o.Id {
			finaloidx = oidx
			break
		}
		oidx++
	}

	var order *types.Order
	// remove the order from the
	if finaloidx != -1 {
		order = s.levels[i].orders[finaloidx]
		s.levels[i].removeOrder(finaloidx)
	}

	if len(s.levels[i].orders) <= 0 {
		s.levels = s.levels[:i+copy(s.levels[i:], s.levels[i+1:])]
	}

	return order, nil
}

type SellSide struct {
	baseSide
}

func NewSellSide(log *logging.Logger) *SellSide {
	return &SellSide{
		baseSide{
			log:    log,
			levels: []*PriceLevel{},
		},
	}
}

func (s *SellSide) AddOrder(o *types.Order) {
	s.getPriceLevel(o.Price).addOrder(o)
}

func (s *SellSide) getPriceLevel(price uint64) *PriceLevel {
	var i int
	// sell side levels should be ordered in ascending
	i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= price })

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

func (s *SellSide) getHighestOrderPrice() (uint64, error) {
	if len(s.levels) <= 0 {
		return 0, ErrNoOrder
	}
	// sell order descending
	return s.levels[0].price, nil
}

func (s *SellSide) getLowestOrderPrice() (uint64, error) {
	if len(s.levels) <= 0 {
		return 0, ErrNoOrder
	}
	// sell order descending
	return s.levels[len(s.levels)-1].price, nil
}

// RemoveOrder will remove an order from the book
func (s *SellSide) RemoveOrder(o *types.Order) (*types.Order, error) {
	// first  we try to find the pricelevel of the order
	var i int
	// sell side levels should be ordered in ascending
	i = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= o.Price })
	// we did not found the level
	// then the order do not exists in the price level
	if i >= len(s.levels) {
		return nil, types.ErrOrderNotFound
	}

	return s.baseSide.removeOrderAtPriceLevelIndex(i, o)
}

func (s *SellSide) uncross(agg *types.Order) ([]*types.Trade, []*types.Order, uint64) {
	timer := metrics.NewTimeCounter("-", "matching", "SellSide.uncross")

	var (
		trades                  []*types.Trade
		impactedOrders          []*types.Order
		lastTradedPrice         uint64
		totalVolumeToFill       uint64
		totalPrice, totalVolume uint64
	)

	if agg.TimeInForce == types.Order_FOK {
		totalVolume = agg.Remaining

		for _, level := range s.levels {
			// in case of network trades, we want to calculate an accurate average price to return
			if agg.Type == types.Order_NETWORK {
				totalVolumeToFill += level.volume
				factor := totalVolume
				if level.volume < totalVolume {
					factor = level.volume
					totalVolume -= level.volume
				}
				totalPrice += level.price * factor
			} else if level.price <= agg.Price {
				totalVolumeToFill += level.volume
			}
		}

		if agg.Type == types.Order_NETWORK {
			// set avg price for order
			agg.Price = totalPrice / agg.Remaining
		}

		if s.log.GetLevel() == logging.DebugLevel {
			s.log.Debug(fmt.Sprintf("totalVolumeToFill %d until price %d, remaining %d\n", totalVolumeToFill, agg.Price, agg.Remaining))
		}

		if totalVolumeToFill <= agg.Remaining {
			timer.EngineTimeCounterAdd()
			return trades, impactedOrders, 0
		}
	}

	var (
		idx     = len(s.levels) - 1
		filled  bool
		ntrades []*types.Trade
		nimpact []*types.Order
	)

	// in here we iterate from the end, as it's easier to remove the
	// price levels from the back of the slice instead of from the front
	// also it will allow us to reduce allocations
	for !filled && idx >= 0 {
		if s.levels[idx].price <= agg.Price {
			filled, ntrades, nimpact = s.levels[idx].uncross(agg)
			trades = append(trades, ntrades...)
			impactedOrders = append(impactedOrders, nimpact...)
			if len(s.levels[idx].orders) <= 0 {
				idx--
			}
		} else {
			break
		}

	}

	// now we nil the price levels that have been completely emptied out
	// then we resize the slice
	// idx can be < to 0 if we went through all price levels
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

	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].Price
	}
	timer.EngineTimeCounterAdd()
	return trades, impactedOrders, lastTradedPrice
}

type BuySide struct {
	baseSide
}

func NewBuySide(log *logging.Logger) *BuySide {
	return &BuySide{
		baseSide{
			log:    log,
			levels: []*PriceLevel{},
		},
	}
}

func (b *BuySide) AddOrder(o *types.Order) {
	b.getPriceLevel(o.Price).addOrder(o)
}

func (b *BuySide) getPriceLevel(price uint64) *PriceLevel {
	var i int
	i = sort.Search(len(b.levels), func(i int) bool { return b.levels[i].price >= price })

	// we found the level just return it.
	if i < len(b.levels) && b.levels[i].price == price {
		return b.levels[i]
	}

	// append new elem first to make sure we have enough place
	// this would reallocate sufficiently then
	// no risk of this being a empty order, as it's overwritten just next with
	// the slice insert
	level := NewPriceLevel(price)
	b.levels = append(b.levels, nil)
	copy(b.levels[i+1:], b.levels[i:])
	b.levels[i] = level
	return level
}

func (b *BuySide) getHighestOrderPrice() (uint64, error) {
	if len(b.levels) <= 0 {
		return 0, ErrNoOrder
	}
	// buy order ascending
	return b.levels[len(b.levels)-1].price, nil
}

func (b *BuySide) getLowestOrderPrice() (uint64, error) {
	if len(b.levels) <= 0 {
		return 0, ErrNoOrder
	}
	// buy order ascending
	return b.levels[0].price, nil
}

// RemoveOrder will remove an order from the book
func (b *BuySide) RemoveOrder(o *types.Order) (*types.Order, error) {
	// first  we try to find the pricelevel of the order
	var i int
	i = sort.Search(len(b.levels), func(i int) bool { return b.levels[i].price >= o.Price })

	// we did not found the level
	// then the order do not exists in the price level
	if i >= len(b.levels) {
		return nil, types.ErrOrderNotFound
	}

	return b.baseSide.removeOrderAtPriceLevelIndex(i, o)

}

func (s *BuySide) uncross(agg *types.Order) ([]*types.Trade, []*types.Order, uint64) {
	timer := metrics.NewTimeCounter("-", "matching", "BuySide.uncross")

	var (
		trades                  []*types.Trade
		impactedOrders          []*types.Order
		lastTradedPrice         uint64
		totalVolumeToFill       uint64
		totalPrice, totalVolume uint64
	)

	if agg.TimeInForce == types.Order_FOK {
		totalVolume = agg.Remaining

		for _, level := range s.levels {
			// in case of network trades, we want to calculate an accurate average price to return
			if agg.Type == types.Order_NETWORK {
				totalVolumeToFill += level.volume
				factor := totalVolume
				if level.volume < totalVolume {
					factor = level.volume
					totalVolume -= level.volume
				}
				totalPrice += level.price * factor
			} else if level.price >= agg.Price {
				totalVolumeToFill += level.volume
			}
		}

		if agg.Type == types.Order_NETWORK {
			// set avg price for order
			agg.Price = totalPrice / agg.Remaining
		}

		if s.log.GetLevel() == logging.DebugLevel {
			s.log.Debug(fmt.Sprintf("totalVolumeToFill %d until price %d, remaining %d\n", totalVolumeToFill, agg.Price, agg.Remaining))
		}

		if totalVolumeToFill <= agg.Remaining {
			timer.EngineTimeCounterAdd()
			return trades, impactedOrders, 0
		}
	}

	var (
		idx     = len(s.levels) - 1
		filled  bool
		ntrades []*types.Trade
		nimpact []*types.Order
	)

	// in here we iterate from the end, as it's easier to remove the
	// price levels from the back of the slice instead of from the front
	// also it will allow us to reduce allocations
	for !filled && idx >= 0 {
		if s.levels[idx].price >= agg.Price {
			filled, ntrades, nimpact = s.levels[idx].uncross(agg)
			trades = append(trades, ntrades...)
			impactedOrders = append(impactedOrders, nimpact...)
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

	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].Price
	}
	timer.EngineTimeCounterAdd()
	return trades, impactedOrders, lastTradedPrice
}
