package matching

import (
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	assert.Equal(t, 0, len(side.levels))
	side.getPriceLevel(100, types.Side_Sell)
	assert.Equal(t, 1, len(side.levels))

	side.getPriceLevel(110, types.Side_Sell)
	assert.Equal(t, 2, len(side.levels))

	side.getPriceLevel(100, types.Side_Sell)
	assert.Equal(t, 2, len(side.levels))
}

func TestAddAndRemoveOrdersToPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	l := side.getPriceLevel(100, types.Side_Sell)
	order := &types.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      types.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
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
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	conf := NewDefaultConfig(logger)
	side := &OrderBookSide{Config: conf}
	l := side.getPriceLevel(100, types.Side_Sell)
	passiveOrder := &types.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      types.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
	}
	l.addOrder(passiveOrder)

	aggresiveOrder := &types.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      types.Side_Buy,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		Timestamp: 0,
	}
	filled, trades, impactedOrders := l.uncross(aggresiveOrder)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
}
