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
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

func TestOrderBook_MarketOrderFOKNotFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceFOK,
		Type:        types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.Price.Uint64())
}

func TestOrderBook_MarketOrderIOCNotFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceIOC,
		Type:        types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.Price.Uint64())
}

func TestOrderBook_MarketOrderFOKPartiallyFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          6,
		Remaining:     6,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	order = types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceFOK,
		Type:        types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.Price.Uint64())

	// Nothing was filled
	assert.Equal(t, uint64(10), confirm.Order.Remaining)

	// No orders
	assert.Nil(t, confirm.Trades)
	assert.Nil(t, confirm.PassiveOrdersAffected)
}

func TestOrderBook_MarketOrderIOCPartiallyFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          6,
		Remaining:     6,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	order2 := types.Order{
		MarketID:    market,
		Party:       "B",
		Side:        types.SideBuy,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceIOC,
		Type:        types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order2)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.Price.Uint64())

	// Something was filled
	assert.Equal(t, uint64(4), confirm.Order.Remaining)

	// One order
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, 1, len(confirm.PassiveOrdersAffected))
}
