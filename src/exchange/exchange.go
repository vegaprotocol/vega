package exchange

import (
	"container/list"
	"errors"
	"fmt"

	"book"
	"proto"
)

type Exchange struct {
	markets	map[string]*book.OrderBook
}

func (e *Exchange) NewMarket(name string) (*book.OrderBook, error) {
	if _, exists := e.markets[name]; !exists {
		e.markets[name] = book.NewBook(name)
		return e.markets[name], nil
	} else {
		return nil, errors.New(fmt.Sprintf("Market already exists: %v", name))
	}
}

func (e *Exchange) AddOrder(order *pb.Order) (*list.List, error) {
	if book, exists := e.markets[order.Market]; exists {
		return book.AddOrder(order), nil
	} else {
		return nil, errors.New(fmt.Sprintf("Market not found: %v", order.Market))
	}
}