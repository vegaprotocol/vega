package matching

import (
	"vega/internal/logging"
	types "vega/proto"
)

func (b OrderBook) validateOrder(orderMessage *types.Order) types.OrderError {
	if orderMessage.Market != b.name {
		b.log.Error("Market ID mismatch",
			logging.String("market", orderMessage.Market),
			logging.String("order-book", b.name),
			logging.Order(*orderMessage))

		return types.OrderError_INVALID_MARKET_ID
	}

	if orderMessage.Timestamp < b.latestTimestamp {
		return types.OrderError_ORDER_OUT_OF_SEQUENCE
	}

	if orderMessage.Remaining > 0 && orderMessage.Remaining != orderMessage.Size {
		return types.OrderError_INVALID_REMAINING_SIZE
	}

	// if order is GTT, validate timestamp and convert to block number
	if orderMessage.Type == types.Order_GTT && orderMessage.ExpirationTimestamp == 0 {
		return types.OrderError_INVALID_EXPIRATION_DATETIME
	}

	return types.OrderError_NONE
}
