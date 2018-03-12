package matching

import (
	"proto"
)

func MakeResponse(order *msg.Order, trades *[]Trade) *msg.OrderConfirmation {
	tradeSet := make([]*msg.Trade, len(*trades))
	for _, t := range *trades {
		tradeSet = append(tradeSet, t.toMessage())
	}
	return &msg.OrderConfirmation{
		Order: order,
		Trades:  tradeSet,
	}
}
