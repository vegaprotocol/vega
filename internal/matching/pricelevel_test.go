package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/dto"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	assert.Equal(t, 0, len(side.levels))
	side.getPriceLevel(decimal.RequireFromString("100"), types.Side_Sell)
	assert.Equal(t, 1, len(side.levels))

	side.getPriceLevel(decimal.RequireFromString("110"), types.Side_Sell)
	assert.Equal(t, 2, len(side.levels))

	side.getPriceLevel(decimal.RequireFromString("100"), types.Side_Sell)
	assert.Equal(t, 2, len(side.levels))
}

func TestAddAndRemoveOrdersToPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	l := side.getPriceLevel(decimal.RequireFromString("100"), types.Side_Sell)
	protoOrder := &types.Order{
		MarketID:  "testOrderBook",
		PartyID:   "A",
		Side:      types.Side_Sell,
		Price:     []byte("101"),
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		CreatedAt: 0,
	}
	order := &dto.Order{}
	order.FromProto(protoOrder)

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

	side := &OrderBookSide{}
	l := side.getPriceLevel(decimal.RequireFromString("100"), types.Side_Sell)
	passiveProtoOrder := &types.Order{
		MarketID:  "testOrderBook",
		PartyID:   "A",
		Side:      types.Side_Sell,
		Price:     []byte("101"),
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		CreatedAt: 0,
	}
	passiveOrder := &dto.Order{}
	passiveOrder.FromProto(passiveProtoOrder)
	l.addOrder(passiveOrder)

	aggresiveProtoOrder := &types.Order{
		MarketID:  "testOrderBook",
		PartyID:   "B",
		Side:      types.Side_Buy,
		Price:     []byte("101"),
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		CreatedAt: 0,
	}
	aggresiveOrder := &dto.Order{}
	aggresiveOrder.FromProto(aggresiveProtoOrder)
	filled, trades, impactedOrders := l.uncross(aggresiveOrder)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
}
