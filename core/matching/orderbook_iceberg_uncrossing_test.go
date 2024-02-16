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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeIceberg(t *testing.T, orderbook *tstOB, market string, id string, side types.Side, price uint64, partyid string, size uint64) *types.Order {
	t.Helper()
	order := getOrder(t, market, id, side, price, partyid, size)
	order.IcebergOrder = &types.IcebergOrder{
		PeakSize:           1,
		MinimumVisibleSize: 1,
	}
	_, err := orderbook.ob.SubmitOrder(order)
	assert.NoError(t, err)
	return order
}

func requireUncrossedBook(t *testing.T, book *tstOB) {
	t.Helper()

	ask, err := book.ob.GetBestAskPrice()
	require.NoError(t, err)

	bid, err := book.ob.GetBestBidPrice()
	require.NoError(t, err)
	require.True(t, bid.LT(ask))
}

func assertTradeSizes(t *testing.T, trades []*types.Trade, sizes ...uint64) {
	t.Helper()
	require.Equal(t, len(sizes), len(trades))
	for i := range trades {
		assert.Equal(t, trades[i].Size, sizes[i])
	}
}

func makeIcebergForPanic(t *testing.T, orderbook *tstOB, market string, id string, side types.Side, price uint64, partyid string, size uint64) *types.Order {
	t.Helper()
	order := getOrder(t, market, id, side, price, partyid, size)
	order.IcebergOrder = &types.IcebergOrder{
		PeakSize:           3832,
		MinimumVisibleSize: 493,
	}
	_, err := orderbook.ob.SubmitOrder(order)
	assert.NoError(t, err)
	return order
}

// TestIcebergPanic is reproducing a bug observed in the market sim. It's skipping a few steps
// to make it minimal but it it's close enough. In summary there are 3 steps below:
// 1. the iceberg order is submitted with peak size of 3832 and size of 8400
// 2. the order is amended to decrease the size and change the price - hence going through ReplaceOrder
// 3. size offset only amendment - no price change - done with amendOrder.
func TestIcebergPanic(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	o1 := makeIcebergForPanic(t, book, market, crypto.RandomHash(), types.SideBuy, 101, "party01", 8400)
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 101, "party02", 10000)

	book.ob.GetIndicativeTrades()

	// decrease the size - not changing the peak size = 3832
	o2 := o1.Clone()
	o2.Size = 1569
	o2.Remaining = 1569
	o2.IcebergOrder.ReservedRemaining = 0
	book.ob.ReplaceOrder(o1, o2)

	// size offset of -512
	o3 := o2.Clone()
	o3.Size = 1057
	o3.Remaining = 1057
	o3.IcebergOrder.ReservedRemaining = 0
	book.ob.AmendOrder(o2, o3)

	book.ob.GetIndicativeTrades()
}

func TestIcebergExtractedSide(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// the iceberg order is on the side with the smallest uncrossing volume and should be
	// fully consumed after uncrossing
	o := makeIceberg(t, book, market, "BuyOrder01", types.SideBuy, 101, "party01", 20)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 98, "party01", 10)

	sell1 := makeOrder(t, book, market, "SellOrder01", types.SideSell, 100, "party02", 10)
	sell2 := makeOrder(t, book, market, "SellOrder02", types.SideSell, 101, "party02", 15)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(101))
	assert.Equal(t, volume, uint64(20))
	assert.Equal(t, side, types.SideBuy)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, price.Uint64(), uint64(101))

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	assertTradeSizes(t, trades, 10, 10)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	requireUncrossedBook(t, book)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 0)

	// the uncrossed order should be the iceberg and it is fully filled and traded
	// fully with sellf order 1, and half of sell order 2
	trades = uncrossedOrders[0].Trades
	assert.Equal(t, o.ID, uncrossedOrders[0].Order.ID)
	assertTradeSizes(t, trades, 10, 10)

	assert.Equal(t, types.OrderStatusFilled, sell1.Status)
	assert.Equal(t, types.OrderStatusActive, sell2.Status)
	assert.Equal(t, uint64(5), sell2.Remaining)
	assert.Equal(t, uint64(10), trades[0].Size)
	assert.Equal(t, uint64(10), trades[1].Size)
}

func TestIcebergAllPriceLevel(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// this order will be big enough to eat into all of the first two icebergs and some of a third at a different pricelevel
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 101, "party01", 20)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 98, "party01", 10)

	// We have two icebergs at one price level with a small peak
	sell1 := makeIceberg(t, book, market, "SellOrder01", types.SideSell, 100, "party02", 5)
	sell2 := makeIceberg(t, book, market, "SellOrder02", types.SideSell, 100, "party02", 5)
	sell3 := makeIceberg(t, book, market, "SellOrder03", types.SideSell, 101, "party02", 15)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder05", types.SideSell, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(101))
	assert.Equal(t, volume, uint64(20))
	assert.Equal(t, side, types.SideBuy)

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	assertTradeSizes(t, trades, 5, 5, 10)
	assert.Equal(t, 3, len(trades))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	requireUncrossedBook(t, book)
	assert.Equal(t, 1, len(uncrossedOrders))
	assert.Equal(t, 0, len(cancels))
	assertTradeSizes(t, uncrossedOrders[0].Trades, 5, 5, 10)

	// first two sell icebergs should be fully filled
	assert.Equal(t, types.OrderStatusFilled, sell1.Status)
	assert.Equal(t, uint64(0), sell1.TrueRemaining())
	assert.Equal(t, types.OrderStatusFilled, sell2.Status)
	assert.Equal(t, uint64(0), sell2.TrueRemaining())

	// and the third iceberg should be refreshed
	assert.Equal(t, types.OrderStatusActive, sell3.Status)
	assert.Equal(t, uint64(5), sell3.TrueRemaining())
	assert.Equal(t, uint64(1), sell3.Remaining)

	// check pricelevel count to be sure the sell side at 100 was removed
	assert.Equal(t, uint64(6), book.ob.GetOrderBookLevelCount())
}

func TestIcebergsDoubleProrata(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// this first order will take off all their peaks, and then 1 off each reserve
	_ = makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 101, "party01", 6)
	// this will then pro-rated take 1 of each reserve again, the icebergs won't refresh in between
	_ = makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 101, "party01", 6)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder05", types.SideBuy, 98, "party01", 10)

	// Populate sell side the three icebergs will be matched pro-rated, twice
	_ = makeIceberg(t, book, market, "SellOrder01", types.SideSell, 100, "party02", 10)
	_ = makeIceberg(t, book, market, "SellOrder02", types.SideSell, 100, "party02", 10)
	_ = makeIceberg(t, book, market, "SellOrder03", types.SideSell, 100, "party02", 10)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder05", types.SideSell, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, uint64(100), price.Uint64())
	assert.Equal(t, volume, uint64(12))
	assert.Equal(t, side, types.SideBuy)

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	assertTradeSizes(t, trades, 2, 2, 2, 2, 2, 2)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	requireUncrossedBook(t, book)
	assert.Equal(t, len(uncrossedOrders), 2)
	assert.Equal(t, len(cancels), 0)
	assertTradeSizes(t, uncrossedOrders[0].Trades, 2, 2, 2)
	assertTradeSizes(t, uncrossedOrders[1].Trades, 2, 2, 2)
	assert.Equal(t, 3, len(uncrossedOrders[0].PassiveOrdersAffected))
	assert.Equal(t, 3, len(uncrossedOrders[1].PassiveOrdersAffected))

	// check pricelevel count to be sure the empty ones were removed
	assert.Equal(t, uint64(5), book.ob.GetOrderBookLevelCount())
}

func TestIcebergsAndNormalOrders(t *testing.T) {
	// this is basically TestIcebergsDoubleProrata with some non-iceberg orders thrown into the uncrossing too
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// this first order will take off all their peaks, the non-iceberg order, and then 1 off each reserve
	_ = makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 101, "party01", 16)
	// this will then pro-rated take 1 of each reserve again, the icebergs won't refresh in between
	_ = makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 101, "party01", 6)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder05", types.SideBuy, 98, "party01", 10)

	// Populate sell side the three icebergs will be matched pro-rated, twice
	_ = makeIceberg(t, book, market, "SellOrder01", types.SideSell, 100, "party02", 10)
	_ = makeIceberg(t, book, market, "SellOrder02", types.SideSell, 100, "party02", 10)
	_ = makeIceberg(t, book, market, "SellOrder03", types.SideSell, 100, "party02", 10)
	_ = makeOrder(t, book, market, "SellOrder04", types.SideSell, 100, "party02", 10)
	makeOrder(t, book, market, "SellOrder05", types.SideSell, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder06", types.SideSell, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, uint64(100), price.Uint64())
	assert.Equal(t, volume, uint64(22))
	assert.Equal(t, side, types.SideBuy)

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	assertTradeSizes(t, trades, 2, 2, 2, 10, 2, 2, 2)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	requireUncrossedBook(t, book)
	assert.Equal(t, len(uncrossedOrders), 2)
	assert.Equal(t, len(cancels), 0)
	assertTradeSizes(t, uncrossedOrders[0].Trades, 2, 2, 2, 10)
	assertTradeSizes(t, uncrossedOrders[1].Trades, 2, 2, 2)
	assert.Equal(t, 4, len(uncrossedOrders[0].PassiveOrdersAffected))
	assert.Equal(t, 3, len(uncrossedOrders[1].PassiveOrdersAffected))

	// check pricelevel count to be sure the empty ones were removed
	assert.Equal(t, uint64(5), book.ob.GetOrderBookLevelCount())
}

func TestIcebergsAndNormalOrders2(t *testing.T) {
	// this is basically TestIcebergsDoubleProrata with some non-iceberg orders thrown into the uncrossing too
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// this first order will take off all their peaks, the non-iceberg order, and then 1 off each reserve
	_ = makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 101, "party01", 40)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder05", types.SideBuy, 98, "party01", 10)

	// Populate sell side the three icebergs will be matched pro-rated, twice
	_ = makeIceberg(t, book, market, "SellOrder01", types.SideSell, 100, "party02", 10)
	_ = makeIceberg(t, book, market, "SellOrder02", types.SideSell, 100, "party02", 10)
	_ = makeIceberg(t, book, market, "SellOrder03", types.SideSell, 100, "party02", 10)
	_ = makeOrder(t, book, market, "SellOrder04", types.SideSell, 100, "party02", 10)
	makeOrder(t, book, market, "SellOrder05", types.SideSell, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder06", types.SideSell, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, uint64(100), price.Uint64())
	assert.Equal(t, volume, uint64(40))
	assert.Equal(t, side, types.SideBuy)

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	assertTradeSizes(t, trades, 10, 10, 10, 10)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	requireUncrossedBook(t, book)
	assert.Equal(t, 1, len(uncrossedOrders))
	assert.Equal(t, len(cancels), 0)
	assertTradeSizes(t, uncrossedOrders[0].Trades, 10, 10, 10, 10)
	assert.Equal(t, 4, len(uncrossedOrders[0].PassiveOrdersAffected))

	// check pricelevel count to be sure the empty ones were removed
	assert.Equal(t, uint64(4), book.ob.GetOrderBookLevelCount())
}
