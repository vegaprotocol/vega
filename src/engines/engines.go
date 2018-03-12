package engines

import "proto"

type MatchingEngine interface {
	CreateMarket(id string)
	SubmitOrder(order msg.Order) (*msg.OrderConfirmation, msg.OrderError)
	DeleteOrder(id string) *msg.Order
	GetMarketData(marketId string) *msg.MarketData
}

type RiskEngine interface {

}

type SettlementEngine interface {

}