package matching
//
//import (
//	"vega/proto"
//)
//
//func MakeResponse(order *msg.Order, trades []Trade, impactedOrders []msg.Order) *msg.OrderConfirmation {
//	tradeSet := make([]*msg.Trade, 0)
//	for _, t := range trades {
//		tradeSet = append(tradeSet, t.toMessage())
//	}
//	expectedPassiveOrdersAffected := make([]*msg.Order, 0)
//	for _, o := range impactedOrders {
//		newOrder := o
//		expectedPassiveOrdersAffected = append(expectedPassiveOrdersAffected, &newOrder)
//	}
//	return &msg.OrderConfirmation{
//		Order:  order,
//		PassiveOrdersAffected: expectedPassiveOrdersAffected,
//		Trades: tradeSet,
//	}
//}
