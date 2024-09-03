// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

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
	side := getTestSide(types.SideSell)
	o := &types.Order{
		ID:            "order1",
		MarketID:      "testmarket",
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
	}
	// add the order to the side
	side.addOrder(o)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		ID:            "order2",
		MarketID:      "testmarket",
		Party:         "C",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
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
	side := getTestSide(types.SideSell)
	assert.Len(t, side.levels, 0)
	pl := side.getPriceLevel(num.NewUint(100))
	assert.Len(t, side.levels, 1)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(101))
	assert.Len(t, side.levels, 2)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(102))
	assert.Len(t, side.levels, 3)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(103))
	assert.Len(t, side.levels, 4)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(104))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = side.getPriceLevel(num.NewUint(102))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(100))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(104))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// test for a buy side
	side = getTestSide(types.SideBuy)
	assert.Len(t, side.levels, 0)
	pl = side.getPriceLevel(num.NewUint(100))
	assert.Len(t, side.levels, 1)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(101))
	assert.Len(t, side.levels, 2)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(102))
	assert.Len(t, side.levels, 3)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(103))
	assert.Len(t, side.levels, 4)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(104))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)

	// get existing one in bounds now
	pl = side.getPriceLevel(num.NewUint(102))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(100))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
	pl = side.getPriceLevel(num.NewUint(104))
	assert.Len(t, side.levels, 5)
	assert.NotNil(t, pl)
}

func TestMemoryAllocationPriceLevelUncrossSide(t *testing.T) {
	side := getTestSide(types.SideSell)
	o := &types.Order{
		ID:            "order1",
		MarketID:      "testmarket",
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
	}
	// add the order to the side
	side.addOrder(o)
	assert.Len(t, side.levels, 1)

	o2 := &types.Order{
		ID:            "order2",
		MarketID:      "testmarket",
		Party:         "C",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	// add the order to the side
	side.addOrder(o2)
	assert.Len(t, side.levels, 2)

	aggressiveOrder := &types.Order{
		ID:            "order3",
		MarketID:      "testmarket",
		Party:         "X",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
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
			ID:            order.ID,
			MarketID:      "testmarket",
			Party:         "A",
			Side:          side,
			Price:         num.NewUint(order.Price),
			OriginalPrice: num.NewUint(order.Price),
			Size:          order.Size,
			Remaining:     order.Size,
			TimeInForce:   types.OrderTimeInForceGTC,
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
		Offset uint64
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
			ID:            order.ID,
			MarketID:      "testmarket",
			Party:         "A",
			Side:          side,
			Price:         num.NewUint(order.Price),
			OriginalPrice: num.NewUint(order.Price),
			Size:          order.Size,
			Remaining:     order.Size,
			TimeInForce:   types.OrderTimeInForceGTC,
		}
		if order.Offset != 0 {
			o.PeggedOrder = &types.PeggedOrder{
				Reference: types.PeggedReferenceMid,
				Offset:    num.NewUint(order.Offset),
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
		Offset uint64
	}

	testOrders := []testOrder{
		{"Order01", 100, 1, 5},
		{"Order02", 101, 1, 6},
		{"Order03", 102, 1, 7},
		{"Order04", 103, 1, 8},
	}

	for _, order := range testOrders {
		o := &types.Order{
			ID:            order.ID,
			MarketID:      "testmarket",
			Party:         "A",
			Side:          side,
			Price:         num.NewUint(order.Price),
			OriginalPrice: num.NewUint(order.Price),
			Size:          order.Size,
			Remaining:     order.Size,
			TimeInForce:   types.OrderTimeInForceGTC,
			PeggedOrder: &types.PeggedOrder{
				Reference: types.PeggedReferenceMid,
				Offset:    num.NewUint(order.Offset),
			},
		}
		// add the order to the side
		obs.addOrder(o)
	}
	return obs
}

func getEmptyTestSide() *OrderBookSide {
	return getTestSide(types.SideSell)
}

func TestExtractOrdersFullLevel(t *testing.T) {
	side := getPopulatedTestSide(types.SideSell)

	assert.Len(t, side.levels, 2)

	orders := side.ExtractOrders(num.NewUint(100), 3, true)
	assert.Len(t, side.levels, 1)
	assert.Len(t, orders, 3)
	assert.EqualValues(t, 3, side.getOrderCount())
}

func TestExtractOrdersPartialLevel(t *testing.T) {
	side := getPopulatedTestSide(types.SideSell)

	assert.Len(t, side.levels, 2)

	orders := side.ExtractOrders(num.NewUint(100), 2, true)
	assert.Len(t, side.levels, 2)
	assert.Len(t, orders, 2)
	assert.EqualValues(t, 4, side.getOrderCount())
}

func TestExtractOrdersCrossLevel(t *testing.T) {
	side := getPopulatedTestSide(types.SideSell)

	assert.Len(t, side.levels, 2)

	orders := side.ExtractOrders(num.NewUint(101), 5, true)
	assert.Len(t, side.levels, 1)
	assert.Len(t, orders, 5)
	assert.EqualValues(t, 1, side.getOrderCount())
}

func TestExtractOrdersWrongVolume(t *testing.T) {
	// Attempt to extract more volume than we have on the book
	side := getPopulatedTestSide(types.SideSell)
	assert.Panics(t, func() { side.ExtractOrders(num.NewUint(101), 30, true) })

	// Attempt to extract more than we have at this price level
	side = getPopulatedTestSide(types.SideSell)
	assert.Panics(t, func() { side.ExtractOrders(num.NewUint(100), 4, true) })
}

func TestExtractOrdersZeroVolume(t *testing.T) {
	// Attempt to extract 0 volume of orders
	side := getPopulatedTestSide(types.SideSell)
	assert.Len(t, side.ExtractOrders(num.NewUint(101), 0, true), 0)
}

func TestBestStatic(t *testing.T) {
	// Empty book
	emptySide := getEmptyTestSide()
	_, err := emptySide.BestStaticPrice()
	assert.Error(t, err)

	_, _, err = emptySide.BestStaticPriceAndVolume()
	assert.Error(t, err)

	// Book with normal and pegs
	side := getPopulatedTestSideWithPegs(types.SideSell)

	price, err := side.BestStaticPrice()
	assert.NoError(t, err)
	assert.EqualValues(t, 101, int(price.Uint64()))

	price, volume, err := side.BestStaticPriceAndVolume()
	assert.NoError(t, err)
	assert.EqualValues(t, 101, int(price.Uint64()))
	assert.EqualValues(t, 1, volume)

	// Book with only pegs
	pegsSide := getPopulatedTestSideWithOnlyPegs(types.SideSell)
	_, err = pegsSide.BestStaticPrice()
	assert.Error(t, err)

	_, _, err = pegsSide.BestStaticPriceAndVolume()
	assert.Error(t, err)
}

func TestGetPriceLevelIfExists(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.SideBuy)
	sellSide := getPopulatedTestSideWithPegs(types.SideSell)

	// Check we can get valid price levels
	bpl := buySide.getPriceLevelIfExists(num.NewUint(100))
	assert.NotNil(t, bpl)
	spl := sellSide.getPriceLevelIfExists(num.NewUint(100))
	assert.NotNil(t, spl)

	// Now try to get a level that does not exist
	bpl = buySide.getPriceLevelIfExists(num.NewUint(200))
	assert.Nil(t, bpl)
	spl = sellSide.getPriceLevelIfExists(num.NewUint(200))
	assert.Nil(t, spl)
}

func TestGetVolume(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.SideBuy)
	sellSide := getPopulatedTestSideWithPegs(types.SideSell)

	// Actual levels
	volume, err := buySide.GetVolume(num.NewUint(101))
	assert.NoError(t, err)
	assert.EqualValues(t, 1, volume)

	volume, err = sellSide.GetVolume(num.NewUint(101))
	assert.NoError(t, err)
	assert.EqualValues(t, 1, volume)

	// Invalid levels
	volume, err = buySide.GetVolume(num.NewUint(200))
	assert.Error(t, err)
	assert.EqualValues(t, 0, volume)

	volume, err = sellSide.GetVolume(num.NewUint(200))
	assert.Error(t, err)
	assert.EqualValues(t, 0, volume)

	// Check total volumes
	totBuyVol := buySide.getTotalVolume()
	assert.EqualValues(t, 6, totBuyVol)

	totSellVol := buySide.getTotalVolume()
	assert.EqualValues(t, 6, totSellVol)
}

func TestFakeUncrossNormal(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.SideBuy)

	order := types.Order{
		ID:            "Id",
		Price:         num.UintZero(),
		OriginalPrice: num.UintZero(),
		Side:          types.SideSell,
		Size:          5,
		Remaining:     5,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeMarket,
	}

	checkWashTrades := false
	fakeTrades, err := buySide.fakeUncross(&order, checkWashTrades)
	assert.Len(t, fakeTrades, 5)
	assert.NoError(t, err)

	trades, _, _, err := buySide.uncross(&order, checkWashTrades)
	assert.Len(t, trades, 5)
	assert.NoError(t, err)

	for i := 0; i < len(trades); i++ {
		assert.Equal(t, trades[i], fakeTrades[i])
	}
}

func TestFakeUncrossSelfTradeFOKMarketOrder(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.SideBuy)

	order := types.Order{
		ID:            "Id",
		Party:         "A",
		Price:         num.UintZero(),
		OriginalPrice: num.UintZero(),
		Side:          types.SideSell,
		Size:          5,
		Remaining:     5,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeMarket,
	}

	checkWashTrades := false
	fakeTrades, err1 := buySide.fakeUncross(&order, checkWashTrades)
	assert.Len(t, fakeTrades, 0)
	assert.Error(t, err1)

	trades, _, _, err2 := buySide.uncross(&order, checkWashTrades)
	assert.Len(t, trades, 0)
	assert.Error(t, err2)

	assert.Equal(t, err1, err2)
}

func TestFakeUncrossSelfTradeNonFOKLimitOrder_DontCheckWashTrades(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.SideBuy)

	order := types.Order{
		ID:            "Id",
		Party:         "A",
		Price:         num.NewUint(105),
		OriginalPrice: num.NewUint(105),
		Side:          types.SideSell,
		Size:          5,
		Remaining:     5,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}

	checkWashTrades := false
	fakeTrades, err := buySide.fakeUncross(&order, checkWashTrades)
	assert.Len(t, fakeTrades, 1)
	assert.NoError(t, err)
	assert.Equal(t, fakeTrades[0].SellOrder, order.ID)

	trades, _, _, err := buySide.uncross(&order, checkWashTrades)
	assert.Len(t, trades, 1)
	assert.NoError(t, err)

	assert.Equal(t, trades[0], fakeTrades[0])
}

func TestFakeUncrossSelfTradeNonFOKLimitOrder_CheckWashTrades(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.SideBuy)

	order := types.Order{
		ID:            "Id",
		Party:         "A",
		Price:         num.NewUint(105),
		OriginalPrice: num.NewUint(105),
		Side:          types.SideSell,
		Size:          5,
		Remaining:     5,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}

	checkWashTrades := true
	fakeTrades, err1 := buySide.fakeUncross(&order, checkWashTrades)
	assert.Len(t, fakeTrades, 0)
	assert.Error(t, err1)
	assert.Equal(t, "party attempted to submit wash trade", err1.Error())

	trades, _, _, err2 := buySide.uncross(&order, checkWashTrades)
	assert.Len(t, trades, 0)
	assert.Error(t, err2)
	assert.Equal(t, "party attempted to submit wash trade", err2.Error())
	assert.Equal(t, err1.Error(), err2.Error())
}

func TestFakeUncrossNotEnoughVolume(t *testing.T) {
	buySide := getPopulatedTestSideWithPegs(types.SideBuy)

	order := types.Order{
		ID:            "Id",
		Price:         num.UintZero(),
		OriginalPrice: num.UintZero(),
		Side:          types.SideSell,
		Size:          7,
		Remaining:     7,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeMarket,
	}

	checkWashTrades := false
	fakeTrades, err := buySide.fakeUncross(&order, checkWashTrades)
	assert.Len(t, fakeTrades, 0)
	assert.NoError(t, err)

	trades, _, _, err := buySide.uncross(&order, checkWashTrades)
	assert.Len(t, trades, 0)
	assert.NoError(t, err)
}

func TestFakeUncrossAuction(t *testing.T) {
	buySide := getPopulatedTestSide(types.SideBuy)

	order1 := &types.Order{
		ID:            "Id",
		Party:         "A",
		Price:         num.NewUint(99),
		OriginalPrice: num.NewUint(99),
		Side:          types.SideSell,
		Size:          3,
		Remaining:     3,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}

	order2 := &types.Order{
		ID:            "Id",
		Party:         "B",
		Price:         num.NewUint(99),
		OriginalPrice: num.NewUint(99),
		Side:          types.SideSell,
		Size:          3,
		Remaining:     3,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}

	orders := []*types.Order{order1, order2}

	fakeTrades, err := buySide.fakeUncrossAuction(orders)
	assert.Len(t, fakeTrades, 6)
	assert.NoError(t, err)

	trades := []*types.Trade{}
	for _, order := range orders {
		trds, _, _, err := buySide.uncross(order, false)
		assert.NoError(t, err)
		trades = append(trades, trds...)
	}
	assert.Len(t, trades, 6)
	assert.NoError(t, err)

	for i, tr := range trades {
		assert.Equal(t, tr, fakeTrades[i])
	}
}
