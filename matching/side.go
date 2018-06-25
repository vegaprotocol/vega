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

func newSide(side msg.Side) *OrderBookSide {
	return &OrderBookSide{
		side:   side,
		levels: btree.New(priceLevelsBTreeDegree),
	}
}

func (s *OrderBookSide) getPriceLevel(price uint64) *PriceLevel {
	var priceLevel *PriceLevel
	item := s.levels.Get(&PriceLevel{price: price})
	if item == nil {
		priceLevel = NewPriceLevel(s, price)
		log.Println("creating new price level :", priceLevel.price)
		s.levels.ReplaceOrInsert(priceLevel)
	} else {
		priceLevel = item.(*PriceLevel)
		log.Println("fetched price level :", priceLevel.price)
		log.Println("number of orders :", priceLevel.orders.Len())
	}
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

func (s *OrderBookSide) pivotPriceLevel(agg *OrderEntry) *PriceLevel {
	if s.side == msg.Side_Buy {
		return &PriceLevel{price: agg.order.Price - 1}
	} else {
		return &PriceLevel{price: agg.order.Price + 1}
	}
}

func (s *OrderBookSide) bestPrice() uint64 {
	if s.topPriceLevel() == nil {
		return 0
	} else {
		return s.topPriceLevel().price
	}
}

//func (s *OrderBookSide) addOrder(o *OrderEntry) *[]Trade {
//	log.Printf("%d of levels on side %s", s.other.levels.Len(), s.other.side)
//	trades := s.other.uncross(o)
//	if o.persist && o.order.Remaining > 0 {
//		s.book.orders[o.order.Id] = o
//		o.book = s.book
//		o.side = s
//		o.priceLevel = s.getPriceLevel(o.order.Price)
//		o.priceLevel.addOrder(o)
//		s.getPriceLevel(o.order.Price).addOrder(o)
//		if !s.book.config.Quiet {
//			log.Printf("Added: %v\n", o)
//		}
//	}
//	return trades
//}

// Go through the price levels from best to worst uncrossing each in turn
//func (s *OrderBookSide) uncross(agg *OrderEntry) *[]Trade {
//	trades := make([]Trade, 0)
//	s.levels.DescendGreaterThan(s.pivotPriceLevel(agg), uncrossPriceLevel(agg, &trades))
//	if len(trades) > 0 {
//		s.book.lastTradedPrice = trades[len(trades)-1].price
//	}
//	return &trades
//}

// Returns closure over the aggressor and trade slice that calls priceLevel.uncross(...)
func uncrossPriceLevel(agg *OrderEntry, trades *[]Trade) func(i btree.Item) bool {
	return func(i btree.Item) bool {
		priceLevel := i.(*PriceLevel)
		filled := priceLevel.uncross(agg, trades)
		return !filled
	}
}

func (s *OrderBookSide) cross(agg *OrderEntry) (*[]Trade, uint64) {
	trades := make([]Trade, 0)
	var lastTradedPrice uint64
	s.levels.DescendGreaterThan(s.pivotPriceLevel(agg), uncrossPriceLevel(agg, &trades))
	if len(trades) > 0 {
		lastTradedPrice = trades[len(trades)-1].price
	}

	for _, trade := range trades {
		s.totalVolume += trade.size
	}
	return &trades, lastTradedPrice
}

func (s *OrderBookSide) addOrder(o *OrderEntry) {
	s.getPriceLevel(o.order.Price).addOrder(o)
	s.totalVolume += o.order.Remaining
	s.orderCount++
}

func (s *OrderBookSide) RemoveOrder(o *OrderEntry) {
	priceLevel := s.getPriceLevel(o.order.Price)
	priceLevel.removeOrder(o)
	s.totalVolume -= o.order.Remaining
	s.orderCount--
	// if number of orders on level is == 0
	if priceLevel.orders.Len() == 0 {
		s.removePriceLevel(priceLevel.price)
	}
}