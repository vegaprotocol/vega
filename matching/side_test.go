package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func getTestSide(side types.Side) *OrderBookSide {
	return &OrderBookSide{
		log:    logging.NewTestLogger(),
		levels: []*PriceLevel{},
		side:   side,
	}
}

func TestMemoryAllocationPriceLevelRemoveOrder(t *testing.T) {
	side := getTestSide(types.Side_SIDE_SELL)
	o := &types.Order{
		Id:          "order1",
		MarketId:    "testmarket",
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}
	// add the order to the side
	side.addOrder(o)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketId:    "testmarket",
		PartyId:     "C",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}

	// add the order to the side
	side.addOrder(o2)
	assert.Len(t, side.levels, 2)

	// remove it and check the length of the array
	// remove second order
	side.RemoveOrder(o2)
	assert.Len(t, side.levels, 1)
}

func TestMemoryAllocationGetPriceLevelReturnAPriceLevelIfItAlreadyExists(t *testing.T) {
	// test for a sell side
	side := getTestSide(types.Side_SIDE_SELL)
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
	side = getTestSide(types.Side_SIDE_BUY)
	assert.Len(t, side.levels, 0)
	pl = side.getPriceLevel(100)
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
}

func TestMemoryAllocationPriceLevelUncrossSide(t *testing.T) {
	side := getTestSide(types.Side_SIDE_SELL)
	o := &types.Order{
		Id:          "order1",
		MarketId:    "testmarket",
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}
	// add the order to the side
	side.addOrder(o)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		Id:          "order2",
		MarketId:    "testmarket",
		PartyId:     "C",
		Side:        types.Side_SIDE_SELL,
		Price:       101,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}

	// add the order to the side
	side.addOrder(o2)
	assert.Len(t, side.levels, 2)

	aggressiveOrder := &types.Order{
		Id:          "order3",
		MarketId:    "testmarket",
		PartyId:     "X",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}
	side.uncross(aggressiveOrder, true)
	assert.Len(t, side.levels, 1)
}

func getPopulatedTestSide(side types.Side) *OrderBookSide {
	obs := getTestSide(types.Side_SIDE_SELL)

	type testOrder struct {
		Id    string
		Price uint64
		Size  uint64
	}

	testOrders := []testOrder{
		{"Order01", 100, 1},
		{"Order02", 100, 1},
		{"Order03", 100, 1},
		{"Order04", 101, 1},
		{"Order05", 101, 1},
		{"Order06", 101, 1},
	}

	for _, order := range testOrders {
		o := &types.Order{
			Id:          order.Id,
			MarketId:    "testmarket",
			PartyId:     "A",
			Side:        types.Side_SIDE_SELL,
			Price:       order.Price,
			Size:        order.Size,
			Remaining:   order.Size,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}
		// add the order to the side
		obs.addOrder(o)
	}
	return obs
}

func TestExtractOrdersFullLevel(t *testing.T) {
	side := getPopulatedTestSide(types.Side_SIDE_SELL)

	assert.Len(t, side.levels, 2)

	orders, err := side.ExtractOrders(100, 3)
	assert.NoError(t, err)
	assert.Len(t, side.levels, 1)
	assert.Len(t, orders, 3)
	assert.EqualValues(t, 3, side.getOrderCount())
}

func TestExtractOrdersPartialLevel(t *testing.T) {
	side := getPopulatedTestSide(types.Side_SIDE_SELL)

	assert.Len(t, side.levels, 2)

	orders, err := side.ExtractOrders(100, 2)
	assert.NoError(t, err)
	assert.Len(t, side.levels, 2)
	assert.Len(t, orders, 2)
	assert.EqualValues(t, 4, side.getOrderCount())
}

func TestExtractOrdersCrossLevel(t *testing.T) {
	side := getPopulatedTestSide(types.Side_SIDE_SELL)

	assert.Len(t, side.levels, 2)

	orders, err := side.ExtractOrders(101, 5)
	assert.NoError(t, err)
	assert.Len(t, side.levels, 1)
	assert.Len(t, orders, 5)
	assert.EqualValues(t, 1, side.getOrderCount())
}
