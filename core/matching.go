package core

import (
	"vega/matching"
	"vega/proto"
)

type MatchingEngine interface {
	CreateMarket(id string)
	SubmitOrder(order msg.Order) (*msg.OrderConfirmation, msg.OrderError)
	DeleteOrder(id string) *msg.Order
	GetMarketData(marketId string) *msg.MarketData
}

func (v Vega) CreateMarket(id string) {
	if _, exists := v.markets[id]; !exists {
		v.markets[id] = matching.NewBook(id, v.config.Matching)
	}
}

func (v Vega) SubmitOrder(order msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	if market, exists := v.markets[order.Market]; exists {
		return market.AddOrder(&order)
	} else {
		return nil, msg.OrderError_INVALID_MARKET_ID
	}
}

func (v Vega) DeleteOrder(id, marketName string) {
	if orderEntry, exists := v.orders[id]; exists {
		if market, exists := v.markets[marketName]; exists {
			market.RemoveOrder(orderEntry)
		}
	}
}

func (v Vega) GetMarketData(marketId string) *msg.MarketData {
	if market, exists := v.markets[marketId]; exists {
		return market.GetMarketData()
	} else {
		return nil
	}
}

func (v Vega) GetMarketDepth(marketId string) *msg.MarketDepth {
	if market, exists := v.markets[marketId]; exists {
		return market.GetMarketDepth()
	} else {
		return nil
	}
}

// run a separate go routine that will read on channel and update orders map