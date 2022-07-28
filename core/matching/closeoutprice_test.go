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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"

	"github.com/stretchr/testify/assert"
)

func TestOrderBook_closeOutPriceBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	tradedOrder1 := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&tradedOrder1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	tradedOrder2 := types.Order{
		MarketID:      market,
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&tradedOrder2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(10, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Incorrect size
	price, err = book.GetCloseoutPrice(0, types.SideBuy)
	assert.Error(t, err, ErrInvalidVolume)
	assert.Equal(t, price.Uint64(), uint64(0))

	// Not enough on the book
	price, err = book.GetCloseoutPrice(200, types.SideBuy)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Wrong side
	price, err = book.GetCloseoutPrice(10, types.SideSell)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))
}

func TestOrderBook_closeOutPriceSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	tradedOrder1 := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&tradedOrder1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	tradedOrder2 := types.Order{
		MarketID:      market,
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&tradedOrder2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	untradedOrder := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&untradedOrder)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(10, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Incorrect size
	price, err = book.GetCloseoutPrice(0, types.SideSell)
	assert.Error(t, err, ErrInvalidVolume)
	assert.Equal(t, price.Uint64(), uint64(0))

	// Not enough on the book
	price, err = book.GetCloseoutPrice(200, types.SideSell)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Wrong side
	price, err = book.GetCloseoutPrice(10, types.SideBuy)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))
}

func TestOrderBook_closeOutPriceBuy2(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(90),
		OriginalPrice: num.NewUint(90),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order3 := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(80),
		OriginalPrice: num.NewUint(80),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order3)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(100, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Normal case
	price, err = book.GetCloseoutPrice(200, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(95))

	// Normal case
	price, err = book.GetCloseoutPrice(300, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(90))
}

func TestOrderBook_closeOutPriceSell2(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(110),
		OriginalPrice: num.NewUint(110),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order3 := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(120),
		OriginalPrice: num.NewUint(120),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order3)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(100, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Normal case
	price, err = book.GetCloseoutPrice(200, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(105))

	// Normal case
	price, err = book.GetCloseoutPrice(300, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(110))
}
