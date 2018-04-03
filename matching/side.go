package matching

import (
	"fmt"

	"vega/proto"

	"github.com/google/btree"
)

const priceLevelsBTreeDegree = 128

type OrderBookSide struct {
	book        *OrderBook
	side        msg.Side
	other       *OrderBookSide
	levels      *btree.BTree
	orderCount  uint64
	totalVolume uint64
}

func makeSide(side msg.Side, book *OrderBook) *OrderBookSide {
	return &OrderBookSide{
		book:   book,
		side:   side,
		levels: btree.New(priceLevelsBTreeDegree),
	}
}

func (s *OrderBookSide) getPriceLevel(price uint64) *PriceLevel {
	var priceLevel *PriceLevel
	item := s.levels.Get(&PriceLevel{side: s.side, price: price})
	if item == nil {
		priceLevel = NewPriceLevel(s, price)
		s.levels.ReplaceOrInsert(priceLevel)
	} else {
		priceLevel = item.(*PriceLevel)
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
	s.levels.Delete(&PriceLevel{side: s.side, price: price})
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
		return &PriceLevel{side: s.side, price: agg.order.Price - 1}
	} else {
		return &PriceLevel{side: s.side, price: agg.order.Price + 1}
	}
}

func (s *OrderBookSide) bestPrice() uint64 {
	if s.topPriceLevel() == nil {
		return 0
	} else {
		return s.topPriceLevel().price
	}
}

func (s *OrderBookSide) addOrder(o *OrderEntry) *[]Trade {
	trades := s.other.uncross(o)
	if o.persist && o.order.Remaining > 0 {
		s.book.orders[o.order.Id] = o
		o.book = s.book
		o.side = s
		o.priceLevel = s.getPriceLevel(o.order.Price)
		o.priceLevel.addOrder(o)
		if !s.book.config.Quiet {
			fmt.Printf("Added: %v\n", o)
		}
	}
	return trades
}

// Go through the price levels from best to worst uncrossing each in turn
func (s *OrderBookSide) uncross(agg *OrderEntry) *[]Trade {
	trades := make([]Trade, 0)
	s.levels.DescendGreaterThan(s.pivotPriceLevel(agg), uncrossPriceLevel(agg, &trades))
	if len(trades) > 0 {
		s.book.lastTradedPrice = trades[len(trades)-1].price
	}
	return &trades
}

// Returns closure over the aggressor and trades slice that calls priceLevel.uncross(...)
func uncrossPriceLevel(agg *OrderEntry, trades *[]Trade) func(i btree.Item) bool {
	return func(i btree.Item) bool {
		priceLevel := i.(*PriceLevel)
		filled := priceLevel.uncross(agg, trades)
		return !filled
	}
}
