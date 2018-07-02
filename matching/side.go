package matching

import (
	"log"
	"vega/proto"

	"github.com/google/btree"
)

const priceLevelsBTreeDegree = 128

type OrderBookSide struct {
	side   msg.Side
	levels *btree.BTree
}

func (s *OrderBookSide) addOrder(o *msg.Order) {
	s.getPriceLevel(o.Price).addOrder(o)
}

func (s *OrderBookSide) RemoveOrder(o *msg.Order) error {
	priceLevel := s.levels.Get(&PriceLevel{price: o.Price}).(*PriceLevel)
	err := priceLevel.removeOrderFromPriceLevel(o)

	if len(priceLevel.orders) == 0 {
		s.removePriceLevel(priceLevel.price)
	}
	return err
}

func (s *OrderBookSide) getPriceLevel(price uint64) *PriceLevel {
	item := s.levels.Get(&PriceLevel{price: price})
	if item == nil {
		priceLevel := NewPriceLevel(price)
		log.Printf("creating new price level price=%d", priceLevel.price)
		s.levels.ReplaceOrInsert(priceLevel)
		return priceLevel
	}
	priceLevel := item.(*PriceLevel)
	log.Printf("fetched price level price=%d with %d orders", priceLevel.price, len(priceLevel.orders))
	return priceLevel
}

func (s *OrderBookSide) removePriceLevel(price uint64) {
	s.levels.Delete(&PriceLevel{price: price})
}

func (s *OrderBookSide) uncross(agg *msg.Order) ([]Trade, []msg.Order, uint64) {
	trades := make([]Trade, 0)
	impactedOrders := make([]msg.Order, 0)
	var lastTradedPrice uint64

	if agg.Side == msg.Side_Sell {
		min := &PriceLevel{price: agg.Price - 1}
		log.Printf("uncross initiated | DescendRange from 1000 to min=%d ", min.price)
		s.levels.DescendGreaterThan(min, uncrossPriceLevel(agg, &trades, &impactedOrders))
	}

	if agg.Side == msg.Side_Buy {
		max := &PriceLevel{price: agg.Price + 1}
		log.Printf("uncross initiated | AscendRange 0 to max=%d", max.price)
		s.levels.AscendLessThan(max, uncrossPriceLevel(agg, &trades, &impactedOrders))
	}

	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].price
	}
	return trades, impactedOrders, lastTradedPrice
}

func uncrossPriceLevel(agg *msg.Order, trades *[]Trade, impactedOrders *[]msg.Order) func(i btree.Item) bool {
	return func(i btree.Item) bool {
		priceLevel := i.(*PriceLevel)
		filled := priceLevel.uncross(agg, trades, impactedOrders)
		return !filled
	}
}
