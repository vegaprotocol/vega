package matching

import (
	"fmt"

	"code.vegaprotocol.io/vega/internal/dto"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/shopspring/decimal"
)

type OrderBookSide struct {
	log *logging.Logger
	// Config
	levels      []*PriceLevel
	proRataMode bool
}

func (s *OrderBookSide) addOrder(o *dto.Order, side types.Side) {
	s.getPriceLevel(o.Price, side).addOrder(o)
}

func (s *OrderBookSide) amendOrder(orderAmended *dto.Order) error {
	priceLevelIndex := -1
	orderIndex := -1

	for idx, priceLevel := range s.levels {
		if priceLevel.price.Equal(orderAmended.Price) {
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

func (s *OrderBookSide) RemoveOrder(o *dto.Order) error {
	//todo: use binary search of expiring price levels (https://gitlab.com/vega-protocol/trading-core/issues/132)
	toDelete := -1
	toRemove := -1
	for idx, priceLevel := range s.levels {
		if priceLevel.price.Equal(o.Price) {
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

func (s *OrderBookSide) getPriceLevel(price decimal.Decimal, side types.Side) *PriceLevel {
	//todo: use binary search of price levels (gitlab.com/vega-protocol/trading-core/issues/90)
	at := -1
	if side == types.Side_Buy {
		// buy side levels should be ordered in descending
		for i, level := range s.levels {
			if level.price.GreaterThan(price) {
				continue
			}
			if level.price.Equal(price) {
				return level
			}
			at = i
			break
		}
	} else {
		// sell side levels should be ordered in ascending
		for i, level := range s.levels {
			if level.price.LessThan(price) {
				continue
			}
			if level.price.Equal(price) {
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

func (s *OrderBookSide) uncross(agg *dto.Order) ([]*types.Trade, []*dto.Order, decimal.Decimal) {

	var (
		trades            []*types.Trade
		impactedOrders    []*dto.Order
		lastTradedPrice   decimal.Decimal
		totalVolumeToFill uint64
	)

	if agg.Type == types.Order_FOK {

		if agg.Side == types.Side_Sell {
			for _, level := range s.levels {
				if level.price.GreaterThanOrEqual(agg.Price) {
					totalVolumeToFill += level.volume
				}
			}
		}

		if agg.Side == types.Side_Buy {
			for _, level := range s.levels {
				if level.price.LessThanOrEqual(agg.Price) {
					totalVolumeToFill += level.volume
				}
			}
		}

		s.log.Debug(fmt.Sprintf("totalVolumeToFill %d until price %d, remaining %d\n", totalVolumeToFill, agg.Price, agg.Remaining))

		if totalVolumeToFill <= agg.Remaining {
			return trades, impactedOrders, decimal.Decimal{}
		}
	}

	if agg.Side == types.Side_Sell {
		for _, level := range s.levels {
			// buy side levels are ordered descending
			if level.price.GreaterThanOrEqual(agg.Price) {
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
			if level.price.LessThanOrEqual(agg.Price) {
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
		lastTradedPrice = decimal.NewFromFloat(float64(trades[len(trades)-1].Price))
	}
	return trades, impactedOrders, lastTradedPrice
}

func (s *OrderBookSide) getLevels() []*PriceLevel {
	return s.levels
}
