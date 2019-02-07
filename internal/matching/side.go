package matching

import (
	types "vega/proto"

	"github.com/pkg/errors"
)

type OrderBookSide struct {
	*Config
	levels      []*PriceLevel
}

func (s *OrderBookSide) addOrder(o *types.Order, side types.Side) {
	s.getPriceLevel(o.Price, side).addOrder(o)
}

func (s *OrderBookSide) amendOrder(orderAmended *types.Order) types.OrderError {
	priceLevelIndex := -1
	orderIndex := -1

	for idx, priceLevel := range s.levels {
		if priceLevel.price == orderAmended.Price {
			priceLevelIndex = idx
			for j, order := range priceLevel.orders {
				if order.Id == orderAmended.Id {
					orderIndex = j
					break
				}
			}
			break
		}
	}

	if priceLevelIndex == -1 || orderIndex == -1 {
		return types.OrderError_ORDER_NOT_FOUND
	}

	if s.levels[priceLevelIndex].orders[orderIndex].Party != orderAmended.Party {
		return types.OrderError_ORDER_AMEND_FAILURE
	}

	if s.levels[priceLevelIndex].orders[orderIndex].Size < orderAmended.Size {
		return types.OrderError_ORDER_AMEND_FAILURE
	}

	if s.levels[priceLevelIndex].orders[orderIndex].Reference != orderAmended.Reference {
		return types.OrderError_ORDER_AMEND_FAILURE
	}

	s.levels[priceLevelIndex].orders[orderIndex] = orderAmended
	return types.OrderError_NONE
}

func (s *OrderBookSide) RemoveOrder(o *types.Order) error {
	// TODO: implement binary search on the slice
	toDelete := -1
	toRemove := -1
	for idx, priceLevel := range s.levels {
		if priceLevel.price == o.Price {
			for j, order := range priceLevel.orders {
				if order.Id == o.Id {
					toRemove = j
					break
				}
			}
			if toRemove != -1 {
				priceLevel.removeOrder(toRemove)
			}
			if len(priceLevel.orders) == 0 {
				toDelete = idx
			}
			break
		}
	}
	if toDelete != -1 {
		copy(s.levels[toDelete:], s.levels[toDelete+1:])
		s.levels = s.levels[:len(s.levels)-1]

	}
	if toRemove == -1 {
		return errors.New("order not found")
	}
	return nil
}

func (s *OrderBookSide) getPriceLevel(price uint64, side types.Side) *PriceLevel {
	// TODO: implement binary search on the slice
	at := -1
	if side == types.Side_Buy {
		// buy side levels should be ordered in descending
		for i, level := range s.levels {
			if level.price > price {
				continue
			}
			if level.price == price {
				return level
			}
			at = i
			break
		}
	} else {
		// sell side levels should be ordered in ascending
		for i, level := range s.levels {
			if level.price < price {
				continue
			}
			if level.price == price {
				return level
			}
			at = i
			break
		}
	}
	level := NewPriceLevel(s.Config, price)
	if at == -1 {
		s.levels = append(s.levels, level)
		return level
	}
	s.levels = append(s.levels[:at], append([]*PriceLevel{level}, s.levels[at:]...)...)
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

		s.log.Debugf("totalVolumeToFill %d until price %d, remaining %d\n", totalVolumeToFill, agg.Price, agg.Remaining)

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
