package matching

import (
	"vega/msg"
	"github.com/pkg/errors"
	"fmt"
)

type OrderBookSide struct {
	levels []*PriceLevel
}

func (s *OrderBookSide) addOrder(o *msg.Order, side msg.Side) {
	s.getPriceLevel(o.Price, side).addOrder(o)
}

func (s *OrderBookSide) RemoveOrder(o *msg.Order) error {
	// TODO: implement binary search on the slice
	toDelete := -1
	for idx, priceLevel := range s.levels {
		if priceLevel.price == o.Price {
			toRemove := -1
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
	if toDelete == -1 {
		fmt.Println("order not found")
		return errors.New("order not found")
	}
	return nil
}

func (s *OrderBookSide) getPriceLevel(price uint64, side msg.Side) *PriceLevel {
	// TODO: implement binary search on the slice
	at := -1
	if side == msg.Side_Buy {
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
	level := NewPriceLevel(price)
	if at == -1 {
		s.levels = append(s.levels, level)
		return level
	}
	s.levels = append(s.levels[:at], append([]*PriceLevel{level}, s.levels[at:]...)...)
	return level
}

func (s *OrderBookSide) uncross(agg *msg.Order) ([]*msg.Trade, []*msg.Order, uint64) {

	var (
		trades          []*msg.Trade
		impactedOrders  []*msg.Order
		lastTradedPrice uint64
	)

	if agg.Side == msg.Side_Sell {
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

	if agg.Side == msg.Side_Buy {
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
