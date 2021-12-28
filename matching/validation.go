package matching

import (
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

const (
	minOrderIDLen = 22
	maxOrderIDLen = 22
)

func (b OrderBook) validateOrder(orderMessage *types.Order) (err error) {
	if orderMessage.Price == nil {
		orderMessage.Price = num.Zero()
	}
	if orderMessage.MarketID != b.marketID {
		b.log.Error("Market ID mismatch",
			logging.String("market", orderMessage.MarketID),
			logging.String("order-book", b.marketID),
			logging.Order(*orderMessage))
		err = types.ErrInvalidMarketID
	} else if orderMessage.Type == types.OrderTypeUnspecified {
		err = types.ErrInvalidType
	} else if orderMessage.Remaining == 0 {
		err = types.ErrInvalidRemainingSize
	} else if orderMessage.TimeInForce == types.OrderTimeInForceGTT && orderMessage.ExpiresAt == 0 {
		// if order is GTT, validate timestamp and convert to block number
		err = types.ErrInvalidExpirationDatetime
	} else if len(orderMessage.Party) == 0 {
		err = types.ErrInvalidPartyID
	} else if orderMessage.Size == 0 {
		err = types.ErrInvalidSize
	} else if orderMessage.Remaining > orderMessage.Size {
		err = types.ErrInvalidRemainingSize
	} else if orderMessage.Type == types.OrderTypeNetwork && orderMessage.TimeInForce != types.OrderTimeInForceFOK {
		err = types.ErrInvalidPersistence
	} else if orderMessage.TimeInForce == types.OrderTimeInForceGTT && orderMessage.Type != types.OrderTypeLimit {
		err = types.ErrInvalidPersistence
	} else if orderMessage.Type == types.OrderTypeMarket &&
		(orderMessage.TimeInForce == types.OrderTimeInForceGTT || orderMessage.TimeInForce == types.OrderTimeInForceGTC) {
		err = types.ErrInvalidPersistence
	} else if b.auction && orderMessage.TimeInForce == types.OrderTimeInForceGFN {
		err = types.ErrInvalidTimeInForce
	} else if !b.auction && orderMessage.TimeInForce == types.OrderTimeInForceGFA {
		err = types.ErrInvalidTimeInForce
	} else if orderMessage.ExpiresAt > 0 && orderMessage.Type == types.OrderTypeMarket {
		err = types.ErrInvalidExpirationDatetime
	}

	return err
}

func validateOrderID(orderID string) error {
	idLen := len(orderID)
	if idLen < minOrderIDLen || idLen > maxOrderIDLen {
		return types.ErrInvalidOrderID
	}
	return nil
}
