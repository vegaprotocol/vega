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

	if orderMessage.Remaining > 0 && orderMessage.Remaining != orderMessage.Size {
		return msg.OrderError_INVALID_REMAINING_SIZE
	}

	if orderMessage.Id != "" {
		return msg.OrderError_NON_EMPTY_NEW_ORDER_ID
	}

	return msg.OrderError_NONE
}
