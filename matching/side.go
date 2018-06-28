package matching

import (
	"log"

	"vega/proto"

	"github.com/google/btree"
)

const priceLevelsBTreeDegree = 128

type OrderBookSide struct {
	side        msg.Side
	levels      *btree.BTree
	orderCount  uint64
	totalVolume uint64
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

func (s *OrderBookSide) getNumberOfPriceLevels() int {
	return s.levels.Len()
}

func (s *OrderBookSide) getOrderCount() uint64 {
	return s.orderCount
}

func (s *OrderBookSide) getTotalVolume() uint64 {
	return s.totalVolume
}

func (s *OrderBookSide) removePriceLevel(price uint64) {
	s.levels.Delete(&PriceLevel{price: price})
}

func (s *OrderBookSide) topPriceLevel() *PriceLevel {
	if s.levels.Len() > 0 {
		return s.levels.Max().(*PriceLevel)
	} else {
		return nil
	}
}

func (s *OrderBookSide) bestPrice() uint64 {
	if s.topPriceLevel() == nil {
		return 0
	} else {
		return s.topPriceLevel().price
	}
}

func uncrossPriceLevel(agg *msg.Order, trades *[]Trade) func(i btree.Item) bool {
	return func(i btree.Item) bool {
		priceLevel := i.(*PriceLevel)
		log.Println("dupa price level ", priceLevel.price)
		filled := priceLevel.uncross(agg, trades)
		return !filled
	}
}

//func (s *OrderBookSide) pivotPriceLevel(agg *msg.Order) *PriceLevel {
//	if s.side == msg.Side_Buy {
//		return &PriceLevel{price: agg.Price - 1}
//	} else {
//		return &PriceLevel{price: agg.Price + 1}
//	}
//}


func (s *OrderBookSide) cross(agg *msg.Order) (*[]Trade, uint64) {
	trades := make([]Trade, 0)
	var lastTradedPrice uint64

	log.Println("order side: ", agg.Side)
	log.Println("book side:", s.side)

	if agg.Side == msg.Side_Sell {

		min := &PriceLevel{price: agg.Price-1}
		log.Printf("uncross initiated | DescendRange from 1000 to min=%d ", min.price)
		log.Println()

		s.levels.DescendGreaterThan(min, uncrossPriceLevel(agg, &trades))
	}

	if agg.Side == msg.Side_Buy {

		max := &PriceLevel{price: agg.Price+1}
		log.Printf("uncross initiated | AscendRange 0 to max=%d", max.price)
		log.Println()
		s.levels.AscendLessThan(max, uncrossPriceLevel(agg, &trades))
	}


	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].price
	}

	for _, trade := range trades {
		s.totalVolume += trade.size
	}

	return &trades, lastTradedPrice
}

func (s *OrderBookSide) addOrder(o *msg.Order) {
	fetchedPriceLevel := s.getPriceLevel(o.Price)
	log.Println("blabla1 ", fetchedPriceLevel.orders)
	fetchedPriceLevel.addOrder(o)

	s.totalVolume += o.Remaining
	s.orderCount++
	log.Println("level ", fetchedPriceLevel.price)
	log.Println("blabla2 ", fetchedPriceLevel.orders)
}

func (s *OrderBookSide) RemoveOrder(o *msg.Order) {
	priceLevel := s.getPriceLevel(o.Price)
	priceLevel.removeOrder(o)
	s.totalVolume -= o.Remaining
	s.orderCount--

	if len(priceLevel.orders) == 0 {
		s.removePriceLevel(priceLevel.price)
	}
}
