package book

import (
	"container/list"
	"fmt"

	"proto"

	"github.com/google/btree"
)

const initialTradeListSize = 10

type Side struct {
	side   pb.Order_Side
	other  *Side
	levels *btree.BTree
	top    *PriceLevel
}

func (s *Side) addOrder(o *wrappedOrder) *[]Trade {
	trades := s.other.uncross(o)
	if o.persist && o.order.Remaining > 0 {
		s.getPriceLevel(o.order.Side, o.order.Price).addOrder(o)
		fmt.Printf("Added: %v\n", o)
	}
	return trades
}

func (s *Side) getPriceLevel(side pb.Order_Side, price uint64) *PriceLevel {
	var priceLevel *PriceLevel
	item := s.levels.Get(&PriceLevel{side: side, price: price})
	if item == nil {
		priceLevel = &PriceLevel{side: s.side, price: price, orders: list.New()}
		s.levels.ReplaceOrInsert(priceLevel)
		s.updateTop(priceLevel)
	} else {
		priceLevel = item.(*PriceLevel)
	}
	return priceLevel
}

func (s *Side) updateTop(l *PriceLevel) {
	if s.top == nil || s.top.Less(l) {
		s.top = l
	}
}

func (s *Side) bestPrice() uint64 {
	if s.top == nil {
		return 0
	} else {
		return s.top.price
	}
}

func (s *Side) uncross(agg *wrappedOrder) *[]Trade {
	if s.top == nil || !agg.crossedWith(s.side, s.top.price) {
		return nil
	}
	trades := make([]Trade, initialTradeListSize)
	s.levels.DescendGreaterThan(
		&PriceLevel{side: s.side, price: agg.order.Price},
		func(i btree.Item) bool {
			priceLevel := i.(*PriceLevel)
			filled := priceLevel.uncross(agg, &trades)
			if priceLevel.volume == 0 {
				s.levels.Delete(priceLevel)
				s.top = s.levels.Max().(*PriceLevel)
			} else {
				s.updateTop(priceLevel)
			}
			return !filled
		})
	return &trades
}
