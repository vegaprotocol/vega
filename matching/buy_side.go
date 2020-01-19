package matching

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
)

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

func (b *BuySide) GetCloseoutPrice(volume uint64) (uint64, error) {
	var (
		vol    uint64        = volume
		levels []*PriceLevel = b.levels
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
	// if we reach this point, chances are vol != 0, in which case we should return an error along with the price
	if vol != 0 {
		err = ErrNotEnoughOrders
		// TODO(jeremy): there's no orders in the book so return the markPrice
		// this is a temporary fix for nice-net and this behaviour will need
		// to be properly specified and handled in the future.
		if vol == volume {
			// no orders at all on the book side
			err = ErrNoOrder
			return 0, err
		}
	}
	return price / (volume - vol), err

}

func (b *BuySide) GetHighestOrderPrice() (uint64, error) {
	if len(b.levels) <= 0 {
		return 0, ErrNoOrder
	}
	// buy order ascending
	return b.levels[len(b.levels)-1].price, nil
}

func (b *BuySide) GetLowestOrderPrice() (uint64, error) {
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

func (s *BuySide) Uncross(agg *types.Order) ([]*types.Trade, []*types.Order, uint64) {
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
