package matching

import (
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

func (b OrderBook) validateOrder(orderMessage *types.Order) error {
	if orderMessage.MarketID != b.marketID {
		b.log.Error("Market ID mismatch",
			logging.String("market", orderMessage.MarketID),
			logging.String("order-book", b.marketID),
			logging.Order(*orderMessage))

		return types.ErrInvalidMarketID
	}

	if orderMessage.CreatedAt < b.latestTimestamp {
		return types.ErrOrderOutOfSequence
	}

	if orderMessage.Remaining > 0 && orderMessage.Remaining != orderMessage.Size {
		return types.ErrInvalidRemainingSize
	}

	// if order is GTT, validate timestamp and convert to block number
	if orderMessage.TimeInForce == types.Order_GTT && orderMessage.ExpiresAt == 0 {
		return types.ErrInvalidExpirationDatetime
	}

	return nil
}
