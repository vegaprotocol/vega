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
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

func TestOrderBookSimple_CancelWrongOrderIncorrectOrderID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000002", // Invalid, must match original
	}
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelWrongOrderIncorrectMarketID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    "incorrectMarket", // Invalid, must match original
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000001",
	}

	assert.Panics(t, func() {
		_, err = book.CancelOrder(&order2)
	},
	)

	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelWrongOrderIncorrectSide(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideSell, // Invalid, must match original
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000001",
	}
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelWrongOrderIncorrectPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(101), // Invalid, must match original
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          "v0000000000000-0000001",
	}
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelOrderIncorrectNonCriticalFields(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orderID := vgcrypto.RandomHash()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          orderID,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,                    // Must match
		Party:       "B",                       // Does not matter
		Side:        types.SideBuy,             // Must match
		Price:       num.NewUint(100),          // Must match
		Size:        10,                        // Does not matter
		Remaining:   10,                        // Does not matter
		TimeInForce: types.OrderTimeInForceGTC, // Does not matter
		Type:        types.OrderTypeLimit,      // Does not matter
		ID:          orderID,                   // Must match
	}
	_, err = book.CancelOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}
