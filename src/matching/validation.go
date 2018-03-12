package matching

import (
	"fmt"

	"proto"
)

func (b OrderBook) validateOrder(orderMessage *msg.Order) msg.OrderError {
	if orderMessage.Market != b.name {
		panic(fmt.Sprintf(
			"Market ID mismatch\norderMessage.Market: %v\nbook.ID: %v",
			orderMessage.Market,
			b.name))
	}
	return msg.OrderError_NONE
}
