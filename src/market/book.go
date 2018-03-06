package market

import (
	"errors"
	"fmt"

	"proto"

	"github.com/google/btree"
)

const priceLevelsBTreeDegree = 32

type OrderBook struct {
	name            string
	buy             *Side
	sell            *Side
	lastTradedPrice uint64
}

func NewBook(name string) *OrderBook {
	b := &OrderBook{
		name: name,
		buy: &Side{
			side:   pb.Order_Buy,
			levels: btree.New(priceLevelsBTreeDegree),
		},
		sell: &Side{
			side:   pb.Order_Sell,
			levels: btree.New(priceLevelsBTreeDegree),
		},
	}
	b.buy.other = b.sell
	b.sell.other = b.buy
	return b
}

func (b *OrderBook) AddOrder(order *pb.Order) (*[]Trade, error) {
	if order.Market != b.GetId() {
		return nil, errors.New(fmt.Sprintf(
			"Market ID mismatch\norder.Market: %v\nbook.ID: %v",
			order.Market,
			b.GetId()))
	}
	wrappedOrder := b.WrapOrder(order)
	var trades *[]Trade
	if order.Side == pb.Order_Buy {
		trades = b.buy.addOrder(wrappedOrder)
	} else { // side == Sell
		trades = b.sell.addOrder(wrappedOrder)
	}
	if trades != nil && len(*trades) > 0 {
		b.lastTradedPrice = (*trades)[len(*trades)-1].price
	}
	return trades, nil
}

func (b *OrderBook) GetName() string {
	return b.name
}

func (b *OrderBook) GetId() string {
	return b.name
}

func (b *OrderBook) GetBBO() (bestBid, bestOffer uint64) {
	return b.buy.bestPrice(), b.sell.bestPrice()
}

//func (b *OrderBook) RemoveOrder(order *Order) bool {
//
//}
//
//func (b *OrderBook) GetBook() (buy, sell []*Order) {
//
//}
