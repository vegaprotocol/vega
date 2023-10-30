// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package matching

import (
	"encoding/hex"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

const (
	orderIDLen = 64
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
	if err != nil || idLen != orderIDLen {
		return types.ErrInvalidOrderID
	}
	return nil
}
