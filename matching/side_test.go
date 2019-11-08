package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func getTestSide() *OrderBookSide {
	return &OrderBookSide{
		log:         logging.NewTestLogger(),
		levels:      []*PriceLevel{},
		proRataMode: false,
	}
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
	side.addOrder(o, types.Side_Sell)
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
	side.addOrder(o2, types.Side_Sell)
	assert.Len(t, side.levels, 2)

	// remove it and check the lenght of the array
	// remove secpmd prder
	side.RemoveOrder(o2)
	assert.Len(t, side.levels, 1)
}

func TestMemoryAllocationGetPriceLevelReturnAPriceLevelIfItAlreadyExists(t *testing.T) {
	// test for a sell side
	side := getTestSide()
	assert.Len(t, side.levels, 0)
	pl := side.getPriceLevel(100, types.Side_Sell)
	assert.Len(t, side.levels, 1)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(101, types.Side_Sell)
	assert.Len(t, side.levels, 2)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(102, types.Side_Sell)
	assert.Len(t, side.levels, 3)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(103, types.Side_Sell)
	assert.Len(t, side.levels, 4)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_Sell)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = side.getPriceLevel(102, types.Side_Sell)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(100, types.Side_Sell)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_Sell)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// test for a buy side
	side = getTestSide()
	assert.Len(t, side.levels, 0)
	pl = side.getPriceLevel(100, types.Side_Buy)
	assert.Len(t, side.levels, 1)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(101, types.Side_Buy)
	assert.Len(t, side.levels, 2)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(102, types.Side_Buy)
	assert.Len(t, side.levels, 3)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(103, types.Side_Buy)
	assert.Len(t, side.levels, 4)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_Buy)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = side.getPriceLevel(102, types.Side_Buy)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(100, types.Side_Buy)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(104, types.Side_Buy)
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
}

func TestMemoryAllocationPriceLevelUncrossRemoveVolumeAtTimestamp(t *testing.T) {
	// we add 3 orders in the same pricelevel at different timestamps
	side := getTestSide()
	o := &types.Order{
		Id:          "order1",
		MarketID:    "testmarket",
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_GTC,
		CreatedAt:   1,
	}
	// add the order to the side
	side.addOrder(o, types.Side_Sell)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        3,
		Remaining:   3,
		TimeInForce: types.Order_GTC,
		CreatedAt:   2,
	}

	// add the order to the side
	side.addOrder(o2, types.Side_Sell)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 2)

	o3 := &types.Order{
		Id:          "order3",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		CreatedAt:   3,
	}

	// add the order to the side
	side.addOrder(o3, types.Side_Sell)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 3)

	aggressiveOrder := &types.Order{
		Id:          "order4",
		MarketID:    "testmarket",
		PartyID:     "X",
		Side:        types.Side_Buy,
		Price:       100,
		Size:        5,
		Remaining:   5,
		TimeInForce: types.Order_GTC,
	}

	// now we uncross for size 2 we should remove the 2 first volumetAtTimestampo
	// and size should now be 1
	side.uncross(aggressiveOrder)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 1)
	assert.Len(t, side.levels[0].orders, 1)
}

func TestMemoryAllocationPriceLevelRemoveOrderRemoveVolumeAtTimestamp(t *testing.T) {
	// we add 3 orders in the same pricelevel at different timestamps
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
		CreatedAt:   1,
	}

	// add the order to the side
	side.addOrder(o, types.Side_Sell)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		CreatedAt:   2,
	}

	// add the order to the side
	side.addOrder(o2, types.Side_Sell)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 2)

	o3 := &types.Order{
		Id:          "order3",
		MarketID:    "testmarket",
		PartyID:     "C",
		Side:        types.Side_Sell,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		CreatedAt:   3,
	}

	// add the order to the side
	side.addOrder(o3, types.Side_Sell)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 3)

	// now we remove order 2
	side.RemoveOrder(o2)
	assert.Len(t, side.levels, 1)
	assert.Len(t, side.levels[0].volumeAtTimestamp, 2)
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
	side.addOrder(o, types.Side_Sell)
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
	side.addOrder(o2, types.Side_Sell)
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
