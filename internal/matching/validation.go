package matching

import (
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

func (b OrderBook) validateOrder(orderMessage *types.Order) error {
	if orderMessage.Market != b.name {
		b.log.Error("Market ID mismatch",
			logging.String("market", orderMessage.Market),
			logging.String("order-book", b.name),
			logging.Order(*orderMessage))

		return types.ErrInvalidMarketID
	}

	if orderMessage.Timestamp < b.latestTimestamp {
		return types.ErrOrderOutOfSequence
	}

	if orderMessage.Remaining > 0 && orderMessage.Remaining != orderMessage.Size {
		return types.ErrInvalidRemainingSize
	}

	// if order is GTT, validate timestamp and convert to block number
	if orderMessage.Type == types.Order_GTT && orderMessage.ExpirationTimestamp == 0 {
		return types.ErrInvalidExpirationDatetime
	}

	return nil
}
