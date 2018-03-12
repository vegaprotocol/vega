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

func (e *Exchange) AddOrder(order *pb.Order) (*market.AddOrderResult, error) {
	if book, exists := e.markets[order.Market]; exists {
		result, err := book.AddOrder(order)
		if err != nil {
			panic(fmt.Sprintf("Error adding order: %v", err))
		}
		return result, nil
	} else {
		return nil, errors.New(fmt.Sprintf("Market not found: %v", order.Market))
	}
}

func (e *Exchange) GetMarketData(marketId string) *pb.MarketData {
	if book, exists := e.markets[marketId]; exists {
		return book.GetMarketData()
	} else {
		return nil
	}
}

func (e *Exchange) RemoveOrder(marketId, orderId string) bool {
	if book, exists := e.markets[marketId]; exists {
		return book.RemoveOrder(orderId)
	}
	return false
}
