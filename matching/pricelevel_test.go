package matching

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"vega/proto"
)

func TestGetPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	assert.Equal(t, 0, len(side.levels))
	side.getPriceLevel(100, msg.Side_Sell)
	assert.Equal(t, 1, len(side.levels))

	side.getPriceLevel(110, msg.Side_Sell)
	assert.Equal(t, 2, len(side.levels))

	side.getPriceLevel(100, msg.Side_Sell)
	assert.Equal(t, 2, len(side.levels))
}

func TestAddAndRemoveOrdersToPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	l := side.getPriceLevel(100, msg.Side_Sell)
	order := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
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
	side := &OrderBookSide{}
	l := side.getPriceLevel(100, msg.Side_Sell)
	passiveOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "A",
		Side:      msg.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
	}
	l.addOrder(passiveOrder)

	aggresiveOrder := &msg.Order{
		Market:    "testOrderBook",
		Party:     "B",
		Side:      msg.Side_Buy,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      msg.Order_GTC,
		Timestamp: 0,
	}
	filled, trades, impactedOrders := l.uncross(aggresiveOrder)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
}
