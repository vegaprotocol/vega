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
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
)

func TestGetPriceLevel(t *testing.T) {
	side := &OrderBookSide{side: types.SideSell}
	assert.Equal(t, 0, len(side.levels))
	side.getPriceLevel(num.NewUint(100))
	assert.Equal(t, 1, len(side.levels))

	side.getPriceLevel(num.NewUint(110))
	assert.Equal(t, 2, len(side.levels))

	side.getPriceLevel(num.NewUint(100))
	assert.Equal(t, 2, len(side.levels))
}

func TestAddAndRemoveOrdersToPriceLevel(t *testing.T) {
	side := &OrderBookSide{side: types.SideSell}
	l := side.getPriceLevel(num.NewUint(100))
	order := &types.Order{
		MarketID:      "testOrderBook",
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
	}

	// add orders
	assert.Equal(t, 0, len(l.orders))
	l.addOrder(order)
	assert.Equal(t, 1, len(l.orders))
	l.addOrder(order)
	assert.Equal(t, 2, len(l.orders))

	// remove orders
	l.removeOrder(1)
	assert.Equal(t, 1, len(l.orders))
	l.removeOrder(0)
	assert.Equal(t, 0, len(l.orders))
}

func TestUncross(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()

	side := &OrderBookSide{side: types.SideSell}
	l := side.getPriceLevel(num.NewUint(100))
	passiveOrder := &types.Order{
		MarketID:      "testOrderBook",
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
	}
	l.addOrder(passiveOrder)

	aggresiveOrder := &types.Order{
		MarketID:      "testOrderBook",
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
	}

	order, fakeTrades, err := l.fakeUncross(aggresiveOrder, true)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), order.Remaining)

	filled, trades, impactedOrders, err := l.uncross(aggresiveOrder, true)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
	assert.NoError(t, err)

	for i, tr := range trades {
		assert.Equal(t, tr, fakeTrades[i])
	}
}

func TestUncrossDecimals(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()

	side := &OrderBookSide{side: types.SideSell}
	l := side.getPriceLevel(num.NewUint(100))
	passiveOrder := &types.Order{
		MarketID:      "testOrderBook",
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101000),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
	}
	l.addOrder(passiveOrder)

	aggresiveOrder := &types.Order{
		MarketID:      "testOrderBook",
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101000),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
	}

	order, fakeTrades, err := l.fakeUncross(aggresiveOrder, true)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), order.Remaining)

	filled, trades, impactedOrders, err := l.uncross(aggresiveOrder, true)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
	assert.NoError(t, err)
	// ensure the price fields are set correctly
	assert.Equal(t, passiveOrder.OriginalPrice.String(), trades[0].MarketPrice.String())
	assert.Equal(t, passiveOrder.Price.String(), trades[0].Price.String())

	for i, tr := range trades {
		assert.Equal(t, tr, fakeTrades[i])
	}
}
