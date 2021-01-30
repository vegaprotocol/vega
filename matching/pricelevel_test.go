package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestGetPriceLevel(t *testing.T) {
	side := &OrderBookSide{side: types.Side_SIDE_SELL}
	assert.Equal(t, 0, len(side.levels))
	side.getPriceLevel(100)
	assert.Equal(t, 1, len(side.levels))

	side.getPriceLevel(110)
	assert.Equal(t, 2, len(side.levels))

	side.getPriceLevel(100)
	assert.Equal(t, 2, len(side.levels))
}

func TestAddAndRemoveOrdersToPriceLevel(t *testing.T) {
	side := &OrderBookSide{side: types.Side_SIDE_SELL}
	l := side.getPriceLevel(100)
	order := &types.Order{
		MarketID:    "testOrderBook",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		CreatedAt:   0,
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

	side := &OrderBookSide{side: types.Side_SIDE_SELL}
	l := side.getPriceLevel(100)
	passiveOrder := &types.Order{
		MarketID:    "testOrderBook",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		CreatedAt:   0,
	}
	l.addOrder(passiveOrder)

	aggresiveOrder := &types.Order{
		MarketID:    "testOrderBook",
		PartyID:     "B",
		Side:        types.Side_SIDE_BUY,
		Price:       101,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		CreatedAt:   0,
	}
	filled, trades, impactedOrders, err := l.uncross(aggresiveOrder, true)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
	assert.NoError(t, err)
}
