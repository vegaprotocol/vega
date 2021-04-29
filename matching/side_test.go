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
	obs := getTestSide(side)

	type testOrder struct {
		ID    string
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
			Id:          order.ID,
			MarketId:    "testmarket",
			PartyId:     "A",
			Side:        side,
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

func getPopulatedTestSideWithPegs(side types.Side) *OrderBookSide {
	obs := getTestSide(side)

	type testOrder struct {
		ID     string
		Price  uint64
		Size   uint64
		Offset int64
	}

	testOrders := []testOrder{
		{"Order01", 100, 1, 5},
		{"Order02", 101, 1, 0},
		{"Order03", 102, 1, 0},
		{"Order04", 103, 1, 8},
		{"Order05", 104, 1, 0},
		{"Order06", 105, 1, 0},
	}

	for _, order := range testOrders {
		o := &types.Order{
			Id:          order.ID,
			MarketId:    "testmarket",
			PartyId:     "A",
			Side:        side,
			Price:       order.Price,
			Size:        order.Size,
			Remaining:   order.Size,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}
		if order.Offset != 0 {
			o.PeggedOrder = &types.PeggedOrder{
				Reference: types.PeggedReference_PEGGED_REFERENCE_MID,
				Offset:    order.Offset,
			}
		}
		// add the order to the side
		obs.addOrder(o)
	}
	return obs
}

func getPopulatedTestSideWithOnlyPegs(side types.Side) *OrderBookSide {
	obs := getTestSide(side)

	type testOrder struct {
		ID     string
		Price  uint64
		Size   uint64
		Offset int64
	}

	testOrders := []testOrder{
		{"Order01", 100, 1, 5},
		{"Order02", 101, 1, 6},
		{"Order03", 102, 1, 7},
		{"Order04", 103, 1, 8},
	}

	for _, order := range testOrders {
		o := &types.Order{
			Id:          order.ID,
			MarketId:    "testmarket",
			PartyId:     "A",
			Side:        side,
			Price:       order.Price,
			Size:        order.Size,
			Remaining:   order.Size,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			PeggedOrder: &types.PeggedOrder{
				Reference: types.PeggedReference_PEGGED_REFERENCE_MID,
				Offset:    order.Offset,
			},
		}
		// add the order to the side
		obs.addOrder(o)
	}
	return obs
}

func getEmptyTestSide(side types.Side) *OrderBookSide {
	obs := getTestSide(types.Side_SIDE_SELL)
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

func TestExtractOrdersWrongVolume(t *testing.T) {
	// Attempt to extract more volume than we have
	side := getPopulatedTestSide(types.Side_SIDE_SELL)
	orders, err := side.ExtractOrders(101, 30)
	assert.Error(t, err)
	assert.Nil(t, orders)

	side = getPopulatedTestSide(types.Side_SIDE_SELL)
	orders, err = side.ExtractOrders(100, 4)
	assert.Error(t, err)
	assert.Nil(t, orders)
}

func TestBestStatic(t *testing.T) {
	// Empty book
	emptySide := getEmptyTestSide(types.Side_SIDE_SELL)
	_, err := emptySide.BestStaticPrice()
	assert.Error(t, err)

	_, _, err = emptySide.BestStaticPriceAndVolume()
	assert.Error(t, err)

	// Book with normal and pegs
	side := getPopulatedTestSideWithPegs(types.Side_SIDE_SELL)

	price, err := side.BestStaticPrice()
	assert.NoError(t, err)
	assert.EqualValues(t, 101, price)

	price, volume, err := side.BestStaticPriceAndVolume()
	assert.NoError(t, err)
	assert.EqualValues(t, 101, price)
	assert.EqualValues(t, 1, volume)

	// Book with only pegs
	pegsSide := getPopulatedTestSideWithOnlyPegs(types.Side_SIDE_SELL)
	_, err = pegsSide.BestStaticPrice()
	assert.Error(t, err)

	_, _, err = pegsSide.BestStaticPriceAndVolume()
	assert.Error(t, err)
}

func TestGetPriceLevelIfExists(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.Side_SIDE_BUY)
	sellSide := getPopulatedTestSideWithPegs(types.Side_SIDE_SELL)

	// Check we can get valid price levels
	bpl := buySide.getPriceLevelIfExists(100)
	assert.NotNil(t, bpl)
	spl := sellSide.getPriceLevelIfExists(100)
	assert.NotNil(t, spl)

	// Now try to get a level that does not exist
	bpl = buySide.getPriceLevelIfExists(200)
	assert.Nil(t, bpl)
	spl = sellSide.getPriceLevelIfExists(200)
	assert.Nil(t, spl)
}

func TestGetVolume(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.Side_SIDE_BUY)
	sellSide := getPopulatedTestSideWithPegs(types.Side_SIDE_SELL)

	// Actual levels
	volume, err := buySide.GetVolume(101)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, volume)

	volume, err = sellSide.GetVolume(101)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, volume)

	// Invalid levels
	volume, err = buySide.GetVolume(200)
	assert.Error(t, err)
	assert.EqualValues(t, 0, volume)

	volume, err = sellSide.GetVolume(200)
	assert.Error(t, err)
	assert.EqualValues(t, 0, volume)

	// Check total volumes
	totBuyVol := buySide.getTotalVolume()
	assert.EqualValues(t, 6, totBuyVol)

	totSellVol := buySide.getTotalVolume()
	assert.EqualValues(t, 6, totSellVol)
}

func TestFakeUncrossNormal(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.Side_SIDE_BUY)

	order := types.Order{
		Id:          "Id",
		Side:        types.Side_SIDE_SELL,
		Size:        5,
		Remaining:   5,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Type:        types.Order_TYPE_MARKET,
	}

	trades, err := buySide.fakeUncross(&order)
	assert.Len(t, trades, 5)
	assert.NoError(t, err)
}

func TestFakeUncrossSelfTrade(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.Side_SIDE_BUY)

	order := types.Order{
		Id:          "Id",
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Size:        5,
		Remaining:   5,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Type:        types.Order_TYPE_MARKET,
	}

	trades, err := buySide.fakeUncross(&order)
	assert.Len(t, trades, 0)
	assert.Error(t, err)
}

func TestFakeUncrossNotEnoughVolume(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.Side_SIDE_BUY)

	order := types.Order{
		Id:          "Id",
		Side:        types.Side_SIDE_SELL,
		Size:        7,
		Remaining:   7,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Type:        types.Order_TYPE_MARKET,
	}

	trades, err := buySide.fakeUncross(&order)
	assert.Len(t, trades, 0)
	assert.Error(t, err)
}
