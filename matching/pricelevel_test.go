package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

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
	filled, trades, impactedOrders, err := l.uncross(aggresiveOrder, true)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
	assert.NoError(t, err)
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
	filled, trades, impactedOrders, err := l.uncross(aggresiveOrder, true)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
	assert.NoError(t, err)
	// ensure the price fields are set correctly
	assert.Equal(t, passiveOrder.OriginalPrice.String(), trades[0].MarketPrice.String())
	assert.Equal(t, passiveOrder.Price.String(), trades[0].Price.String())
}
