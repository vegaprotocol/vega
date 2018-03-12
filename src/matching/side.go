package matching

import (
	"container/list"
	"fmt"

	"proto"

	"github.com/google/btree"
)

const priceLevelsBTreeDegree = 32

type OrderBookSide struct {
	book   *OrderBook
	side   msg.Side
	other  *OrderBookSide
	levels *btree.BTree
}

func makeSide(side msg.Side, book *OrderBook) *OrderBookSide {
	return &OrderBookSide{
		book:   book,
		side:   side,
		levels: btree.New(priceLevelsBTreeDegree),
	}
}

func (s *OrderBookSide) addOrder(o *OrderEntry) *[]Trade {
	trades := s.other.uncross(o)
	if o.persist && o.order.Remaining > 0 {
		s.book.orders[o.id] = o
		o.book = s.book
		o.side = s
		o.priceLevel = s.getPriceLevel(o.order.Price)
		o.priceLevel.addOrder(o)
		fmt.Printf("Added: %v\n", o)
	}
	return trades
}

func (s *OrderBookSide) getPriceLevel(price uint64) *PriceLevel {
	var priceLevel *PriceLevel
	item := s.levels.Get(&PriceLevel{side: s.side, price: price})
	if item == nil {
		priceLevel = &PriceLevel{side: s.side, price: price, orders: list.New()}
		s.levels.ReplaceOrInsert(priceLevel)
	} else {
		priceLevel = item.(*PriceLevel)
	}
	return priceLevel
}

func (s *OrderBookSide) removePriceLevel(price uint64) {
	s.levels.Delete(&PriceLevel{side: s.side, price: price})
}

func (s *OrderBookSide) topPriceLevel() *PriceLevel  {
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

func (s *OrderBookSide) uncross(agg *OrderEntry) *[]Trade {
	trades := make([]Trade, 0)

	// No trades if the book isn't crossed
	if s.topPriceLevel() == nil || !agg.crossedWith(s.side, s.bestPrice()) {
		return &trades
	}

	// We want s.levels.DescendGreaterThanOrEqual so modify the price accordingly
	var cutoffPriceLevel *PriceLevel
	if s.side == msg.Side_Buy {
		cutoffPriceLevel = &PriceLevel{side: s.side, price: agg.order.Price - 1}
	} else {
		cutoffPriceLevel = &PriceLevel{side: s.side, price: agg.order.Price + 1}
	}

	// Go through the price levels from best to worst uncrossing each in turn
	s.levels.DescendGreaterThan(
		cutoffPriceLevel,
		func(i btree.Item) bool {
			priceLevel := i.(*PriceLevel)
			filled := priceLevel.uncross(agg, &trades)
			return !filled
		})
	if len(trades) > 0 {
		s.book.lastTradedPrice = trades[len(trades)-1].price
	}
	return &trades
}
