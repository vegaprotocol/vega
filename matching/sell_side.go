package matching

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
)

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

func (s *SellSide) GetCloseoutPrice(volume uint64) (uint64, error) {
	var (
		vol    uint64        = volume
		levels []*PriceLevel = s.levels
		price  uint64
		err    error
	)
	for i := len(levels) - 1; i >= 0; i-- {
		lvl := levels[i]
		if lvl.volume >= vol {
			price += lvl.price * vol
			return price / volume, err
		}
		price += lvl.price * lvl.volume
		vol -= lvl.volume
	}
	// at this point, we should check vol, make sure it's 0, if not return an error to indicate something is wrong
	// still return the price for the volume we could close out, so the caller can make a decision on what to do
	if vol != 0 {
		err = ErrNotEnoughOrders
		// TODO(jeremy): there's no orders in the book so return the markPrice
		// this is a temporary fix for nicenet and this behaviour will need
		// to be properaly specified and handled in the future.
		if vol == volume {
			// no orders at all
			err = ErrNoOrder
			return 0, err
		}
	}
	return price / (volume - vol), err
}

func (s *SellSide) GetHighestOrderPrice() (uint64, error) {
	if len(s.levels) <= 0 {
		return 0, ErrNoOrder
	}
	// sell order descending
	return s.levels[0].price, nil
}

func (s *SellSide) GetLowestOrderPrice() (uint64, error) {
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

func (s *SellSide) Uncross(agg *types.Order) ([]*types.Trade, []*types.Order, uint64) {
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
