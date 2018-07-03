package matching

import (
	"vega/proto"
)

type OrderBookSide struct {
	levels []*PriceLevel
}

func (s *OrderBookSide) addOrder(o *msg.Order, side msg.Side) {
	s.getPriceLevel(o.Price, side).addOrder(o)
}

func (s *OrderBookSide) RemoveOrder(o *msg.Order) error {
	//for {
	//	// binary search `s`
	//	// idx = index in s
	//	idx := 0
	//	priceLevel := s[idx]
	//	if len()
	//}
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
				copy(priceLevel.orders[toRemove:], priceLevel.orders[toRemove+1:])
				priceLevel.orders = priceLevel.orders[:len(priceLevel.orders)-1]
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
	return nil
}

func (s *OrderBookSide) getPriceLevel(price uint64, side msg.Side) *PriceLevel {
	at := -1
	if side == msg.Side_Buy {
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
	} else {
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
	}
	level := NewPriceLevel(price)
	if at == -1 {
		s.levels = append(s.levels, level)
		return level
	}
	s.levels = append(s.levels[:at], append([]*PriceLevel{level}, s.levels[at:]...)...)
	return level
	//item := s.levels.Get(&PriceLevel{price: price})
	//if item == nil {
	//	priceLevel := NewPriceLevel(price)
	//	//log.Printf("creating new price level price=%d", priceLevel.price)
	//	s.levels.ReplaceOrInsert(priceLevel)
	//	return priceLevel
	//}
	//priceLevel := item.(*PriceLevel)
	////log.Printf("fetched price level price=%d with %d orders", priceLevel.price, len(priceLevel.orders))
	//return priceLevel
}

//func (s OrderBookSide) removePriceLevel(price uint64) {
//	s.levels.Delete(&PriceLevel{price: price})
//}

func (s *OrderBookSide) uncross(agg *msg.Order) ([]*msg.Trade, []*msg.Order, uint64) {

	var (
		trades          []*msg.Trade
		impactedOrders  []*msg.Order
		lastTradedPrice uint64
	)

	if agg.Side == msg.Side_Sell {
		for _, order := range s.levels {
			if order.price >= agg.Price {
				ntrades, nimpact := order.uncross(agg)
				trades = append(trades, ntrades...)
				impactedOrders = append(impactedOrders, nimpact...)
				break
			} else {
				break
			}
		}
	}

	if agg.Side == msg.Side_Buy {
		for _, order := range s.levels {
			if order.price <= agg.Price {
				ntrades, nimpact := order.uncross(agg)
				trades = append(trades, ntrades...)
				impactedOrders = append(impactedOrders, nimpact...)
				break
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
