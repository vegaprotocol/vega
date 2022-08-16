// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package matching

import (
	"encoding/hex"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

const (
	orderIdLen = 64
)

func (b OrderBook) validateOrder(orderMessage *types.Order) (err error) {
	if orderMessage.Price == nil {
		orderMessage.Price = num.UintZero()
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
	_, err := hex.DecodeString(orderID)
	idLen := len(orderID)
	if err != nil || idLen != orderIdLen {
		return types.ErrInvalidOrderID
	}
	return nil
}
