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
	"testing"

	vgcrypto "code.vegaprotocol.io/vega/core/libs/crypto"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"

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
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
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

	orderId := vgcrypto.RandomHash()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		ID:          orderId,
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
		ID:          orderId,                   // Must match
	}
	_, err = book.CancelOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}
