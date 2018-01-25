package mbook

import (
	"container/list"
	"fmt"

	"github.com/google/btree"
)

type Side struct {
	side   BuySell
	other  *Side
	levels *btree.BTree
	top    *PriceLevel
}

func (s *Side) addOrder(o *Order) *list.List {
	trades := s.other.uncross(o)
	if o.remaining > 0 {
		s.getPriceLevel(o.side, o.price).addOrder(o)
		fmt.Printf("Added: %v\n", o)
	}
	return trades
}

func (s *Side) getPriceLevel(side BuySell, price uint64) *PriceLevel {
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

func (s *Side) uncross(agg *Order) *list.List {
	if s.top == nil || !agg.crossedWith(s.side, s.top.price) {
		return nil
	}
	var trades = list.New()
	s.levels.DescendGreaterThan(
		&PriceLevel{side: s.side, price: agg.price},
		func(i btree.Item) bool {
			priceLevel := i.(*PriceLevel)
			filled := priceLevel.uncross(agg, trades)
			if priceLevel.volume == 0 {
				s.levels.Delete(priceLevel)
				s.top = s.levels.Max().(*PriceLevel)
			} else {
				s.updateTop(priceLevel)
			}
			return !filled
		})
	return trades
}
