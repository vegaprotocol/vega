package matching

import (
	"fmt"

	"vega/proto"
)

func (b OrderBook) validateOrder(orderMessage *msg.Order) msg.OrderError {
	if orderMessage.Market != b.name {
		panic(fmt.Sprintf(
			"Market ID mismatch\norderMessage.Market: %v\nbook.ID: %v",
			orderMessage.Market,
			b.name))
	}
	if orderMessage.Timestamp < b.latestTimestamp {
		return msg.OrderError_ORDER_OUT_OF_SEQUENCE
	}
	return msg.OrderError_NONE
}
