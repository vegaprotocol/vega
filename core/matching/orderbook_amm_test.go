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

package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/matching/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderbookAMM(t *testing.T) {
	t.Run("test empty book and AMM", testEmptyBookAndAMM)
	t.Run("test empty book and matching AMM", testEmptyBookMatchingAMM)
	t.Run("test empty book and matching AMM with incoming FOK", testEmptyBookMatchingAMMFOK)
	t.Run("test matching between price levels", testMatchBetweenPriceLevels)
	t.Run("test matching with orders on both sides", testMatchOrdersBothSide)
	t.Run("test check book accounts for AMM volumes", testOnlyAMMOrders)
}

func testOnlyAMMOrders(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()
	sellPrice := num.NewUint(111)
	buyPrice := num.NewUint(95)
	// create some regular orders
	sell := createOrder(t, tst, 10, sellPrice)
	buy := createOrder(t, tst, 10, buyPrice)
	buy.Side = types.SideBuy
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(4).Return(nil)
	tst.obs.EXPECT().NotifyFinished().Times(3)
	_, err := tst.book.SubmitOrder(sell)
	require.NoError(t, err)
	_, err = tst.book.SubmitOrder(buy)
	require.NoError(t, err)
	// create some pegged orders
	pob := createOrder(t, tst, 100, nil)
	pob.Party = "B"
	pob.Side = types.SideBuy
	pob.PeggedOrder = &types.PeggedOrder{
		Reference: types.PeggedReferenceBestBid,
		Offset:    num.NewUint(10),
	}
	pos := createOrder(t, tst, 100, nil)
	pos.Party = "S"
	pos.PeggedOrder = &types.PeggedOrder{
		Reference: types.PeggedReferenceBestAsk,
		Offset:    num.NewUint(10),
	}
	require.NoError(t, err)
	_, err = tst.book.SubmitOrder(pos)
	require.NoError(t, err)
	require.Equal(t, uint64(1), tst.book.GetPeggedOrdersCount())
	// now cancel the non-pegged orders
	require.NoError(t, err)
	one, zero := uint64(1), uint64(0)

	// only buy orders
	tst.obs.EXPECT().BestPricesAndVolumes().Times(1).Return(num.UintOne(), one, nil, zero)
	check := tst.book.CheckBook()
	require.False(t, check)

	// buy and sell orders
	tst.obs.EXPECT().BestPricesAndVolumes().Times(1).Return(num.UintOne(), one, num.UintOne(), one)
	check = tst.book.CheckBook()
	require.True(t, check)
}

func testEmptyBookAndAMM(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()
	price := num.NewUint(100)

	// fake uncross
	o := createOrder(t, tst, 100, price)
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), nil, price).Times(1)
	tst.obs.EXPECT().NotifyFinished().Times(1)
	trades, err := tst.book.GetTrades(o)
	assert.NoError(t, err)
	assert.Len(t, trades, 0)

	// uncross
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), nil, price).Times(1)
	tst.obs.EXPECT().NotifyFinished().Times(1)
	conf, err := tst.book.SubmitOrder(o)
	assert.NoError(t, err)
	assert.Len(t, conf.PassiveOrdersAffected, 0)
	assert.Len(t, conf.Trades, 0)
}

func testEmptyBookMatchingAMM(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()
	price := num.NewUint(100)

	o := createOrder(t, tst, 1000, price)
	generated := createGeneratedOrders(t, tst, price)

	// fake uncross
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), nil, price).Times(1).Return(generated)
	tst.obs.EXPECT().NotifyFinished().Times(1)
	trades, err := tst.book.GetTrades(o)
	assert.NoError(t, err)
	assert.Len(t, trades, 2)

	// uncross
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), nil, price).Times(1).Return(generated)
	tst.obs.EXPECT().NotifyFinished().Times(1)
	conf, err := tst.book.SubmitOrder(o)
	assert.NoError(t, err)
	assertConf(t, conf, 2, 10)
}

func testEmptyBookMatchingAMMFOK(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()
	price := num.NewUint(100)

	o := createOrder(t, tst, 20, price)
	generated := createGeneratedOrders(t, tst, price)

	o.TimeInForce = types.OrderTimeInForceFOK

	// fake uncross
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), nil, price).Times(2).Return(generated)
	tst.obs.EXPECT().NotifyFinished().Times(2)
	trades, err := tst.book.GetTrades(o)
	assert.NoError(t, err)
	assert.Len(t, trades, 2)

	// uncross
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), nil, price).Times(2).Return(generated)
	tst.obs.EXPECT().NotifyFinished().Times(2)
	conf, err := tst.book.SubmitOrder(o)
	assert.NoError(t, err)
	assertConf(t, conf, 2, 10)
}

func testMatchBetweenPriceLevels(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()

	createPriceLevels(t, tst, 10,
		num.NewUint(100),
		num.NewUint(110),
		num.NewUint(120),
	)

	price := num.NewUint(90)
	size := uint64(1000)

	o := createOrder(t, tst, size, price)
	generated := createGeneratedOrders(t, tst, price)

	// price levels at 100, 110, 120, incoming order at 100
	// expect it to consume all volume at the three levels, and between each level we'll submit to offbook
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(4).Return(generated)
	tst.obs.EXPECT().NotifyFinished().Times(1)
	trades, err := tst.book.GetTrades(o)
	assert.NoError(t, err)

	// 3 trades with each price level, and then 2 trades from AMM in the intervals
	// (nil, 120) (120, 110) (110, 100) (100, 90)
	// so 3 + (2 * 4) = 11
	assert.Len(t, trades, 11)

	// uncross
	expectOffbookOrders(t, tst, price, nil, num.NewUint(120))
	expectOffbookOrders(t, tst, price, num.NewUint(120), num.NewUint(110))
	expectOffbookOrders(t, tst, price, num.NewUint(110), num.NewUint(100))
	expectOffbookOrders(t, tst, price, num.NewUint(100), num.NewUint(90))
	tst.obs.EXPECT().NotifyFinished().Times(1)

	conf, err := tst.book.SubmitOrder(o)
	assert.NoError(t, err)
	assertConf(t, conf, 11, 10)
}

func testMatchOrdersBothSide(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()

	createPriceLevels(t, tst, 10,
		num.NewUint(120),
		num.NewUint(110),
	)

	// this one will be on the opposite side of the book as price levels
	// sell order willing to sell at 130
	oPrice := uint64(130)
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	tst.obs.EXPECT().NotifyFinished().Times(1)
	o := createOrder(t, tst, 10, num.NewUint(oPrice))

	conf, err := tst.book.SubmitOrder(o)
	assert.NoError(t, err)
	assertConf(t, conf, 0, 0)

	price := num.NewUint(90)
	size := uint64(1000)
	o = createOrder(t, tst, size, price)
	generated := createGeneratedOrders(t, tst, price)

	// price levels at 100, 110, 120, incoming order at 100
	// expect it to consume all volume at the three levels, and between each level we'll submit to offbook
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(3).Return(generated)
	tst.obs.EXPECT().NotifyFinished().Times(1)
	trades, err := tst.book.GetTrades(o)
	assert.NoError(t, err)

	// 3 trades with each price level, and then 2 trades from AMM in the intervals
	// (nil, 120) (120, 110) (110, 100) (100, 90)
	// so 3 + (2 * 4) = 11
	assert.Len(t, trades, 8)

	// uncross
	expectOffbookOrders(t, tst, price, nil, num.NewUint(120))
	expectOffbookOrders(t, tst, price, num.NewUint(120), num.NewUint(110))
	expectOffbookOrders(t, tst, price, num.NewUint(110), num.NewUint(90))
	tst.obs.EXPECT().NotifyFinished().Times(1)

	conf, err = tst.book.SubmitOrder(o)
	assert.NoError(t, err)
	assertConf(t, conf, 8, 10)
}

func TestAMMOnlyBestPrices(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()

	tst.obs.EXPECT().BestPricesAndVolumes().Return(
		num.NewUint(1999),
		uint64(10),
		num.NewUint(2001),
		uint64(9),
	).AnyTimes()

	// Best
	price, err := tst.book.GetBestAskPrice()
	require.NoError(t, err)
	assert.Equal(t, "2001", price.String())

	price, err = tst.book.GetBestBidPrice()
	require.NoError(t, err)
	assert.Equal(t, "1999", price.String())

	// Best and volume
	price, volume, err := tst.book.BestOfferPriceAndVolume()
	require.NoError(t, err)
	assert.Equal(t, "2001", price.String())
	assert.Equal(t, uint64(9), volume)

	price, volume, err = tst.book.BestBidPriceAndVolume()
	require.NoError(t, err)
	assert.Equal(t, "1999", price.String())
	assert.Equal(t, uint64(10), volume)

	// Best static
	price, err = tst.book.GetBestStaticAskPrice()
	require.NoError(t, err)
	assert.Equal(t, "2001", price.String())

	price, err = tst.book.GetBestStaticBidPrice()
	require.NoError(t, err)
	assert.Equal(t, "1999", price.String())

	// Best static and volume
	price, volume, err = tst.book.GetBestStaticAskPriceAndVolume()
	require.NoError(t, err)
	assert.Equal(t, "2001", price.String())
	assert.Equal(t, uint64(9), volume)

	price, volume, err = tst.book.GetBestStaticBidPriceAndVolume()
	require.NoError(t, err)
	assert.Equal(t, "1999", price.String())
	assert.Equal(t, uint64(10), volume)
}

func TestIndicativeTradesAMMOnly(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()
	tst.obs.EXPECT().NotifyFinished().Times(1)

	expectCrossedAMMs(t, tst, 100, 150)
	tst.book.EnterAuction()

	ret := []*types.Order{createOrder(t, tst, 100, num.NewUint(100))}
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(o *types.Order, _, _ *num.Uint) []*types.Order {
			o.Remaining = 0
			return ret
		},
	)

	trades, err := tst.book.GetIndicativeTrades()
	require.NoError(t, err)
	assert.Equal(t, 26, len(trades))
}

func TestIndicativeTradesAMMOrderbookNotCrosses(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()
	tst.obs.EXPECT().NotifyFinished().Times(1)

	expectCrossedAMMs(t, tst, 100, 150)
	tst.book.EnterAuction()

	// submit an order each side outside of the crossed region
	o := createOrder(t, tst, 10, num.NewUint(90))
	o.Side = types.SideBuy
	_, err := tst.book.SubmitOrder(o)
	require.NoError(t, err)

	o = createOrder(t, tst, 10, num.NewUint(160))
	o.Side = types.SideSell
	_, err = tst.book.SubmitOrder(o)
	require.NoError(t, err)

	ret := []*types.Order{createOrder(t, tst, 100, num.NewUint(100))}
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(26).DoAndReturn(
		func(o *types.Order, _, _ *num.Uint) []*types.Order {
			o.Remaining = 0
			return ret
		},
	)

	trades, err := tst.book.GetIndicativeTrades()
	require.NoError(t, err)
	assert.Equal(t, 26, len(trades))
}

func TestIndicativeTradesAMMCrossedOrders(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()
	tst.obs.EXPECT().NotifyFinished().Times(1)

	expectCrossedAMMs(t, tst, 100, 150)
	tst.book.EnterAuction()

	// submit an order each side outside of the crossed region
	o := createOrder(t, tst, 10, num.NewUint(110))
	o.Side = types.SideBuy
	_, err := tst.book.SubmitOrder(o)
	require.NoError(t, err)

	o = createOrder(t, tst, 5, num.NewUint(125))
	o.Side = types.SideSell
	_, err = tst.book.SubmitOrder(o)
	require.NoError(t, err)

	o = createOrder(t, tst, 5, num.NewUint(126))
	o.Side = types.SideSell
	_, err = tst.book.SubmitOrder(o)
	require.NoError(t, err)

	ret := []*types.Order{createOrder(t, tst, 100, num.NewUint(100))}
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(ret)

	ret = []*types.Order{createOrder(t, tst, 5, num.NewUint(100))}
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(ret)

	ret = []*types.Order{createOrder(t, tst, 100, num.NewUint(100))}
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(23).Return(ret)

	trades, err := tst.book.GetIndicativeTrades()
	require.NoError(t, err)
	assert.Equal(t, 27, len(trades))
}

func TestUncrossedBookDoesNotExpandAMMs(t *testing.T) {
	tst := getTestOrderBookWithAMM(t)
	defer tst.ctrl.Finish()

	// AMM with buy at 99 and SELL at 101
	tst.obs.EXPECT().BestPricesAndVolumes().Return(num.NewUint(uint64(99)), uint64(10), num.NewUint(uint64(101)), uint64(10)).AnyTimes()

	// enter auction when not crossed we should not try to expand AMM's
	tst.book.EnterAuction()
	assert.Equal(t, "0", tst.book.GetIndicativePrice().String())
}

func assertConf(t *testing.T, conf *types.OrderConfirmation, n int, size uint64) {
	t.Helper()
	assert.Len(t, conf.PassiveOrdersAffected, n)
	assert.Len(t, conf.Trades, n)
	for i := range conf.Trades {
		assert.Equal(t, conf.Trades[i].Size, size)
		assert.Equal(t, conf.PassiveOrdersAffected[i].Remaining, uint64(0))
	}
}

func expectOffbookOrders(t *testing.T, tst *tstOrderbook, price, first, last *num.Uint) {
	t.Helper()
	generated := createGeneratedOrders(t, tst, price)
	tst.obs.EXPECT().SubmitOrder(gomock.Any(), first, last).Times(1).Return(generated)
}

func expectCrossedAMMs(t *testing.T, tst *tstOrderbook, min, max int) {
	t.Helper()
	tst.obs.EXPECT().BestPricesAndVolumes().Return(num.NewUint(uint64(max)), uint64(10), num.NewUint(uint64(min)), uint64(10)).AnyTimes()

	orders1 := createOrderbookShape(t, tst, min, max, types.SideBuy, "A")
	orders2 := createOrderbookShape(t, tst, min, max, types.SideSell, "B")

	tst.obs.EXPECT().OrderbookShape(gomock.Any(), gomock.Any(), gomock.Any()).Return(orders1, orders2)
}

type tstOrderbook struct {
	ctrl     *gomock.Controller
	book     *matching.CachedOrderBook
	obs      *mocks.MockOffbookSource
	marketID string
}

func createOrder(t *testing.T, tst *tstOrderbook, size uint64, price *num.Uint) *types.Order {
	t.Helper()
	return &types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      tst.marketID,
		Party:         "A",
		Side:          types.SideSell,
		Price:         price,
		OriginalPrice: price,
		Size:          size,
		Remaining:     size,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
}

func createGeneratedOrders(t *testing.T, tst *tstOrderbook, price *num.Uint) []*types.Order {
	t.Helper()

	orders := []*types.Order{}
	for i := 0; i < 2; i++ {
		o := createOrder(t, tst, 10, price)
		o.Side = types.OtherSide(o.Side)
		o.Party = "C"
		orders = append(orders, o)
	}

	return orders
}

func createPriceLevels(t *testing.T, tst *tstOrderbook, size uint64, levels ...*num.Uint) {
	t.Helper()

	tst.obs.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(len(levels))
	tst.obs.EXPECT().NotifyFinished().Times(len(levels))
	for _, l := range levels {
		o := createOrder(t, tst, size, l)
		o.Side = types.OtherSide(o.Side)
		o.Party = "B"
		conf, err := tst.book.SubmitOrder(o)
		require.NoError(t, err)
		require.Len(t, conf.Trades, 0)
	}
}

func createOrderbookShape(t *testing.T, tst *tstOrderbook, from, to int, side types.Side, party string) []*types.Order {
	t.Helper()

	orders := []*types.Order{}
	for i := from; i <= to; i++ {
		o := createOrder(t, tst, 10, num.NewUint(uint64(i)))
		o.GeneratedOffbook = true
		o.Side = side
		o.Party = party
		orders = append(orders, o)
	}
	return orders
}

func getTestOrderBookWithAMM(t *testing.T) *tstOrderbook {
	t.Helper()

	ctrl := gomock.NewController(t)
	obs := mocks.NewMockOffbookSource(ctrl)

	marketID := "testMarket"
	book := matching.NewCachedOrderBook(logging.NewTestLogger(), matching.NewDefaultConfig(), "testMarket", false, peggedOrderCounterForTest)
	book.SetOffbookSource(obs)

	return &tstOrderbook{
		ctrl:     ctrl,
		book:     book,
		obs:      obs,
		marketID: marketID,
	}
}
