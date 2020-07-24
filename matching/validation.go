package matching

import (
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrMarketOrderDuringAuction = errors.New("market order not allowed during auction")
	ErrInvalidAuctionTIF        = errors.New("order not allowed during auction")
)

const minOrderIDLen = 22
const maxOrderIDLen = 22

func (b OrderBook) validateOrder(orderMessage *types.Order) (err error) {
	timer := metrics.NewTimeCounter(b.marketID, "matching", "validateOrder")
	// check this first, if this error doesn't get reassigned
	// we might want to add the order anyway if it's market to be valid outside of auctions
	if b.auction {
		if orderMessage.Type == types.Order_TYPE_MARKET {
			err = ErrMarketOrderDuringAuction
		} else if orderMessage.TimeInForce == types.Order_TIF_FOK || orderMessage.TimeInForce == types.Order_TIF_IOC {
			err = ErrInvalidAuctionTIF
		}
	}
	if orderMessage.MarketID != b.marketID {
		b.log.Error("Market ID mismatch",
			logging.String("market", orderMessage.MarketID),
			logging.String("order-book", b.marketID),
			logging.Order(*orderMessage))
		err = types.ErrInvalidMarketID
	} else if orderMessage.Type == types.Order_TYPE_UNSPECIFIED {
		err = types.ErrInvalidType
	} else if orderMessage.Remaining == 0 {
		err = types.ErrInvalidRemainingSize
	} else if orderMessage.TimeInForce == types.Order_TIF_GTT && orderMessage.ExpiresAt == 0 {
		// if order is GTT, validate timestamp and convert to block number
		err = types.ErrInvalidExpirationDatetime
	} else if len(orderMessage.PartyID) == 0 {
		err = types.ErrInvalidPartyID
	} else if orderMessage.Size == 0 {
		err = types.ErrInvalidSize
	} else if orderMessage.Remaining > orderMessage.Size {
		err = types.ErrInvalidRemainingSize
	} else if orderMessage.Type == types.Order_TYPE_NETWORK && orderMessage.TimeInForce != types.Order_TIF_FOK {
		err = types.ErrInvalidPersistence
	} else if orderMessage.TimeInForce == types.Order_TIF_GTT && orderMessage.Type != types.Order_TYPE_LIMIT {
		err = types.ErrInvalidPersistence
	} else if orderMessage.Type == types.Order_TYPE_MARKET &&
		(orderMessage.TimeInForce == types.Order_TIF_GTT || orderMessage.TimeInForce == types.Order_TIF_GTC) {
		err = types.ErrInvalidPersistence
	}
	timer.EngineTimeCounterAdd()
	return
}

func validateOrderID(orderID string) error {
	idLen := len(orderID)
	if idLen < minOrderIDLen || idLen > maxOrderIDLen {
		return types.ErrInvalidOrderID
	}
	return nil
}
