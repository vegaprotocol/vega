package matching

import (
	"vega/proto"
)

func MakeResponse(order *msg.Order, trades *[]Trade) *msg.OrderConfirmation {
	tradeSet := make([]*msg.Trade, 0)
	for _, t := range *trades {
		tradeSet = append(tradeSet, t.toMessage())
	}
	return &msg.OrderConfirmation{
		Order:  order,
		Trades: tradeSet,
	}
}
