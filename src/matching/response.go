package matching

import (
	"proto"
)

func MakeResponse(orderId string, trades *[]Trade) *msg.OrderConfirmation {
	tradeSet := make([]*msg.Trade, len(*trades))
	for _, t := range *trades {
		tradeSet = append(tradeSet, t.toMessage())
	}
	return &msg.OrderConfirmation{
		OrderId: orderId,
		Trades:  tradeSet,
	}
}
