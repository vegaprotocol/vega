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
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// launch aggressiveOrder orders from both sides to fully clear the order book.
type aggressiveOrderScenario struct {
	aggressiveOrder               *types.Order
	expectedPassiveOrdersAffected []types.Order
	expectedTrades                []types.Trade
}

func peggedOrderCounterForTest(int64) {}

type tstOB struct {
	ob  *matching.CachedOrderBook
	log *logging.Logger
}

func (t *tstOB) Finish() {
	t.log.Sync()
}

func getCurrentUtcTimestampNano() int64 {
	return time.Now().UnixNano()
}

func getTestOrderBook(t *testing.T, market string) *tstOB {
	t.Helper()
	tob := tstOB{
		log: logging.NewTestLogger(),
	}
	tob.ob = matching.NewCachedOrderBook(tob.log, matching.NewDefaultConfig(), market, false, peggedOrderCounterForTest)

	tob.ob.LogRemovedOrdersDebug = true
	return &tob
}

func TestOrderBook_GetClosePNL(t *testing.T) {
	t.Run("Get Buy-side close PNL values", getClosePNLBuy)
	t.Run("Get Sell-side close PNL values", getClosePNLSell)
	t.Run("Get Incomplete close-out-pnl (check error) - Buy", getClosePNLIncompleteBuy)
	t.Run("Get Incomplete close-out-pnl (check error) - Sell", getClosePNLIncompleteSell)
	t.Run("Get Best bid price and volume", testBestBidPriceAndVolume)
	t.Run("Get Best offer price and volume", testBestOfferPriceAndVolume)
}

func TestOrderBook_CancelBulk(t *testing.T) {
	t.Run("Cancel all order for a party", cancelAllOrderForAParty)
	t.Run("Get all order for a party", getAllOrderForAParty)
	t.Run("Party with no order cancel nothing", partyWithNoOrderCancelNothing)
}

func TestGetVolumeAtPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	buyPrices := []uint64{
		90,
		85,
		80,
		75,
		70,
		65,
		50,
	}
	sellPrices := []uint64{
		100,
		105,
		110,
		120,
		125,
		130,
		150,
	}
	// populate a book with buy orders ranging between 50 and 90
	// sell orders starting at 100, up to 150. All orders have a size of 2
	orders := getTestOrders(t, market, 2, buyPrices, sellPrices)
	for _, o := range orders {
		_, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
	}
	t.Run("Getting volume at price with a single price level returns the volume for that price level", func(t *testing.T) {
		// check the buy side
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[0]), types.SideBuy)
		require.Equal(t, uint64(2), v)
		// check the sell side
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[0]), types.SideSell)
		require.Equal(t, uint64(2), v)
	})
	t.Run("Getting volume at price containing all price levels returns the total volume on that side of the book", func(t *testing.T) {
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[len(buyPrices)-1]), types.SideBuy)
		exp := uint64(len(buyPrices) * 2)
		require.Equal(t, exp, v)
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[len(sellPrices)-1]), types.SideSell)
		exp = uint64(len(sellPrices) * 2)
		require.Equal(t, exp, v)
	})
	t.Run("Getting volume at a price that is out of range returns zero volume", func(t *testing.T) {
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[0]+1), types.SideBuy)
		require.Equal(t, uint64(0), v)
		// check the sell side
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[0]-1), types.SideSell)
		require.Equal(t, uint64(0), v)
	})
	t.Run("Getting volume at price allowing for more than all price levels returns the total volume on that side of the book", func(t *testing.T) {
		// lowest buy order -1
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[len(buyPrices)-1]-1), types.SideBuy)
		exp := uint64(len(buyPrices) * 2)
		require.Equal(t, exp, v)
		// highest sell order on the book +1
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[len(sellPrices)-1]+1), types.SideSell)
		exp = uint64(len(sellPrices) * 2)
		require.Equal(t, exp, v)
	})
	t.Run("Getting volume at a price that is somewhere in the middle returns the correct volume", func(t *testing.T) {
		idx := len(buyPrices) / 2
		// remember: includes 0 idx
		exp := uint64(idx*2 + 2)
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[idx]), types.SideBuy)
		require.Equal(t, exp, v)
		idx = len(sellPrices) / 2
		exp = uint64(idx*2 + 2)
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[idx]), types.SideSell)
		require.Equal(t, exp, v)
	})
}

func TestGetVolumeAtPriceIceberg(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	buyPrices := []uint64{
		90,
		85,
		80,
		75,
		70,
		65,
		50,
	}
	sellPrices := []uint64{
		100,
		105,
		110,
		120,
		125,
		130,
		150,
	}
	// populate a book with buy orders ranging between 50 and 90
	// sell orders starting at 100, up to 150. All orders are iceberg orders with a visible size of 2, hidden size of 2.
	orders := getTestOrders(t, market, 4, buyPrices, sellPrices)
	for _, o := range orders {
		// make this an iceberg order
		o.IcebergOrder = &types.IcebergOrder{
			PeakSize:           2,
			MinimumVisibleSize: 2,
		}
		_, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
	}
	t.Run("Getting volume at price with a single price level returns the volume for that price level", func(t *testing.T) {
		// check the buy side
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[0]), types.SideBuy)
		require.Equal(t, uint64(4), v)
		// check the sell side
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[0]), types.SideSell)
		require.Equal(t, uint64(4), v)
	})
	t.Run("Getting volume at price containing all price levels returns the total volume on that side of the book", func(t *testing.T) {
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[len(buyPrices)-1]), types.SideBuy)
		exp := uint64(len(buyPrices) * 4)
		require.Equal(t, exp, v)
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[len(sellPrices)-1]), types.SideSell)
		exp = uint64(len(sellPrices) * 4)
		require.Equal(t, exp, v)
	})
	t.Run("Getting volume at a price that is out of range returns zero volume", func(t *testing.T) {
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[0]+1), types.SideBuy)
		require.Equal(t, uint64(0), v)
		// check the sell side
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[0]-1), types.SideSell)
		require.Equal(t, uint64(0), v)
	})
	t.Run("Getting volume at price allowing for more than all price levels returns the total volume on that side of the book", func(t *testing.T) {
		// lowest buy order -1
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[len(buyPrices)-1]-1), types.SideBuy)
		exp := uint64(len(buyPrices) * 4)
		require.Equal(t, exp, v)
		// highest sell order on the book +1
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[len(sellPrices)-1]+1), types.SideSell)
		exp = uint64(len(sellPrices) * 4)
		require.Equal(t, exp, v)
	})
	t.Run("Getting volume at a price that is somewhere in the middle returns the correct volume", func(t *testing.T) {
		idx := len(buyPrices) / 2
		// remember: includes 0 idx
		exp := uint64(idx*4 + 4)
		v := book.ob.GetVolumeAtPrice(num.NewUint(buyPrices[idx]), types.SideBuy)
		require.Equal(t, exp, v)
		idx = len(sellPrices) / 2
		exp = uint64(idx*4 + 4)
		v = book.ob.GetVolumeAtPrice(num.NewUint(sellPrices[idx]), types.SideSell)
		require.Equal(t, exp, v)
	})
}

func TestHash(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orders := []*types.Order{
		{
			ID:            "1111111111111111111111",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(10),
			OriginalPrice: num.NewUint(10),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            "2222222222222222222222",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(30),
			OriginalPrice: num.NewUint(30),
			Size:          5,
			Remaining:     5,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            "3333333333333333333333",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideSell,
			Price:         num.NewUint(200),
			OriginalPrice: num.NewUint(200),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            "4444444444444444444444",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideSell,
			Price:         num.NewUint(400),
			OriginalPrice: num.NewUint(400),
			Size:          10,
			Remaining:     10,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
	}

	for _, o := range orders {
		_, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
	}

	hash := book.ob.Hash()
	require.Equal(t,
		"fc0073b33273253dd021d4fdd330e00c32c8b484c8bb484abac92acfb9d575bf",
		hex.EncodeToString(hash),
		"It should match against the known hash",
	)
	// compute the hash 100 times for determinism verification
	for i := 0; i < 100; i++ {
		got := book.ob.Hash()
		require.Equal(t, hash, got)
	}
}

func cancelAllOrderForAParty(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orderID1 := vgcrypto.RandomHash()
	orderID2 := vgcrypto.RandomHash()
	orderID3 := vgcrypto.RandomHash()
	orderID4 := vgcrypto.RandomHash()

	orders := []*types.Order{
		{
			ID:            orderID1,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            orderID2,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          5,
			Remaining:     5,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            orderID3,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(200),
			OriginalPrice: num.NewUint(200),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            orderID4,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          10,
			Remaining:     10,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	confs, err := book.ob.CancelAllOrders("A")
	assert.NoError(t, err)
	assert.Len(t, confs, 3)
	expectedIDs := map[string]struct{}{
		orderID1: {},
		orderID2: {},
		orderID4: {},
	}
	for _, conf := range confs {
		if _, ok := expectedIDs[conf.Order.ID]; ok {
			delete(expectedIDs, conf.Order.ID)
		} else {
			t.Fatalf("unexpected order has been cancelled %v", conf.Order)
		}
	}
}

func getAllOrderForAParty(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orderID1 := vgcrypto.RandomHash()
	orderID2 := vgcrypto.RandomHash()
	orderID3 := vgcrypto.RandomHash()
	orderID4 := vgcrypto.RandomHash()

	orders := []*types.Order{
		{
			ID:            orderID1,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            orderID2,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          5,
			Remaining:     5,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            orderID3,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(200),
			OriginalPrice: num.NewUint(200),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            orderID4,
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          10,
			Remaining:     10,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	ordersLs := book.ob.GetOrdersPerParty("A")
	assert.Len(t, ordersLs, 3)
	expectedIDs := map[string]struct{}{
		orderID1: {},
		orderID2: {},
		orderID4: {},
	}
	for _, o := range ordersLs {
		if _, ok := expectedIDs[o.ID]; ok {
			delete(expectedIDs, o.ID)
		} else {
			t.Fatalf("unexpected order has been cancelled %v", o)
		}
	}
}

func partyWithNoOrderCancelNothing(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	orders := []*types.Order{
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          5,
			Remaining:     5,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(200),
			OriginalPrice: num.NewUint(200),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          10,
			Remaining:     10,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
	}
	for _, o := range orders {
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
	}
	ordersLs := book.ob.GetOrdersPerParty("X")
	assert.Len(t, ordersLs, 0)
}

func testBestBidPriceAndVolume(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          5,
			Remaining:     5,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(200),
			OriginalPrice: num.NewUint(200),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "d",
			Side:          types.SideBuy,
			Price:         num.NewUint(300),
			OriginalPrice: num.NewUint(300),
			Size:          10,
			Remaining:     10,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
	}
	for _, o := range orders {
		trades, getErr := book.ob.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(confirm.Trades), len(trades))
	}

	price, volume, err := book.ob.BestBidPriceAndVolume()
	assert.NoError(t, err)
	assert.Equal(t, uint64(300), price.Uint64())
	assert.Equal(t, uint64(15), volume)
}

func testBestOfferPriceAndVolume(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideSell,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideSell,
			Price:         num.NewUint(10),
			OriginalPrice: num.NewUint(10),
			Size:          5,
			Remaining:     5,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideSell,
			Price:         num.NewUint(200),
			OriginalPrice: num.NewUint(200),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "d",
			Side:          types.SideSell,
			Price:         num.NewUint(10),
			OriginalPrice: num.NewUint(10),
			Size:          10,
			Remaining:     10,
			TimeInForce:   types.OrderTimeInForceGTC,
		},
	}
	for _, o := range orders {
		trades, getErr := book.ob.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
	}

	price, volume, err := book.ob.BestOfferPriceAndVolume()
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), price.Uint64())
	assert.Equal(t, uint64(15), volume)
}

func getClosePNLIncompleteBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(110),
			OriginalPrice: num.NewUint(110),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.ob.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		2: 210 / 2,
		1: 110,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.ob.GetCloseoutPrice(vol, types.SideBuy)
		assert.Equal(t, exp, price.Uint64())
		assert.NoError(t, err)
	}
	price, err := book.ob.GetCloseoutPrice(3, types.SideBuy)
	assert.Equal(t, callExp[2], price.Uint64())
	assert.Equal(t, matching.ErrNotEnoughOrders, err)
}

func getClosePNLIncompleteSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideSell,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideSell,
			Price:         num.NewUint(110),
			OriginalPrice: num.NewUint(110),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.ob.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		2: 210 / 2,
		1: 100,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.ob.GetCloseoutPrice(vol, types.SideSell)
		assert.Equal(t, exp, price.Uint64())
		assert.NoError(t, err)
	}
	price, err := book.ob.GetCloseoutPrice(3, types.SideSell)
	assert.Equal(t, callExp[2], price.Uint64())
	assert.Equal(t, matching.ErrNotEnoughOrders, err)
}

func getClosePNLBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideBuy,
			Price:         num.NewUint(110),
			OriginalPrice: num.NewUint(110),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "C",
			Side:          types.SideBuy,
			Price:         num.NewUint(120),
			OriginalPrice: num.NewUint(120),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.ob.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		3: 330 / 3,
		2: 230 / 2,
		1: 120,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.ob.GetCloseoutPrice(vol, types.SideBuy)
		assert.Equal(t, exp, price.Uint64())
		assert.NoError(t, err)
	}
}

func getClosePNLSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	// 3 orders of size 1, 3 different prices
	orders := []*types.Order{
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideSell,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideSell,
			Price:         num.NewUint(110),
			OriginalPrice: num.NewUint(110),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "C",
			Side:          types.SideSell,
			Price:         num.NewUint(120),
			OriginalPrice: num.NewUint(120),
			Size:          1,
			Remaining:     1,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
	}
	for _, o := range orders {
		trades, getErr := book.ob.GetTrades(o)
		assert.NoError(t, getErr)
		confirm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(confirm.Trades))
		assert.Equal(t, len(trades), len(confirm.Trades))
	}
	// volume + expected price
	callExp := map[uint64]uint64{
		3: 330 / 3,
		2: 210 / 2,
		1: 100,
	}
	// this calculates the actual volume
	for vol, exp := range callExp {
		price, err := book.ob.GetCloseoutPrice(vol, types.SideSell)
		assert.NoError(t, err)
		assert.Equal(t, exp, price.Uint64())
	}
}

func TestOrderBook_CancelReturnsTheOrderFromTheBook(t *testing.T) {
	market := "cancel-returns-order"
	party := "p1"

	book := getTestOrderBook(t, market)
	defer book.Finish()
	currentTimestamp := getCurrentUtcTimestampNano()

	orderID := vgcrypto.RandomHash()
	order1 := types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(1),
		OriginalPrice: num.NewUint(1),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     currentTimestamp,
		ID:            orderID,
	}
	order2 := types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(1),
		OriginalPrice: num.NewUint(1),
		Size:          100,
		Remaining:     1, // use a wrong remaining here to get the order from the book
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     currentTimestamp,
		ID:            orderID,
	}

	trades, getErr := book.ob.GetTrades(&order1)
	assert.NoError(t, getErr)
	confirm, err := book.ob.SubmitOrder(&order1)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	o, err := book.ob.CancelOrder(&order2)
	assert.Equal(t, err, nil)
	assert.Equal(t, o.Order.Remaining, order1.Remaining)
}

func TestOrderBook_RemoveExpiredOrders(t *testing.T) {
	market := "expiringOrderBookTest"
	party := "clay-davis"

	book := getTestOrderBook(t, market)
	defer book.Finish()
	currentTimestamp := getCurrentUtcTimestampNano()
	someTimeLater := currentTimestamp + (1000 * 1000)

	order1 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(1),
		OriginalPrice: num.NewUint(1),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		CreatedAt:     currentTimestamp,
		ExpiresAt:     someTimeLater,
		ID:            "1",
	}
	trades, getErr := book.ob.GetTrades(order1)
	assert.NoError(t, getErr)
	confirm, err := book.ob.SubmitOrder(order1)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order2 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(3298),
		OriginalPrice: num.NewUint(3298),
		Size:          99,
		Remaining:     99,
		TimeInForce:   types.OrderTimeInForceGTT,
		CreatedAt:     currentTimestamp,
		ExpiresAt:     someTimeLater + 1,
		ID:            "2",
	}
	trades, getErr = book.ob.GetTrades(order2)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order2)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order3 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(771),
		OriginalPrice: num.NewUint(771),
		Size:          19,
		Remaining:     19,
		TimeInForce:   types.OrderTimeInForceGTT,
		CreatedAt:     currentTimestamp,
		ExpiresAt:     someTimeLater,
		ID:            "3",
	}
	trades, getErr = book.ob.GetTrades(order3)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order3)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order4 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(1000),
		OriginalPrice: num.NewUint(1000),
		Size:          7,
		Remaining:     7,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     currentTimestamp,
		ID:            "4",
	}
	trades, getErr = book.ob.GetTrades(order4)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order4)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order5 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(199),
		OriginalPrice: num.NewUint(199),
		Size:          99999,
		Remaining:     99999,
		TimeInForce:   types.OrderTimeInForceGTT,
		CreatedAt:     currentTimestamp,
		ExpiresAt:     someTimeLater,
		ID:            "5",
	}

	trades, getErr = book.ob.GetTrades(order5)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order5)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order6 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     currentTimestamp,
		ID:            "6",
	}
	trades, getErr = book.ob.GetTrades(order6)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order6)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order7 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(41),
		OriginalPrice: num.NewUint(41),
		Size:          9999,
		Remaining:     9999,
		TimeInForce:   types.OrderTimeInForceGTT,
		CreatedAt:     currentTimestamp,
		ExpiresAt:     someTimeLater + 9999,
		ID:            "7",
	}
	trades, getErr = book.ob.GetTrades(order7)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order7)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order8 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(1),
		OriginalPrice: num.NewUint(1),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		CreatedAt:     currentTimestamp,
		ExpiresAt:     someTimeLater - 9999,
		ID:            "8",
	}
	trades, getErr = book.ob.GetTrades(order8)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order8)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order9 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(65),
		OriginalPrice: num.NewUint(65),
		Size:          12,
		Remaining:     12,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     currentTimestamp,
		ID:            "9",
	}
	trades, getErr = book.ob.GetTrades(order9)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order9)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))

	order10 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         party,
		Side:          types.SideSell,
		Price:         num.NewUint(1),
		OriginalPrice: num.NewUint(1),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		CreatedAt:     currentTimestamp,
		ExpiresAt:     someTimeLater - 1,
		ID:            "10",
	}
	trades, getErr = book.ob.GetTrades(order10)
	assert.NoError(t, getErr)
	confirm, err = book.ob.SubmitOrder(order10)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(trades), len(confirm.Trades))
}

// test for order validation.
func TestOrderBook_SubmitOrder2WithValidation(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()
	timeStampOrder := types.Order{
		Status:    types.OrderStatusActive,
		Type:      types.OrderTypeLimit,
		ID:        "timestamporderID",
		MarketID:  market,
		Party:     "A",
		CreatedAt: 10,
		Side:      types.SideBuy,
		Size:      1,
		Remaining: 1,
	}
	trades, getErr := book.ob.GetTrades(&timeStampOrder)
	assert.NoError(t, getErr)
	confirm, err := book.ob.SubmitOrder(&timeStampOrder)
	assert.NoError(t, err)
	assert.Equal(t, len(trades), len(confirm.Trades))
	// cancel order again, just so we set the timestamp as expected
	book.ob.CancelOrder(&timeStampOrder)

	invalidRemainingSizeOrdertypes := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     300,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
		ID:            "id-number-one",
	}
	trades, getErr = book.ob.GetTrades(invalidRemainingSizeOrdertypes)
	_, err = book.ob.SubmitOrder(invalidRemainingSizeOrdertypes)
	assert.Equal(t, err, getErr)
	assert.Equal(t, types.OrderErrorInvalidRemainingSize, err)
	assert.Equal(t, 0, len(trades))
}

func TestOrderBook_DeleteOrder(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
	}

	trades, err := book.ob.GetTrades(newOrder)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(trades))
	book.ob.SubmitOrder(newOrder)

	_, err = book.ob.DeleteOrder(newOrder)
	require.NoError(t, err)

	book.ob.PrintState("AFTER REMOVE ORDER")
}

func TestOrderBook_RemoveOrder(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
	}

	trades, err := book.ob.GetTrades(newOrder)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(trades))
	book.ob.SubmitOrder(newOrder)

	// First time we remove the order it should succeed
	_, err = book.ob.RemoveOrder(newOrder.ID)
	assert.Error(t, err, vega.ErrInvalidOrderID)

	// Second time we try to remove the order it should fail
	_, err = book.ob.RemoveOrder(newOrder.ID)
	assert.Error(t, err)
}

func TestOrderBook_SubmitOrderInvalidMarket(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      "invalid",
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
		ID:            vgcrypto.RandomHash(),
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.Error(t, getErr)
	assert.Equal(t, 0, len(trades))
	_, err := book.ob.SubmitOrder(newOrder)
	require.Error(t, err)
	assert.ErrorIs(t, err, types.OrderErrorInvalidMarketID)
	assert.ErrorIs(t, err, getErr)
}

func TestOrderBook_CancelSellOrder(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// Arrange
	id := vgcrypto.RandomHash()
	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
		ID:            id,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order
	assert.Equal(t, len(trades), len(confirmation.Trades))

	// Act
	res, err := book.ob.CancelOrder(orderAdded)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, id, res.Order.ID)
	assert.Equal(t, types.OrderStatusCancelled, res.Order.Status)

	book.ob.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelBuyOrder(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING VALID ORDER")

	// Arrange
	id := vgcrypto.RandomHash()
	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
		ID:            id,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)
	assert.NoError(t, err)
	assert.Equal(t, len(trades), len(confirmation.Trades))
	orderAdded := confirmation.Order

	// Act
	res, err := book.ob.CancelOrder(orderAdded)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, id, res.Order.ID)
	assert.Equal(t, types.OrderStatusCancelled, res.Order.Status)

	book.ob.PrintState("AFTER CANCEL ORDER")
}

func TestOrderBook_CancelOrderByID(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING VALID ORDER BY ID")

	id := vgcrypto.RandomHash()
	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     0,
		ID:            id,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)
	assert.NotNil(t, confirmation, "submit order should succeed")
	assert.NoError(t, err, "submit order should succeed")
	orderAdded := confirmation.Order
	assert.NotNil(t, orderAdded, "submitted order is expected to be valid")
	assert.Equal(t, len(trades), len(confirmation.Trades))

	orderFound, err := book.ob.GetOrderByID(orderAdded.ID)
	assert.NotNil(t, orderFound, "order lookup should work for the order just submitted")
	assert.NoError(t, err, "order lookup should not fail")

	res, err := book.ob.CancelOrder(orderFound)
	assert.NotNil(t, res, "cancelling should work for a valid order that was just found")
	assert.NoError(t, err, "order cancel should not fail")

	orderFound, err = book.ob.GetOrderByID(orderAdded.ID)
	assert.Error(t, err, "order lookup for an already cancelled order should fail")
	assert.Nil(t, orderFound, "order lookup for an already cancelled order should not be possible")

	book.ob.PrintState("AFTER CANCEL ORDER BY ID")
}

func TestOrderBook_CancelOrderMarketMismatch(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING MARKET MISMATCH ORDER")

	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()
	newOrder := &types.Order{
		Status:    types.OrderStatusActive,
		Type:      types.OrderTypeLimit,
		MarketID:  market,
		ID:        vgcrypto.RandomHash(),
		Party:     "A",
		Size:      100,
		Remaining: 100,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order
	assert.Equal(t, len(trades), len(confirmation.Trades))

	orderAdded.MarketID = "invalid" // Bad market, malformed?

	assert.Panics(t, func() { _, err = book.ob.CancelOrder(orderAdded) })
}

func TestOrderBook_CancelOrderInvalidID(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN CANCELLING INVALID ORDER")

	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()
	newOrder := &types.Order{
		Status:    types.OrderStatusActive,
		Type:      types.OrderTypeLimit,
		MarketID:  market,
		ID:        "id",
		Party:     "A",
		Size:      100,
		Remaining: 100,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)
	assert.NoError(t, err)
	orderAdded := confirmation.Order
	assert.Equal(t, len(trades), len(confirmation.Trades))

	_, err = book.ob.CancelOrder(orderAdded)
	if err != nil {
		logger.Debug("error cancelling order", logging.Error(err))
	}

	assert.Equal(t, types.OrderErrorInvalidOrderID, err)
}

func expectTrade(t *testing.T, expectedTrade, trade *types.Trade) {
	t.Helper()
	// run asserts for protocol trade data
	assert.Equal(t, expectedTrade.Type, trade.Type, "invalid trade type")
	assert.Equal(t, int(expectedTrade.Price.Uint64()), int(trade.Price.Uint64()), "invalid trade price")
	assert.Equal(t, int(expectedTrade.Size), int(trade.Size), "invalid trade size")
	assert.Equal(t, expectedTrade.Buyer, trade.Buyer, "invalid trade buyer")
	assert.Equal(t, expectedTrade.Seller, trade.Seller, "invalid trade seller")
	assert.Equal(t, expectedTrade.Aggressor, trade.Aggressor, "invalid trade aggressor")
}

func expectOrder(t *testing.T, expectedOrder, order *types.Order) {
	t.Helper()
	// run asserts for order
	assert.Equal(t, expectedOrder.MarketID, order.MarketID, "invalid order market id")
	assert.Equal(t, expectedOrder.Party, order.Party, "invalid order party id")
	assert.Equal(t, expectedOrder.Side, order.Side, "invalid order side")
	assert.Equal(t, int(expectedOrder.Price.Uint64()), int(order.Price.Uint64()), "invalid order price")
	assert.Equal(t, int(expectedOrder.Size), int(order.Size), "invalid order size")
	assert.Equal(t, int(expectedOrder.Remaining), int(order.Remaining), "invalid order remaining")
	assert.Equal(t, expectedOrder.TimeInForce, order.TimeInForce, "invalid order tif")
	assert.Equal(t, expectedOrder.CreatedAt, order.CreatedAt, "invalid order created at")
}

func TestOrderBook_AmendOrder(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	confirmation, err := book.ob.SubmitOrder(newOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	err = book.ob.AmendOrder(newOrder, editedOrder)
	if err != nil {
		t.Log(err)
	}

	assert.Nil(t, err)
}

func TestOrderBook_AmendOrderInvalidRemaining(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Party:         "A",
		ID:            "123456",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          100,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}
	err = book.ob.AmendOrder(newOrder, editedOrder)
	if err != types.OrderErrorInvalidRemainingSize {
		t.Log(err)
	}

	assert.Equal(t, types.OrderErrorInvalidRemainingSize, err)
}

func TestOrderBook_AmendOrderInvalidAmend(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	confirmation, err := book.ob.SubmitOrder(newOrder)
	assert.Equal(t, err, getErr)
	assert.Equal(t, 0, len(trades))

	fmt.Printf("confirmation : %+v", confirmation)

	editedOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Party:         "A",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	err = book.ob.AmendOrder(newOrder, editedOrder)
	assert.Equal(t, types.OrderErrorNotFound, err)
}

func TestOrderBook_AmendOrderInvalidAmend1(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMENDING ORDER")

	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()
	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "A",
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "B",
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
	}

	err = book.ob.AmendOrder(newOrder, editedOrder)
	if err != types.OrderErrorAmendFailure {
		t.Log(err)
	}

	assert.Equal(t, types.OrderErrorAmendFailure, err)
}

func TestOrderBook_AmendOrderInvalidAmendOutOfSequence(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMENDING OUT OF SEQUENCE ORDER")

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "A",
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "A",
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     5,
	}

	err = book.ob.AmendOrder(newOrder, editedOrder)
	if err != types.OrderErrorOutOfSequence {
		t.Log(err)
	}

	assert.Equal(t, types.OrderErrorOutOfSequence, err)
}

func TestOrderBook_AmendOrderInvalidAmendSize(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN AMEND ORDER INVALID SIZE")

	newOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "A",
		Size:          200,
		Remaining:     200,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, "123456", confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	editedOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            "123456",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "B",
		Size:          300,
		Remaining:     300,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
	}

	err = book.ob.AmendOrder(newOrder, editedOrder)
	if err != types.OrderErrorAmendFailure {
		t.Log(err)
	}

	assert.Equal(t, types.OrderErrorAmendFailure, err)
}

// ProRata mode OFF which is a default config for vega ME.
func TestOrderBook_SubmitOrderProRataModeOff(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// logger := logging.NewTestLogger()
	// defer logger.Sync()
	// logger.Debug("BEGIN PRO-RATA MODE OFF")

	const numberOfTimestamps = 2
	m := make(map[int64][]*types.Order, numberOfTimestamps)

	// sell and buy side orders at timestamp 0
	m[0] = []*types.Order{
		// Side Sell
		{
			ID:            "V0000000032-0000000009",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "A",
			Side:          types.SideSell,
			Price:         num.NewUint(101),
			OriginalPrice: num.NewUint(101),
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            "V0000000032-0000000010",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "B",
			Side:          types.SideSell,
			Price:         num.NewUint(101),
			OriginalPrice: num.NewUint(101),
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		// Side Buy
		{
			ID:            "V0000000032-0000000011",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "C",
			Side:          types.SideBuy,
			Price:         num.NewUint(98),
			OriginalPrice: num.NewUint(98),
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
		{
			ID:            "V0000000032-0000000012",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "D",
			Side:          types.SideBuy,
			Price:         num.NewUint(98),
			OriginalPrice: num.NewUint(98),
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     0,
		},
	}

	// sell and buy orders at timestamp 1
	m[1] = []*types.Order{
		// Side Sell
		{
			ID:            "V0000000032-0000000013",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "E",
			Side:          types.SideSell,
			Price:         num.NewUint(101),
			OriginalPrice: num.NewUint(101),
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     1,
		},
		// Side Buy
		{
			ID:            "V0000000032-0000000014",
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         "F",
			Side:          types.SideBuy,
			Price:         num.NewUint(99),
			OriginalPrice: num.NewUint(99),
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     1,
		},
	}

	timestamps := []int64{0, 1}
	for _, timestamp := range timestamps {
		for _, o := range m[timestamp] {
			trades, getErr := book.ob.GetTrades(o)
			assert.NoError(t, getErr)
			confirmation, err := book.ob.SubmitOrder(o)
			// this should not return any errors
			assert.Equal(t, nil, err)
			// this should not generate any trades
			assert.Equal(t, 0, len(confirmation.Trades))
			assert.Equal(t, len(trades), len(confirmation.Trades))
		}
	}

	scenario := []aggressiveOrderScenario{
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				ID:            "V0000000032-0000000015",
				Status:        types.OrderStatusActive,
				Type:          types.OrderTypeLimit,
				MarketID:      market,
				Party:         "M",
				Side:          types.SideBuy,
				Price:         num.NewUint(101),
				OriginalPrice: num.NewUint(101),
				Size:          100,
				Remaining:     100,
				TimeInForce:   types.OrderTimeInForceGTC,
				CreatedAt:     3,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.TradeTypeDefault,
					MarketID:  market,
					Price:     num.NewUint(101),
					Size:      100,
					Buyer:     "M",
					Seller:    "A",
					Aggressor: types.SideBuy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:        types.OrderStatusActive,
					Type:          types.OrderTypeLimit,
					MarketID:      market,
					Party:         "A",
					Side:          types.SideSell,
					Price:         num.NewUint(101),
					OriginalPrice: num.NewUint(101),
					Size:          100,
					Remaining:     0,
					TimeInForce:   types.OrderTimeInForceGTC,
					CreatedAt:     0,
				},
			},
		},
		{
			// same price level, remaining on the passive
			aggressiveOrder: &types.Order{
				Status:        types.OrderStatusActive,
				Type:          types.OrderTypeLimit,
				MarketID:      market,
				Party:         "N",
				Side:          types.SideBuy,
				Price:         num.NewUint(102),
				OriginalPrice: num.NewUint(102),
				Size:          200,
				Remaining:     200,
				TimeInForce:   types.OrderTimeInForceGTC,
				CreatedAt:     4,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.TradeTypeDefault,
					MarketID:  market,
					Price:     num.NewUint(101),
					Size:      100,
					Buyer:     "N",
					Seller:    "B",
					Aggressor: types.SideBuy,
				},
				{
					Type:      types.TradeTypeDefault,
					MarketID:  market,
					Price:     num.NewUint(101),
					Size:      100,
					Buyer:     "N",
					Seller:    "E",
					Aggressor: types.SideBuy,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:        types.OrderStatusActive,
					Type:          types.OrderTypeLimit,
					MarketID:      market,
					Party:         "B",
					Side:          types.SideSell,
					Price:         num.NewUint(101),
					OriginalPrice: num.NewUint(101),
					Size:          100,
					Remaining:     0,
					TimeInForce:   types.OrderTimeInForceGTC,
					CreatedAt:     0,
				},
				{
					Status:        types.OrderStatusActive,
					Type:          types.OrderTypeLimit,
					MarketID:      market,
					Party:         "E",
					Side:          types.SideSell,
					Price:         num.NewUint(101),
					OriginalPrice: num.NewUint(101),
					Size:          100,
					Remaining:     0,
					TimeInForce:   types.OrderTimeInForceGTC,
					CreatedAt:     1,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				Status:        types.OrderStatusActive,
				Type:          types.OrderTypeLimit,
				MarketID:      market,
				Party:         "O",
				Side:          types.SideSell,
				Price:         num.NewUint(97),
				OriginalPrice: num.NewUint(97),
				Size:          250,
				Remaining:     250,
				TimeInForce:   types.OrderTimeInForceGTC,
				CreatedAt:     5,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.TradeTypeDefault,
					MarketID:  market,
					Price:     num.NewUint(99),
					Size:      100,
					Buyer:     "F",
					Seller:    "O",
					Aggressor: types.SideSell,
				},
				{
					Type:      types.TradeTypeDefault,
					MarketID:  market,
					Price:     num.NewUint(98),
					Size:      100,
					Buyer:     "C",
					Seller:    "O",
					Aggressor: types.SideSell,
				},
				{
					Type:      types.TradeTypeDefault,
					MarketID:  market,
					Price:     num.NewUint(98),
					Size:      50,
					Buyer:     "D",
					Seller:    "O",
					Aggressor: types.SideSell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:        types.OrderStatusActive,
					Type:          types.OrderTypeLimit,
					MarketID:      market,
					Party:         "F",
					Side:          types.SideBuy,
					Price:         num.NewUint(99),
					OriginalPrice: num.NewUint(99),
					Size:          100,
					Remaining:     0,
					TimeInForce:   types.OrderTimeInForceGTC,
					CreatedAt:     1,
				},
				{
					Status:        types.OrderStatusActive,
					Type:          types.OrderTypeLimit,
					MarketID:      market,
					Party:         "C",
					Side:          types.SideBuy,
					Price:         num.NewUint(98),
					OriginalPrice: num.NewUint(98),
					Size:          100,
					Remaining:     0,
					TimeInForce:   types.OrderTimeInForceGTC,
					CreatedAt:     0,
				},
				{
					Status:        types.OrderStatusActive,
					Type:          types.OrderTypeLimit,
					MarketID:      market,
					Party:         "D",
					Side:          types.SideBuy,
					Price:         num.NewUint(98),
					OriginalPrice: num.NewUint(98),
					Size:          100,
					Remaining:     50,
					TimeInForce:   types.OrderTimeInForceGTC,
					CreatedAt:     0,
				},
			},
		},
		{
			// same price level, NO PRORATA
			aggressiveOrder: &types.Order{
				Status:        types.OrderStatusActive,
				Type:          types.OrderTypeLimit,
				MarketID:      market,
				Party:         "X",
				Side:          types.SideSell,
				Price:         num.NewUint(98),
				OriginalPrice: num.NewUint(98),
				Size:          50,
				Remaining:     50,
				TimeInForce:   types.OrderTimeInForceGTC,
				CreatedAt:     6,
			},
			expectedTrades: []types.Trade{
				{
					Type:      types.TradeTypeDefault,
					MarketID:  market,
					Price:     num.NewUint(98),
					Size:      50,
					Buyer:     "D",
					Seller:    "X",
					Aggressor: types.SideSell,
				},
			},
			expectedPassiveOrdersAffected: []types.Order{
				{
					Status:        types.OrderStatusActive,
					Type:          types.OrderTypeLimit,
					MarketID:      market,
					Party:         "D",
					Side:          types.SideBuy,
					Price:         num.NewUint(98),
					OriginalPrice: num.NewUint(98),
					Size:          100,
					Remaining:     0,
					TimeInForce:   types.OrderTimeInForceGTC,
					CreatedAt:     0,
				},
			},
		},
	}

	for i, s := range scenario {
		fmt.Println()
		fmt.Println()
		fmt.Printf("SCENARIO %d / %d ------------------------------------------------------------------", i+1, len(scenario))
		fmt.Println()
		fmt.Println("aggressor: ", s.aggressiveOrder)
		fmt.Println("expectedPassiveOrdersAffected: ", s.expectedPassiveOrdersAffected)
		fmt.Println("expectedTrades: ", s.expectedTrades)
		fmt.Println()

		trades, getErr := book.ob.GetTrades(s.aggressiveOrder)
		assert.NoError(t, getErr)
		confirmationtypes, err := book.ob.SubmitOrder(s.aggressiveOrder)

		// this should not return any errors
		assert.Equal(t, nil, err)

		// this should not generate any trades
		assert.Equal(t, len(s.expectedTrades), len(confirmationtypes.Trades))
		assert.Equal(t, len(confirmationtypes.Trades), len(trades))

		fmt.Println("CONFIRMATION types:")
		fmt.Println("-> Aggressive:", confirmationtypes.Order)
		fmt.Println("-> Trades :", confirmationtypes.Trades)
		fmt.Println("-> PassiveOrdersAffected:", confirmationtypes.PassiveOrdersAffected)
		fmt.Printf("Scenario: %d / %d \n", i+1, len(scenario))

		// trades should match expected trades
		for i, exp := range s.expectedTrades {
			expectTrade(t, &exp, confirmationtypes.Trades[i])
			expectTrade(t, &exp, trades[i])
		}

		// orders affected should match expected values
		for i, exp := range s.expectedPassiveOrdersAffected {
			expectOrder(t, &exp, confirmationtypes.PassiveOrdersAffected[i])
		}
	}
}

// Validate that an IOC order that is not fully filled
// is not added to the order book.ob.
func TestOrderBook_PartialFillIOCOrder(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()
	logger.Debug("BEGIN PARTIAL FILL IOC ORDER")

	orderID := vgcrypto.RandomHash()
	newOrder := &types.Order{
		ID:            orderID,
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "A",
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
	}

	trades, getErr := book.ob.GetTrades(newOrder)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(newOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, orderID, confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	iocOrderID := vgcrypto.RandomHash()
	iocOrder := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            iocOrderID,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "B",
		Size:          20,
		Remaining:     20,
		TimeInForce:   types.OrderTimeInForceIOC,
		CreatedAt:     10,
	}
	trades, getErr = book.ob.GetTrades(iocOrder)
	assert.NoError(t, getErr)
	confirmation, err = book.ob.SubmitOrder(iocOrder)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, iocOrderID, confirmation.Order.ID)
	assert.Equal(t, 1, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	// Check to see if the order still exists (it should not)
	nonorder, err := book.ob.GetOrderByID(iocOrderID)
	assert.Equal(t, matching.ErrOrderDoesNotExist, err)
	assert.Nil(t, nonorder)
}

func makeOrder(t *testing.T, orderbook *tstOB, market string, id string, side types.Side, price uint64, partyid string, size uint64) *types.Order {
	t.Helper()
	order := getOrder(t, market, id, side, price, partyid, size)
	_, err := orderbook.ob.SubmitOrder(order)
	assert.Equal(t, err, nil)
	return order
}

func getOrder(t *testing.T, market string, id string, side types.Side, price uint64, partyid string, size uint64) *types.Order {
	t.Helper()
	order := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            id,
		Side:          side,
		Price:         num.NewUint(price),
		OriginalPrice: num.NewUint(price),
		Party:         partyid,
		Size:          size,
		Remaining:     size,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
	}
	return order
}

/*****************************************************************************/
/*                             GFN/GFA TESTING                               */
/*****************************************************************************/

func TestOrderBook_GFNMarketNoExpiry(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Enter a GFN market order with no expiration time
	buyOrder := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	buyOrder.TimeInForce = types.OrderTimeInForceGFN
	buyOrder.Type = types.OrderTypeMarket
	buyOrder.ExpiresAt = 0
	buyOrderConf, err := book.ob.SubmitOrder(buyOrder)
	assert.NoError(t, err)
	assert.NotNil(t, buyOrderConf)

	// Enter a GFN market order with no expiration time
	sellOrder := getOrder(t, market, "SellOrder01", types.SideSell, 100, "party01", 10)
	sellOrder.TimeInForce = types.OrderTimeInForceGFN
	sellOrder.Type = types.OrderTypeMarket
	sellOrder.ExpiresAt = 0
	sellOrderConf, err := book.ob.SubmitOrder(sellOrder)
	assert.NoError(t, err)
	assert.NotNil(t, sellOrderConf)
}

func TestOrderBook_GFNMarketWithExpiry(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Enter a GFN market order with an expiration time (which is invalid)
	buyOrder := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	buyOrder.TimeInForce = types.OrderTimeInForceGFN
	buyOrder.Type = types.OrderTypeMarket
	buyOrder.ExpiresAt = 100
	buyOrderConf, err := book.ob.SubmitOrder(buyOrder)
	assert.Error(t, err, types.ErrInvalidExpirationDatetime)
	assert.Nil(t, buyOrderConf)

	// Enter a GFN market order with an expiration time (which is invalid)
	sellOrder := getOrder(t, market, "SellOrder01", types.SideSell, 100, "party01", 10)
	sellOrder.TimeInForce = types.OrderTimeInForceGFN
	sellOrder.Type = types.OrderTypeMarket
	sellOrder.ExpiresAt = 100
	sellOrderConf, err := book.ob.SubmitOrder(sellOrder)
	assert.Error(t, err, types.ErrInvalidExpirationDatetime)
	assert.Nil(t, sellOrderConf)
}

func TestOrderBook_GFNLimitInstantMatch(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Normal limit buy order to match against
	buyOrder := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	buyOrderConf, err := book.ob.SubmitOrder(buyOrder)
	assert.NoError(t, err)
	assert.NotNil(t, buyOrderConf)

	// Enter a GFN market order with an expiration time (which is invalid)
	sellOrder := getOrder(t, market, "SellOrder01", types.SideSell, 100, "party02", 10)
	sellOrder.TimeInForce = types.OrderTimeInForceGFN
	sellOrder.Type = types.OrderTypeLimit
	sellOrderConf, err := book.ob.SubmitOrder(sellOrder)
	assert.NoError(t, err)
	assert.NotNil(t, sellOrderConf)
}

// AUCTION TESTING.
func TestOrderBook_AuctionGFNAreRejected(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()
	assert.True(t, book.ob.InAuction())

	// Try to add an order of type GFN which should be rejected
	order := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	order.TimeInForce = types.OrderTimeInForceGFN
	orderConf, err := book.ob.SubmitOrder(order)
	assert.Equal(t, err, types.OrderErrorInvalidTimeInForce)
	assert.Nil(t, orderConf)
}

func TestOrderBook_ContinuousGFAAreRejected(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// We start in continuous mode
	assert.False(t, book.ob.InAuction())

	// Try to add an order of type GFA which should be rejected
	order := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	order.TimeInForce = types.OrderTimeInForceGFA
	orderConf, err := book.ob.SubmitOrder(order)
	assert.Equal(t, err, types.OrderErrorInvalidTimeInForce)
	assert.Nil(t, orderConf)
}

func TestOrderBook_GFNOrdersCancelledInAuction(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// We start in continuous mode
	assert.False(t, book.ob.InAuction())

	// Add a GFN order
	order := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	order.TimeInForce = types.OrderTimeInForceGFN
	orderConf, err := book.ob.SubmitOrder(order)
	assert.NoError(t, err)
	assert.NotNil(t, orderConf)

	// Switch to auction and makes sure the order is cancelled
	orders := book.ob.EnterAuction()
	assert.Equal(t, len(orders), 1)
	assert.Equal(t, book.ob.GetTotalNumberOfOrders(), int64(1))
}

func TestOrderBook_GFAOrdersCancelledInContinuous(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Flip straight to auction mode
	_ = book.ob.EnterAuction()
	assert.True(t, book.ob.InAuction())

	// Add a GFA order
	order := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	order.TimeInForce = types.OrderTimeInForceGFA
	orderConf, err := book.ob.SubmitOrder(order)
	assert.NoError(t, err)
	assert.NotNil(t, orderConf)

	// Switch to continuous mode and makes sure the order is cancelled
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.NoError(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, book.ob.GetTotalNumberOfOrders(), int64(1))
	assert.Equal(t, len(cancels), 1)
}

func TestOrderBook_IndicativePriceAndVolumeState(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// We start in continuous trading mode
	assert.False(t, book.ob.InAuction())
	assert.Equal(t, book.ob.GetTotalNumberOfOrders(), int64(0))

	// Get indicative auction price and volume which should be zero as we are out of auction
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.SideUnspecified)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, price.Uint64(), uint64(0))

	// Switch to auction mode
	book.ob.EnterAuction()
	assert.True(t, book.ob.InAuction())
	assert.Equal(t, book.ob.GetTotalNumberOfOrders(), int64(0))

	// Get indicative auction price and volume
	price, volume, side = book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.SideUnspecified)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, price.Uint64(), uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.False(t, book.ob.InAuction())
	assert.Equal(t, book.ob.GetTotalNumberOfOrders(), int64(0))
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolumeEmpty(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()
	assert.True(t, book.ob.InAuction())

	// No trades!

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.SideUnspecified)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, price.Uint64(), uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.False(t, book.ob.InAuction())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)
}

func TestOrderBook_IndicativePriceAndVolumeOnlyBuySide(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Trades on just one side of the book
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 99, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 98, "party01", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.SideUnspecified)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, price.Uint64(), uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)

	// All of the orders should remain on the book
	assert.Equal(t, book.ob.GetTotalNumberOfOrders(), int64(3))
}

func TestOrderBook_IndicativePriceAndVolumeOnlySellSide(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Trades on just one side of the book
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 100, "party01", 10)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 99, "party01", 10)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 98, "party01", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(0))
	assert.Equal(t, volume, uint64(0))
	assert.Equal(t, side, types.SideUnspecified)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, price.Uint64(), uint64(0))

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)

	// All of the orders should remain on the book
	assert.Equal(t, book.ob.GetTotalNumberOfOrders(), int64(3))
}

func TestOrderBook_IndicativePriceAndVolume1(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 101, "party01", 20)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 98, "party01", 10)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 100, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 101, "party02", 15)
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
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 0)
	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

func TestOrderBook_IndicativePriceAndVolume2(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 101, "party01", 30)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 100, "party01", 10)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 99, "party01", 20)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 98, "party01", 10)
	makeOrder(t, book, market, "BuyOrder05", types.SideBuy, 97, "party01", 5)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 100, "party02", 30)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 101, "party02", 15)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 102, "party02", 5)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 103, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, uint64(100), price.Uint64())
	assert.Equal(t, uint64(30), volume)
	assert.Equal(t, types.SideBuy, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, uint64(100), price.Uint64())

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(uncrossedOrders))
	assert.Equal(t, 0, len(cancels))

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

func TestOrderBook_IndicativePriceAndVolume3(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 104, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 103, "party01", 20)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 102, "party01", 15)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 98, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 97, "party02", 20)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 96, "party02", 15)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, 100, int(price.Uint64()))
	assert.Equal(t, 45, int(volume))
	assert.Equal(t, types.SideBuy, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, 100, int(price.Uint64()))

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 3, len(uncrossedOrders))
	assert.Equal(t, 0, len(cancels))

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

func TestOrderBook_IndicativePriceAndVolume4(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 99, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 98, "party01", 25)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 97, "party01", 5)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 102, "party02", 30)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 101, "party02", 15)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 100, "party02", 5)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, uint64(0), price.Uint64())
	assert.Equal(t, uint64(0), volume)
	assert.Equal(t, types.SideUnspecified, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, uint64(0), price.Uint64())

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 0)
	assert.Equal(t, len(cancels), 0)

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

func TestOrderBook_IndicativePriceAndVolume5(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 100, "party01", 7)
	makeOrder(t, book, market, "BuyOrder05", types.SideBuy, 99, "party01", 6)
	makeOrder(t, book, market, "BuyOrder06", types.SideBuy, 98, "party01", 5)
	makeOrder(t, book, market, "BuyOrder07", types.SideBuy, 97, "party01", 4)
	makeOrder(t, book, market, "BuyOrder08", types.SideBuy, 96, "party01", 3)
	makeOrder(t, book, market, "BuyOrder09", types.SideBuy, 95, "party01", 2)
	makeOrder(t, book, market, "BuyOrder10", types.SideBuy, 94, "party01", 1)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 105, "party02", 1)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 104, "party02", 2)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 103, "party02", 3)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 102, "party02", 4)
	makeOrder(t, book, market, "SellOrder05", types.SideSell, 101, "party02", 5)
	makeOrder(t, book, market, "SellOrder06", types.SideSell, 100, "party02", 6)
	makeOrder(t, book, market, "SellOrder07", types.SideSell, 99, "party02", 7)
	makeOrder(t, book, market, "SellOrder08", types.SideSell, 98, "party02", 8)
	makeOrder(t, book, market, "SellOrder09", types.SideSell, 97, "party02", 9)
	makeOrder(t, book, market, "SellOrder10", types.SideSell, 96, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, uint64(99), price.Uint64())
	assert.Equal(t, uint64(34), volume)
	assert.Equal(t, types.SideBuy, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, uint64(99), price.Uint64())

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 4, len(uncrossedOrders))
	assert.Equal(t, 0, len(cancels))

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

// Set up an auction so that the sell side is processed when we uncross.
func TestOrderBook_IndicativePriceAndVolume6(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 100, "party01", 7)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 99, "party02", 1)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 98, "party02", 2)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 97, "party02", 3)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 96, "party02", 4)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, 101, int(price.Uint64()))
	assert.Equal(t, 10, int(volume))
	assert.Equal(t, types.SideBuy, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, 101, int(price.Uint64()))

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 0)

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

// Check that multiple orders per price level work.
func TestOrderBook_IndicativePriceAndVolume7(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 103, "party01", 1)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 102, "party01", 1)
	makeOrder(t, book, market, "BuyOrder05", types.SideBuy, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder06", types.SideBuy, 101, "party01", 1)
	makeOrder(t, book, market, "BuyOrder07", types.SideBuy, 100, "party01", 7)
	makeOrder(t, book, market, "BuyOrder08", types.SideBuy, 100, "party01", 1)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 99, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 98, "party02", 10)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 97, "party02", 10)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 96, "party02", 7)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, uint64(99), price.Uint64())
	assert.Equal(t, uint64(37), volume)
	assert.Equal(t, types.SideSell, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, uint64(99), price.Uint64())

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 4, len(uncrossedOrders))
	assert.Equal(t, 0, len(cancels))

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

func TestOrderBook_IndicativePriceAndVolume8(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 103, "party01", 1)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 102, "party01", 1)
	makeOrder(t, book, market, "BuyOrder05", types.SideBuy, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder06", types.SideBuy, 101, "party01", 1)
	makeOrder(t, book, market, "BuyOrder07", types.SideBuy, 100, "party01", 7)
	makeOrder(t, book, market, "BuyOrder08", types.SideBuy, 100, "party01", 1)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 99, "party02", 10)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 98, "party02", 10)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 97, "party02", 10)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 96, "party02", 9)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, 99, int(price.Uint64()))
	assert.Equal(t, 38, int(volume))
	assert.Equal(t, types.SideBuy, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, 99, int(price.Uint64()))

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 8, len(uncrossedOrders))
	assert.Equal(t, 0, len(cancels))

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

func TestOrderBook_IndicativePriceAndVolume9(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 110, "party01", 1)
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 110, "party02", 1)

	makeOrder(t, book, market, "BuyOrder01-2", types.SideBuy, 111, "party01", 1)
	makeOrder(t, book, market, "SellOrder01-2", types.SideSell, 111, "party02", 1)

	makeOrder(t, book, market, "BuyOrder01-3", types.SideBuy, 133, "party01", 2)
	makeOrder(t, book, market, "SellOrder01-3", types.SideSell, 133, "party02", 2)

	makeOrder(t, book, market, "BuyOrder01-4", types.SideBuy, 303, "party01", 10)
	makeOrder(t, book, market, "SellOrder01-4", types.SideSell, 303, "party02", 10)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, 303, int(price.Uint64()))
	assert.Equal(t, 10, int(volume))
	assert.Equal(t, types.SideBuy, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, 303, int(price.Uint64()))

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(uncrossedOrders))
	assert.Equal(t, 0, len(cancels))

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

// check behaviour consistent in the presence of wash trades.
func TestOrderBook_IndicativePriceAndVolume10(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	// Populate buy side
	makeOrder(t, book, market, "BuyOrder01", types.SideBuy, 103, "party01", 10)
	makeOrder(t, book, market, "BuyOrder02", types.SideBuy, 102, "party01", 9)
	makeOrder(t, book, market, "BuyOrder03", types.SideBuy, 101, "party01", 8)
	makeOrder(t, book, market, "BuyOrder04", types.SideBuy, 100, "party01", 7)

	// Populate sell side
	makeOrder(t, book, market, "SellOrder01", types.SideSell, 99, "party02", 1)
	makeOrder(t, book, market, "SellOrder02", types.SideSell, 98, "party01", 1)
	makeOrder(t, book, market, "SellOrder03", types.SideSell, 98, "party02", 1)
	makeOrder(t, book, market, "SellOrder04", types.SideSell, 97, "party02", 3)
	makeOrder(t, book, market, "SellOrder05", types.SideSell, 96, "party02", 4)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, 101, int(price.Uint64()))
	assert.Equal(t, 10, int(volume))
	assert.Equal(t, types.SideBuy, side)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, 101, int(price.Uint64()))

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 0)

	nTrades := 0
	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			nTrades++
			assert.Equal(t, price, x.Price)
		}
	}
	assert.Equal(t, len(trades), nTrades)
}

func TestOrderBook_UncrossTest1(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	bo1 := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 5)
	bo1.TimeInForce = types.OrderTimeInForceGFA
	book.ob.SubmitOrder(bo1)

	so1 := getOrder(t, market, "SellOrder01", types.SideSell, 100, "party02", 5)
	so1.TimeInForce = types.OrderTimeInForceGFA
	book.ob.SubmitOrder(so1)

	bo2 := getOrder(t, market, "BuyOrder02", types.SideBuy, 100, "party01", 5)
	bo2.TimeInForce = types.OrderTimeInForceGFA
	book.ob.SubmitOrder(bo2)

	so2 := getOrder(t, market, "SellOrder02", types.SideSell, 101, "party02", 5)
	so2.TimeInForce = types.OrderTimeInForceGFA
	book.ob.SubmitOrder(so2)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(100))
	assert.Equal(t, volume, uint64(5))
	assert.Equal(t, side, types.SideSell)
	price = book.ob.GetIndicativePrice()
	assert.Equal(t, price.Uint64(), uint64(100))

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	for _, x := range trades {
		assert.Equal(t, price, x.Price)
	}

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(cancels), 2)

	for _, o := range uncrossedOrders {
		for _, x := range o.Trades {
			assert.Equal(t, price, x.Price)
		}
	}
}

// this is a test for issue 2060 to ensure we process FOK orders properly.
func TestOrderBook_NetworkOrderSuccess(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orders := []*types.Order{
		{
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			ID:            "123456",
			Side:          types.SideBuy,
			Price:         num.NewUint(100),
			OriginalPrice: num.NewUint(100),
			Party:         "A",
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     10,
		},
		{
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			ID:            "234561",
			Side:          types.SideBuy,
			Price:         num.NewUint(1),
			OriginalPrice: num.NewUint(1),
			Party:         "B",
			Size:          100,
			Remaining:     100,
			TimeInForce:   types.OrderTimeInForceGTC,
			CreatedAt:     11,
		},
	}

	// now we add the trades to the book
	for _, o := range orders {
		cnfm, err := book.ob.SubmitOrder(o)
		assert.NoError(t, err)
		assert.Len(t, cnfm.Trades, 0)
	}

	// no price for network order
	// we want to consume the whole book
	netorder := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeNetwork,
		MarketID:    market,
		ID:          "345612",
		Side:        types.SideSell,
		Party:       "C",
		Size:        200,
		Remaining:   200,
		TimeInForce: types.OrderTimeInForceFOK,
		CreatedAt:   12,
	}

	cnfm, err := book.ob.SubmitOrder(netorder)
	assert.NoError(t, err)
	assert.Equal(t, types.OrderStatusFilled, netorder.Status)
	assert.Equal(t, 50, int(netorder.Price.Uint64()))
	assert.Equal(t, 0, int(netorder.Remaining))
	_ = cnfm
}

func TestOrderBook_GetTradesInLineWithSubmitOrderDuringAuction(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)

	orders := book.ob.EnterAuction()
	assert.Equal(t, 0, len(orders))
	order1Id := vgcrypto.RandomHash()
	order2Id := vgcrypto.RandomHash()

	order1 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            order1Id,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "A",
		Size:          100,
		Remaining:     100,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
	}

	trades, getErr := book.ob.GetTrades(order1)
	assert.NoError(t, getErr)
	confirmation, err := book.ob.SubmitOrder(order1)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, order1Id, confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	order2 := &types.Order{
		Status:        types.OrderStatusActive,
		Type:          types.OrderTypeLimit,
		MarketID:      market,
		ID:            order2Id,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Party:         "B",
		Size:          20,
		Remaining:     20,
		TimeInForce:   types.OrderTimeInForceGTC,
		CreatedAt:     10,
	}
	trades, getErr = book.ob.GetTrades(order2)
	assert.NoError(t, getErr)
	confirmation, err = book.ob.SubmitOrder(order2)

	assert.Equal(t, nil, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, order2Id, confirmation.Order.ID)
	assert.Equal(t, 0, len(confirmation.Trades))
	assert.Equal(t, len(trades), len(confirmation.Trades))

	// Confirm both orders still on the book
	order, err := book.ob.GetOrderByID(order1Id)
	assert.NotNil(t, order)
	assert.Nil(t, err)
	order, err = book.ob.GetOrderByID(order2Id)
	assert.NotNil(t, order)
	assert.Nil(t, err)
}

func TestOrderBook_AuctionUncrossWashTrades(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	bo1 := getOrder(t, market, "BuyOrder01", types.SideBuy, 100, "party01", 5)
	bo1.TimeInForce = types.OrderTimeInForceGFA
	book.ob.SubmitOrder(bo1)

	so1 := getOrder(t, market, "SellOrder01", types.SideSell, 100, "party01", 5)
	so1.TimeInForce = types.OrderTimeInForceGFA
	book.ob.SubmitOrder(so1)

	// Get indicative auction price and volume
	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	assert.Equal(t, price.Uint64(), uint64(100))
	assert.Equal(t, volume, uint64(5))
	assert.Equal(t, side, types.SideBuy)

	// Get indicative trades
	trades, err := book.ob.GetIndicativeTrades()
	assert.NoError(t, err)
	assert.Equal(t, len(trades), 1)

	// Leave auction and uncross the book
	uncrossedOrders, cancels, err := book.ob.LeaveAuction(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(uncrossedOrders), 1)
	assert.Equal(t, len(uncrossedOrders[0].Trades), 1)
	assert.Equal(t, len(cancels), 0)

	// Assure indicative trade has same (relevant) data as the actual trade
	// because the trades are generated when calling LeaveAuction, the aggressor will be unspecified.
	assert.Equal(t, uncrossedOrders[0].Trades[0].Aggressor, types.SideUnspecified)
	// and thus the aggressor side will not match the value we get from the indicative trades
	assert.NotEqual(t, uncrossedOrders[0].Trades[0].Aggressor, trades[0].Aggressor)
	assert.Equal(t, uncrossedOrders[0].Trades[0].Buyer, trades[0].Buyer)
	assert.Equal(t, uncrossedOrders[0].Trades[0].Seller, trades[0].Seller)
	assert.Equal(t, uncrossedOrders[0].Trades[0].Size, trades[0].Size)
	assert.Equal(t, uncrossedOrders[0].Trades[0].Price, trades[0].Price)

	// Assure trade is indeed a wash trade
	assert.Equal(t, trades[0].Buyer, trades[0].Seller)
}

func TestOrderBook_AuctionUncrossTamlyn(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	order1 := getOrder(t, market, "Order1", types.SideBuy, 16, "Tamlyn", 100)
	order1.TimeInForce = types.OrderTimeInForceGFA
	conf, err := book.ob.SubmitOrder(order1)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order2 := getOrder(t, market, "Order2", types.SideSell, 20, "Tamlyn", 100)
	order2.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order2)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order3 := getOrder(t, market, "Order3", types.SideSell, 3, "Tamlyn", 100)
	order3.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order3)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order4 := getOrder(t, market, "Order4", types.SideSell, 18, "David", 100)
	order4.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order4)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order5 := getOrder(t, market, "Order5", types.SideBuy, 1000, "Tamlyn", 100)
	order5.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order5)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order6 := getOrder(t, market, "Order6", types.SideBuy, 2000, "David", 100)
	order6.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order6)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order7 := getOrder(t, market, "Order7", types.SideSell, 14, "Tamlyn", 15)
	order7.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order7)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order8 := getOrder(t, market, "Order8", types.SideBuy, 14, "Tamlyn", 2)
	order8.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order8)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	order9 := getOrder(t, market, "Order9", types.SideSell, 1, "David", 10)
	order9.TimeInForce = types.OrderTimeInForceGFA
	conf, err = book.ob.SubmitOrder(order9)
	require.NoError(t, err)
	assert.NotNil(t, conf)

	// Get indicative auction price and volume
	//	price, volume, side := book.ob.GetIndicativePriceAndVolume()
	//	assert.Equal(t, price, uint64(100))
	//	assert.Equal(t, volume, uint64(5))
	//	assert.Equal(t, side, types.Side_SIDE_BUY)

	// Leave auction and uncross the book
	//	uncrossedOrders, cancels, err := book.ob.leaveAuction(time.Now())
	//	assert.Nil(t, err)
	//	assert.Equal(t, len(uncrossedOrders), 1)
	//	assert.Equal(t, len(cancels), 0)
}

// Add some pegged orders to the order book and check they are parked when going into auction.
func TestOrderBook_PeggedOrders(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	require.Equal(t, uint64(0), book.ob.GetPeggedOrdersCount())

	// We need some orders on the book to get a valid bestbis/bestask/mid price
	makeOrder(t, book, market, "PriceSetterBuy", types.SideBuy, 100, "party01", 1)
	makeOrder(t, book, market, "PriceSetterSell", types.SideSell, 101, "party01", 1)

	require.Equal(t, uint64(0), book.ob.GetPeggedOrdersCount())

	bestask, err := book.ob.GetBestAskPrice()
	assert.NoError(t, err)
	bestbid, err := book.ob.GetBestBidPrice()
	assert.NoError(t, err)
	assert.Equal(t, bestask.Uint64(), uint64(101))
	assert.Equal(t, bestbid.Uint64(), uint64(100))

	orderID := crypto.RandomHash()
	bp1 := getOrder(t, market, orderID, types.SideBuy, 100, "party01", 5)
	bp1.PeggedOrder = &types.PeggedOrder{
		Reference: types.PeggedReferenceMid,
		Offset:    num.NewUint(3),
	}
	book.ob.SubmitOrder(bp1)

	require.Equal(t, uint64(1), book.ob.GetPeggedOrdersCount())

	sp1 := getOrder(t, market, "SellPeg1", types.SideSell, 100, "party01", 5)
	sp1.PeggedOrder = &types.PeggedOrder{
		Reference: types.PeggedReferenceMid,
		Offset:    num.NewUint(3),
	}
	book.ob.SubmitOrder(sp1)

	// wash trade, doesn't go through so still expect 1
	require.Equal(t, uint64(1), book.ob.GetPeggedOrdersCount())

	// Leave auction and uncross the book
	cancels := book.ob.EnterAuction()
	assert.Equal(t, len(cancels), 0)

	book.ob.CancelOrder(bp1)
	require.Equal(t, uint64(0), book.ob.GetPeggedOrdersCount())
}

func TestOrderBook_BidAndAskPresentAfterAuction(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()
	assert.True(t, book.ob.InAuction())

	require.Equal(t, false, book.ob.BidAndAskPresentAfterAuction())
	require.Equal(t, false, book.ob.CanUncross())

	matchingPrice := uint64(100)
	party1 := "party1"
	party2 := "party2"
	makeOrder(t, book, market, vgcrypto.RandomHash(), types.SideBuy, matchingPrice-1, party1, 1)

	require.Equal(t, false, book.ob.BidAndAskPresentAfterAuction())
	require.Equal(t, false, book.ob.CanUncross())

	makeOrder(t, book, market, vgcrypto.RandomHash(), types.SideSell, matchingPrice+1, party2, 1)

	require.Equal(t, true, book.ob.BidAndAskPresentAfterAuction())
	require.Equal(t, false, book.ob.CanUncross())

	makeOrder(t, book, market, vgcrypto.RandomHash(), types.SideBuy, matchingPrice, party1, 1)

	require.Equal(t, true, book.ob.BidAndAskPresentAfterAuction())
	require.Equal(t, false, book.ob.CanUncross())

	makeOrder(t, book, market, vgcrypto.RandomHash(), types.SideSell, matchingPrice, party2, 1)

	require.Equal(t, true, book.ob.BidAndAskPresentAfterAuction())
	require.Equal(t, true, book.ob.CanUncross())

	_, err := book.ob.CancelAllOrders(party1)
	require.NoError(t, err)
	_, err = book.ob.CancelAllOrders(party2)
	require.NoError(t, err)

	require.Equal(t, int64(0), book.ob.GetTotalNumberOfOrders())

	makeOrder(t, book, market, vgcrypto.RandomHash(), types.SideBuy, matchingPrice, party1, 1)

	require.Equal(t, false, book.ob.BidAndAskPresentAfterAuction())
	require.Equal(t, false, book.ob.CanUncross())

	makeOrder(t, book, market, vgcrypto.RandomHash(), types.SideSell, matchingPrice, party2, 1)

	require.Equal(t, false, book.ob.BidAndAskPresentAfterAuction())
	require.Equal(t, false, book.ob.CanUncross())
}

func TestOrderBook_AuctionUncrossWashTrades2(t *testing.T) {
	market := vgrand.RandomStr(5)
	book := getTestOrderBook(t, market)
	defer book.Finish()

	logger := logging.NewTestLogger()
	defer logger.Sync()

	// Switch to auction mode
	book.ob.EnterAuction()

	tt0_0 := getOrder(t, market, "tt_0_0", types.SideBuy, 90, "tt_0", 1000)
	_, err := book.ob.SubmitOrder(tt0_0)
	require.NoError(t, err)
	tt0_1 := getOrder(t, market, "tt_0_1", types.SideSell, 200, "tt_0", 1000)
	_, err = book.ob.SubmitOrder(tt0_1)
	require.NoError(t, err)

	tt1_0 := getOrder(t, market, "tt_1_0", types.SideSell, 110, "tt_1", 50)
	_, err = book.ob.SubmitOrder(tt1_0)
	require.NoError(t, err)
	tt2_0 := getOrder(t, market, "tt_2_0", types.SideBuy, 110, "tt_2", 20)
	_, err = book.ob.SubmitOrder(tt2_0)
	require.NoError(t, err)
	tt3_0 := getOrder(t, market, "tt_3_0", types.SideBuy, 110, "tt_3", 30)
	_, err = book.ob.SubmitOrder(tt3_0)
	require.NoError(t, err)

	indicativeTrades, err := book.ob.GetIndicativeTrades()
	require.NoError(t, err)
	require.Equal(t, 2, len(indicativeTrades))

	require.Equal(t, tt1_0.Party, indicativeTrades[0].Seller)
	require.Equal(t, tt2_0.Party, indicativeTrades[0].Buyer)
	require.Equal(t, tt2_0.Size, indicativeTrades[0].Size)
	require.Equal(t, tt2_0.Price, indicativeTrades[0].Price)

	require.Equal(t, tt1_0.Party, indicativeTrades[1].Seller)
	require.Equal(t, tt3_0.Party, indicativeTrades[1].Buyer)
	require.Equal(t, tt3_0.Size, indicativeTrades[1].Size)
	require.Equal(t, tt3_0.Price, indicativeTrades[1].Price)

	// Add wash trades
	tt4_0 := getOrder(t, market, "tt_4_0", types.SideSell, 110, "tt_4", 40)
	_, err = book.ob.SubmitOrder(tt4_0)
	require.NoError(t, err)
	tt4_1 := getOrder(t, market, "tt_4_1", types.SideBuy, 110, "tt_4", 40)
	_, err = book.ob.SubmitOrder(tt4_1)
	require.NoError(t, err)

	indicativeTrades, err = book.ob.GetIndicativeTrades()
	require.NoError(t, err)
	// Expecting one more indicative trade now
	require.Equal(t, 3, len(indicativeTrades))

	// The first two should stay as they were
	require.Equal(t, tt1_0.Party, indicativeTrades[0].Seller)
	require.Equal(t, tt2_0.Party, indicativeTrades[0].Buyer)
	require.Equal(t, tt2_0.Size, indicativeTrades[0].Size)
	require.Equal(t, tt2_0.Price, indicativeTrades[0].Price)

	require.Equal(t, tt1_0.Party, indicativeTrades[1].Seller)
	require.Equal(t, tt3_0.Party, indicativeTrades[1].Buyer)
	require.Equal(t, tt3_0.Size, indicativeTrades[1].Size)
	require.Equal(t, tt3_0.Price, indicativeTrades[1].Price)

	// The third one should be the wash trade
	require.Equal(t, tt4_0.Party, indicativeTrades[2].Seller)
	require.Equal(t, tt4_1.Party, indicativeTrades[2].Buyer)
	require.Equal(t, tt4_0.Size, indicativeTrades[2].Size)
	require.Equal(t, tt4_0.Price, indicativeTrades[2].Price)

	confs, ordersToCancel, err := book.ob.LeaveAuction(time.Now())
	require.NoError(t, err)
	require.Equal(t, 3, len(confs))
	require.Equal(t, 0, len(ordersToCancel))

	for i, c := range confs {
		require.Equal(t, 1, len(c.Trades))
		require.Equal(t, c.Trades[0].Buyer, indicativeTrades[i].Buyer)
		require.Equal(t, c.Trades[0].Seller, indicativeTrades[i].Seller)
		require.Equal(t, c.Trades[0].Size, indicativeTrades[i].Size)
		require.Equal(t, c.Trades[0].Price, indicativeTrades[i].Price)
	}
}

// just generates random orders with the given prices. Uses parties provided by accessing
// parties[i%len(parties)], where i is the current index in the buy/sell prices slice.
// if parties is empty, []string{"A", "B"} is used.
func getTestOrders(t *testing.T, market string, fixedSize uint64, buyPrices, sellPrices []uint64) []*types.Order {
	t.Helper()
	parties := []string{"A", "B"}
	orders := make([]*types.Order, 0, len(buyPrices)+len(sellPrices))
	for i, p := range buyPrices {
		size := fixedSize
		if size == 0 {
			size = uint64(rand.Int63n(10-1) + 1)
		}
		orders = append(orders, &types.Order{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         parties[i%len(parties)],
			Side:          types.SideBuy,
			Price:         num.NewUint(p),
			OriginalPrice: num.NewUint(p),
			Size:          size,
			Remaining:     size,
			TimeInForce:   types.OrderTimeInForceGTC,
		})
	}
	for i, p := range sellPrices {
		size := fixedSize
		if size == 0 {
			size = uint64(rand.Int63n(10-1) + 1)
		}
		orders = append(orders, &types.Order{
			ID:            vgcrypto.RandomHash(),
			Status:        types.OrderStatusActive,
			Type:          types.OrderTypeLimit,
			MarketID:      market,
			Party:         parties[i%len(parties)],
			Side:          types.SideSell,
			Price:         num.NewUint(p),
			OriginalPrice: num.NewUint(p),
			Size:          size,
			Remaining:     size,
			TimeInForce:   types.OrderTimeInForceGTC,
		})
	}
	return orders
}

func TestVwapEmptySide(t *testing.T) {
	ob := getTestOrderBook(t, "market1")
	_, err := ob.ob.VWAP(0, types.SideBuy)
	require.Error(t, err)
	_, err = ob.ob.VWAP(0, types.SideSell)
	require.Error(t, err)

	_, err = ob.ob.VWAP(10, types.SideBuy)
	require.Error(t, err)
	_, err = ob.ob.VWAP(10, types.SideSell)
	require.Error(t, err)
}

func TestVwapZeroVolume(t *testing.T) {
	ob := getTestOrderBook(t, "market1")

	buyPrices := []uint64{
		90,
	}
	sellPrices := []uint64{
		100,
	}

	orders := getTestOrders(t, "market1", 10, buyPrices, sellPrices)
	for _, o := range orders {
		_, err := ob.ob.SubmitOrder(o)
		assert.NoError(t, err)
	}

	// when the volume passed is 0
	vwap, err := ob.ob.VWAP(0, types.SideBuy)
	require.NoError(t, err)
	require.Equal(t, "90", vwap.String())
	vwap, err = ob.ob.VWAP(0, types.SideSell)
	require.NoError(t, err)
	require.Equal(t, "100", vwap.String())
}

func TestVwapNotEnoughVolume(t *testing.T) {
	ob := getTestOrderBook(t, "market1")

	buyPrices := []uint64{
		90,
		95,
		100,
	}
	sellPrices := []uint64{
		200,
		210,
		220,
	}

	orders := getTestOrders(t, "market1", 10, buyPrices, sellPrices)
	for _, o := range orders {
		_, err := ob.ob.SubmitOrder(o)
		assert.NoError(t, err)
	}

	// there's 30 in the order book
	_, err := ob.ob.VWAP(40, types.SideBuy)
	require.Error(t, err)
	_, err = ob.ob.VWAP(40, types.SideSell)
	require.Error(t, err)
}

func TestVWAP(t *testing.T) {
	ob := getTestOrderBook(t, "market1")

	buyPrices := []uint64{
		60,
		70,
		100,
	}
	sellPrices := []uint64{
		200,
		210,
		220,
	}

	orders := getTestOrders(t, "market1", 10, buyPrices, sellPrices)
	for _, o := range orders {
		_, err := ob.ob.SubmitOrder(o)
		assert.NoError(t, err)
	}

	// Bid side
	// =========
	vwap, err := ob.ob.VWAP(5, types.SideBuy)
	require.NoError(t, err)
	require.Equal(t, "100", vwap.String())

	vwap, err = ob.ob.VWAP(10, types.SideBuy)
	require.NoError(t, err)
	require.Equal(t, "100", vwap.String())

	// (100 * 10 + 70 * 5)/15
	vwap, err = ob.ob.VWAP(15, types.SideBuy)
	require.NoError(t, err)
	require.Equal(t, "90", vwap.String())

	// (100 + 70)/2
	vwap, err = ob.ob.VWAP(20, types.SideBuy)
	require.NoError(t, err)
	require.Equal(t, "85", vwap.String())

	// (100 * 10 + 70 * 10 + 60 * 5)/25
	vwap, err = ob.ob.VWAP(25, types.SideBuy)
	require.NoError(t, err)
	require.Equal(t, "80", vwap.String())

	// (100 + 70 + 60)/3
	vwap, err = ob.ob.VWAP(30, types.SideBuy)
	require.NoError(t, err)
	require.Equal(t, "76", vwap.String())

	// Ask side
	// =========
	vwap, err = ob.ob.VWAP(5, types.SideSell)
	require.NoError(t, err)
	require.Equal(t, "200", vwap.String())

	vwap, err = ob.ob.VWAP(10, types.SideSell)
	require.NoError(t, err)
	require.Equal(t, "200", vwap.String())

	// (200 * 10 + 210 * 5)/15
	vwap, err = ob.ob.VWAP(15, types.SideSell)
	require.NoError(t, err)
	require.Equal(t, "203", vwap.String())

	// (200 + 210)/2
	vwap, err = ob.ob.VWAP(20, types.SideSell)
	require.NoError(t, err)
	require.Equal(t, "205", vwap.String())

	// (200 * 10 + 210 * 10 + 220 * 5)/25
	vwap, err = ob.ob.VWAP(25, types.SideSell)
	require.NoError(t, err)
	require.Equal(t, "208", vwap.String())

	// (200 + 210 + 220)/3
	vwap, err = ob.ob.VWAP(30, types.SideSell)
	require.NoError(t, err)
	require.Equal(t, "210", vwap.String())
}
