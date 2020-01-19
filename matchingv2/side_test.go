package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func getTestSide() *SellSide {
	return NewSellSide(logging.NewTestLogger())
}

func getTestBuySide() *BuySide {
	return NewBuySide(logging.NewTestLogger())
}

func TestMemoryAllocationPriceLevelRemoveOrder(t *testing.T) {
	side := getTestSide()
	o := &types.Order{
		Id:          "order1",
		MarketID:    "testmarket",
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
	}
	// add the order to the side
	side.AddOrder(o)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_Sell,
		Price:       101,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
	}

	// add the order to the side
	side.AddOrder(o2)
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
	pl := side.getPriceLevel(100)
	assert.Len(t, side.levels, 1)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(101)
	assert.Len(t, side.levels, 2)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(102)
	assert.Len(t, side.levels, 3)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(103)
	assert.Len(t, side.levels, 4)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = side.getPriceLevel(102)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(100)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// test for a buy side
	bside := getTestBuySide()
	assert.Len(t, bside.levels, 0)
	pl = bside.getPriceLevel(100)
	assert.Len(t, bside.levels, 1)
	assert.NotNil(t, pl)
	pl = bside.getPriceLevel(101)
	assert.Len(t, bside.levels, 2)
	assert.NotNil(t, pl)
	pl = bside.getPriceLevel(102)
	assert.Len(t, bside.levels, 3)
	assert.NotNil(t, pl)
	pl = bside.getPriceLevel(103)
	assert.Len(t, bside.levels, 4)
	assert.NotNil(t, pl)
	pl = bside.getPriceLevel(104)
	assert.Len(t, bside.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = bside.getPriceLevel(102)
	assert.Len(t, bside.levels, 5)
	assert.NotNil(t, pl)
	pl = bside.getPriceLevel(100)
	assert.Len(t, bside.levels, 5)
	assert.NotNil(t, pl)
	pl = bside.getPriceLevel(104)
	assert.Len(t, bside.levels, 5)
	assert.NotNil(t, pl)
}

func TestMemoryAllocationPriceLevelUncrossSide(t *testing.T) {
	side := getTestSide()
	o := &types.Order{
		Id:          "order1",
		MarketID:    "testmarket",
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
	}
	// add the order to the side
	side.AddOrder(o)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_Sell,
		Price:       101,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
	}

	// add the order to the side
	side.AddOrder(o2)
	assert.Len(t, side.levels, 2)

	aggressiveOrder := &types.Order{
		Id:          "order3",
		MarketID:    "testmarket",
		PartyID:     "X",
		Side:        types.Side_Buy,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
	}

	side.uncross(aggressiveOrder)
	assert.Len(t, side.levels, 1)
}
