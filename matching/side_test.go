package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func getTestSide() *OrderBookSide {
	return &OrderBookSide{
		log:    logging.NewTestLogger(),
		levels: []*PriceLevel{},
	}
}

func TestMemoryAllocationPriceLevelRemoveOrder(t *testing.T) {
	side := getTestSide()
	o := &types.Order{
		Id:          "order1",
		MarketID:    "testmarket",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
	}
	// add the order to the side
	side.addOrder(o, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
	}

	// add the order to the side
	side.addOrder(o2, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 2)

	// remove it and check the length of the array
	// remove secpmd prder
	side.RemoveOrder(o2)
	assert.Len(t, side.levels, 1)
}

func TestMemoryAllocationGetPriceLevelReturnAPriceLevelIfItAlreadyExists(t *testing.T) {
	// test for a sell side
	side := getTestSide()
	assert.Len(t, side.levels, 0)
	pl := side.getPriceLevel(100, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 1)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(101, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 2)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(102, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 3)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(103, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 4)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = side.getPriceLevel(102, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(100, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// test for a buy side
	side = getTestSide()
	assert.Len(t, side.levels, 0)
	pl = side.getPriceLevel(100, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 1)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(101, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 2)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(102, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 3)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(103, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 4)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = side.getPriceLevel(102, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(100, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
}

func TestMemoryAllocationPriceLevelUncrossSide(t *testing.T) {
	side := getTestSide()
	o := &types.Order{
		Id:          "order1",
		MarketID:    "testmarket",
		PartyID:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
	}
	// add the order to the side
	side.addOrder(o, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
	}

	// add the order to the side
	side.addOrder(o2, types.Side_SIDE_SELL)
	assert.Len(t, side.levels, 2)

	aggressiveOrder := &types.Order{
		Id:          "order3",
		MarketID:    "testmarket",
		PartyID:     "X",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
	}

	side.uncross(aggressiveOrder)
	assert.Len(t, side.levels, 1)
}
