package matching

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type OrderBookSide struct {
	log *logging.Logger
	// Config
	levels      []*PriceLevel
	proRataMode bool
}

func (s *OrderBookSide) addOrder(o *types.Order, side types.Side) {
	s.getPriceLevel(o.Price, side).addOrder(o)
}

func (s *OrderBookSide) amendOrder(orderAmended *types.Order) error {
	priceLevelIndex := -1
	orderIndex := -1

	i := sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= orderAmended.Price })
	if i < len(s.levels) && s.levels[i].price == orderAmended.Price {
		priceLevelIndex = i
		for j, order := range s.levels[i].orders {
			if order.Id == orderAmended.Id {
				orderIndex = j
				break
			}
		}
	}

	if priceLevelIndex == -1 || orderIndex == -1 {
		return types.ErrOrderNotFound
	}

	if s.levels[priceLevelIndex].orders[orderIndex].PartyID != orderAmended.PartyID {
		return types.ErrOrderAmendFailure
	}

	if s.levels[priceLevelIndex].orders[orderIndex].Size < orderAmended.Size {
		return types.ErrOrderAmendFailure
	}

	if s.levels[priceLevelIndex].orders[orderIndex].Reference != orderAmended.Reference {
		return types.ErrOrderAmendFailure
	}

	s.levels[priceLevelIndex].orders[orderIndex] = orderAmended
	return nil
}

func (s *OrderBookSide) RemoveOrder(o *types.Order) error {

	toDelete := -1
	toRemove := -1
	i := sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= o.Price })
	if i < len(s.levels) && s.levels[i].price == o.Price {
		for j, order := range s.levels[i].orders {
			if order.Id == o.Id {
				toRemove = j
				break
			}
		}
		if toRemove != -1 {
			s.levels[i].removeOrder(toRemove)
		}
		if len(s.levels[i].orders) == 0 {
			toDelete = i
		}
	}

	if toDelete != -1 {
		copy(s.levels[toDelete:], s.levels[toDelete+1:])
		s.levels = s.levels[:len(s.levels)-1]

	}
	if toRemove == -1 {
		return types.ErrOrderNotFound
	}
	return nil
}

func (s *OrderBookSide) getPriceLevel(price uint64, side types.Side) *PriceLevel {
	var at int
	if side == types.Side_Buy {
		at = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price <= price })
		if at < len(s.levels) && s.levels[at].price == price {
			return s.levels[at]
		}
	} else {
		at = sort.Search(len(s.levels), func(i int) bool { return s.levels[i].price >= price })
		if at < len(s.levels) && s.levels[at].price == price {
			return s.levels[at]
		}
	}

	level := NewPriceLevel(price, s.proRataMode)
	if len(s.levels) <= 0 {
		s.levels = append(s.levels, level)
	} else {
		s.levels = append(s.levels, &PriceLevel{})
		copy(s.levels[at+1:], s.levels[at:])
		s.levels[at] = level
	}

	return level
}

func (s *OrderBookSide) uncross(agg *types.Order) ([]*types.Trade, []*types.Order, uint64) {

	var (
		trades            []*types.Trade
		impactedOrders    []*types.Order
		lastTradedPrice   uint64
		totalVolumeToFill uint64
	)

	if agg.Type == types.Order_FOK {

		if agg.Side == types.Side_Sell {
			for _, level := range s.levels {
				if level.price >= agg.Price {
					totalVolumeToFill += level.volume
				}
			}
		}

		if agg.Side == types.Side_Buy {
			for _, level := range s.levels {
				if level.price <= agg.Price {
					totalVolumeToFill += level.volume
				}
			}
		}

		s.log.Debug(fmt.Sprintf("totalVolumeToFill %d until price %d, remaining %d\n", totalVolumeToFill, agg.Price, agg.Remaining))

		if totalVolumeToFill <= agg.Remaining {
			return trades, impactedOrders, 0
		}
	}

	if agg.Side == types.Side_Sell {
		for _, level := range s.levels {
			// buy side levels are ordered descending
			if level.price >= agg.Price {
				filled, nTrades, nImpact := level.uncross(agg)
				trades = append(trades, nTrades...)
				impactedOrders = append(impactedOrders, nImpact...)
				if filled {
					break
				}
			} else {
				break
			}
		}
	}

	if agg.Side == types.Side_Buy {
		for _, level := range s.levels {
			// sell side levels are ordered ascending
			if level.price <= agg.Price {
				filled, nTrades, nImpact := level.uncross(agg)
				trades = append(trades, nTrades...)
				impactedOrders = append(impactedOrders, nImpact...)
				if filled {
					break
				}
			} else {
				break
			}
		}
	}

	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].Price
	}
	return trades, impactedOrders, lastTradedPrice
}

func (s *OrderBookSide) getLevels() []*PriceLevel {
	return s.levels
}
