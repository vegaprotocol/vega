package vega

import (
	"matching"
	"proto"
)

type Vega struct {
	markets map[string]*matching.OrderBook
	orders map[string]*matching.OrderEntry
}

func New() *Vega {
	return &Vega{
		markets: make(map[string]*matching.OrderBook),
		orders: make(map[string]*matching.OrderEntry),
	}
}

func (v Vega) CreateMarket(id string) {
	if _, exists := v.markets[id]; !exists {
		v.markets[id] = matching.NewBook(id, v.orders)
	}
}

func (v Vega) SubmitOrder(order msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	if market, exists := v.markets[order.Market]; exists {
		return market.AddOrder(&order)
	} else {
		return nil, msg.OrderError_INVALID_MARKET_ID
	}
}

func (v Vega) DeleteOrder(id string) *msg.Order {
	if orderEntry, exists := v.orders[id]; exists {
		return orderEntry.GetBook().RemoveOrder(id)
	} else {
		return nil
	}
}

func (v Vega) GetMarketData(marketId string) *msg.MarketData {
	if market, exists := v.markets[marketId]; exists {
		return market.GetMarketData()
	} else {
		return nil
	}
}