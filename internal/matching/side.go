package matching

import (
	"fmt"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/metrics"
	types "code.vegaprotocol.io/vega/proto"
)

type OrderBookSide struct {
	log *logging.Logger
	// Config
	levels      []*PriceLevel
	proRataMode bool
}

func (s *OrderBookSide) addOrder(o *types.Order, side types.Side) {
	// update the price-volume map
	s.getPriceLevel(o.Price, side).addOrder(o)
}

func (s *OrderBookSide) amendOrder(orderAmended *types.Order) error {
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

func (s *OrderBookSide) RemoveOrder(o *types.Order) error {
	//todo: use binary search of expiring price levels (https://gitlab.com/vega-protocol/trading-core/issues/132)
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
		return types.ErrOrderNotFound
	}
	return nil
}

func (s *OrderBookSide) getPriceLevel(price uint64, side types.Side) *PriceLevel {
	//todo: use binary search of price levels (gitlab.com/vega-protocol/trading-core/issues/90)
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
	level := NewPriceLevel(price, s.proRataMode)
	if at == -1 {
		s.levels = append(s.levels, level)
		return level
	}
	s.levels = append(s.levels[:at], append([]*PriceLevel{level}, s.levels[at:]...)...)
	return level
}

func (s *OrderBookSide) uncross(agg *types.Order) ([]*types.Trade, []*types.Order, uint64) {
	timer := metrics.NewTimeCounter("-", "matching", "OrderBookSide.uncross")

	var (
		trades                  []*types.Trade
		impactedOrders          []*types.Order
		lastTradedPrice         uint64
		totalVolumeToFill       uint64
		totalPrice, totalVolume uint64
	)

	if agg.TimeInForce == types.Order_FOK {
		totalVolume = agg.Remaining

		if agg.Side == types.Side_Sell {
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
		}

		if agg.Side == types.Side_Buy {
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
	timer.EngineTimeCounterAdd()
	return trades, impactedOrders, lastTradedPrice
}

func (s *OrderBookSide) getLevels() []*PriceLevel {
	return s.levels
}
