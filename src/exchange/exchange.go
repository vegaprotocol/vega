package exchange

import (
	"errors"
	"fmt"

	"market"
	"proto"
)

type Exchange struct {
	markets map[string]*market.OrderBook
}

func New() *Exchange {
	return &Exchange{
		markets: make(map[string]*market.OrderBook),
	}
}

func (e *Exchange) NewMarket(name string) (*market.OrderBook, error) {
	if _, exists := e.markets[name]; !exists {
		e.markets[name] = market.NewBook(name)
		return e.markets[name], nil
	} else {
		return nil, errors.New(fmt.Sprintf("Market already exists: %v", name))
	}
}

func (e *Exchange) AddOrder(order *pb.Order) (*[]market.Trade, error) {
	if book, exists := e.markets[order.Market]; exists {
		return book.AddOrder(order)
	} else {
		return nil, errors.New(fmt.Sprintf("Market not found: %v", order.Market))
	}
}
