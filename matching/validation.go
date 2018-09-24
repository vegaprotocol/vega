package matching

import (
	"fmt"

	"vega/log"
	"vega/msg"
)

func (b OrderBook) validateOrder(orderMessage *msg.Order) msg.OrderError {
	if orderMessage.Market != b.name {
		log.Infof(fmt.Sprintf(
			"Market ID mismatch\norderMessage.Market: %v\nbook.ID: %v",
			orderMessage.Market,
			b.name))
		return msg.OrderError_INVALID_MARKET_ID
	}

	if orderMessage.Timestamp < b.latestTimestamp {
		return msg.OrderError_ORDER_OUT_OF_SEQUENCE
	}

	if orderMessage.Remaining > 0 && orderMessage.Remaining != orderMessage.Size {
		return msg.OrderError_INVALID_REMAINING_SIZE
	}

	// if order is GTT, validate timestamp and convert to block number
	if orderMessage.Type == msg.Order_GTT &&
		(orderMessage.ExpirationDatetime == "" || orderMessage.ExpirationTimestamp == 0) {
		return msg.OrderError_INVALID_EXPIRATION_DATETIME
	}

	return msg.OrderError_NONE
}
