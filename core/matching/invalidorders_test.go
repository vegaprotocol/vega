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

func TestOrderBookInvalid_emptyMarketID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    "",
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        1,
		Remaining:   1,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidMarketID, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_emptyPartyID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        1,
		Remaining:   1,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPartyID, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_ZeroSize(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        0,
		Remaining:   0,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidRemainingSize, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_ZeroPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.UintZero(),
		Size:        1,
		Remaining:   1,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), confirm.Order.Price.Uint64())
}

func TestOrderBookInvalid_RemainingTooBig(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   11,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidRemainingSize, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTCMarket(t *testing.T) {
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
		Type:        types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTCNetwork(t *testing.T) {
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
		Type:        types.OrderTypeNetwork,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTTMarket(t *testing.T) {
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
		TimeInForce: types.OrderTimeInForceGTT,
		Type:        types.OrderTypeMarket,
		ExpiresAt:   1,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTTNetwork(t *testing.T) {
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
		TimeInForce: types.OrderTimeInForceGTT,
		Type:        types.OrderTypeNetwork,
		ExpiresAt:   1,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_IOCNetwork(t *testing.T) {
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
		TimeInForce: types.OrderTimeInForceIOC,
		Type:        types.OrderTypeNetwork,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}
