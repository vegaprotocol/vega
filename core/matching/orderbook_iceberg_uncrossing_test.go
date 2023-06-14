// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package matching_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeIceberg(t *testing.T, orderbook *tstOB, market string, id string, side types.Side, price uint64, partyid string, size uint64) *types.Order {
	t.Helper()
	order := getOrder(t, market, id, side, price, partyid, size)
	order.IcebergOrder = &types.IcebergOrder{
		InitialPeakSize: 1,
		MinimumPeakSize: 1,
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
