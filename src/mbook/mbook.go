package mbook

import (
	"container/list"

	"github.com/google/btree"
)

const priceLevelsBTreeDegree = 32

type MBook struct {
	market string
	buy    *Side
	sell   *Side
}

func NewBook(market string) *MBook {
	b := &MBook{
		market: market,
		buy: &Side{
			side:   Buy,
			levels: btree.New(priceLevelsBTreeDegree),
		},
		sell: &Side{
			side:   Sell,
			levels: btree.New(priceLevelsBTreeDegree),
		},
	}
	b.buy.other = b.sell
	b.sell.other = b.buy
	return b
}

func (b *MBook) AddOrder(side BuySell, size uint64, price uint64, party string) *list.List {
	if side == Buy {
		return b.buy.addOrder(&Order{
			party,
			side,
			size,
			size,
			price,
			nil,
		})
	} else { // side == Sell
		return b.sell.addOrder(&Order{
			party,
			side,
			size,
			size,
			price,
			nil,
		})
	}
}

func (b *MBook) GetBBO() (bestBid, bestOffer uint64) {
	return b.buy.bestPrice(), b.sell.bestPrice()
}
