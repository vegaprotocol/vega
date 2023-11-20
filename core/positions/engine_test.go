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

package positions_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePosition(t *testing.T) {
	t.Run("Update position regular", testUpdatePositionRegular)
}

func TestGetOpenInterest(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	var (
		buyer         = "buyer_id"
		buyer2        = "buyer_id2"
		seller        = "seller_id"
		size   uint64 = 10
		price         = num.NewUint(10000)
	)
	passive1 := &types.Order{
		Party:     buyer,
		Remaining: size,
		Price:     price,
		Side:      types.SideBuy,
	}
	passive2 := &types.Order{
		Party:     buyer2,
		Remaining: size,
		Price:     price,
		Side:      types.SideBuy,
	}
	aggressive := &types.Order{
		Party:     seller,
		Remaining: size * 2,
		Price:     price,
		Side:      types.SideSell,
	}
	engine.RegisterOrder(context.TODO(), passive1)
	engine.RegisterOrder(context.TODO(), passive2)
	engine.RegisterOrder(context.TODO(), aggressive)

	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     num.NewUint(10000),
		Size:      size,
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	_ = engine.Update(context.Background(), &trade, passive1, aggressive)
	trade = types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     num.NewUint(10000),
		Size:      size,
		Buyer:     buyer2,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	_ = engine.Update(context.Background(), &trade, passive2, aggressive)
	// 3 positions
	// 2 at + 10
	// 1 at -20
	// we should get an open interest of 20
	openInterest := engine.GetOpenInterest()
	assert.Equal(t, 20, int(openInterest))
}

func testUpdatePositionRegular(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	var (
		buyer         = "buyer_id"
		seller        = "seller_id"
		size   uint64 = 10
		price         = num.NewUint(10000)
	)
	passive := &types.Order{
		Party:     buyer,
		Remaining: size,
		Price:     price,
		Side:      types.SideBuy,
	}
	aggressive := &types.Order{
		Party:     seller,
		Remaining: size,
		Price:     price,
		Side:      types.SideSell,
	}
	engine.RegisterOrder(context.TODO(), passive)
	engine.RegisterOrder(context.TODO(), aggressive)

	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     price,
		Size:      size,
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	positions := engine.Update(context.Background(), &trade, passive, aggressive)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))
	for _, p := range pos {
		if p.Party() == buyer {
			assert.Equal(t, int64(size), p.Size())
			assert.Equal(t, num.UintZero(), p.VWBuy())
		} else {
			assert.Equal(t, int64(-size), p.Size())
			assert.Equal(t, num.UintZero(), p.VWSell())
		}
	}
}

func TestRemoveDistressedEmpty(t *testing.T) {
	data := []events.MarketPosition{
		mp{
			party: "test",
			size:  1,
			price: num.NewUint(1000),
		},
	}
	e := getTestEngine(t)
	ret := e.RemoveDistressed(data)
	assert.Empty(t, ret)
}

func TestRegisterUnregisterOrder(t *testing.T) {
	t.Run("Test successful order register", testRegisterOrderSuccessful)
	t.Run("Test successful order unregister", testUnregisterOrderSuccessful)
	t.Run("Test unsuccessful order unregister", testUnregisterOrderUnsuccessful)
}

func testRegisterOrderSuccessful(t *testing.T) {
	const (
		buysize  int64 = 123
		sellsize int64 = 456
	)
	e := getTestEngine(t)
	orderBuy := types.Order{
		Party:     "test_party",
		Side:      types.SideBuy,
		Size:      uint64(buysize),
		Remaining: uint64(buysize),
		Price:     num.UintZero(),
	}
	pos := e.RegisterOrder(context.TODO(), &orderBuy)
	assert.Equal(t, buysize, pos.Buy())
	assert.Zero(t, pos.Sell())
	assert.True(t, pos.Price().IsZero())
	assert.Zero(t, pos.Size())
	positions := e.Positions()
	assert.Equal(t, 1, len(positions))
	assert.Equal(t, pos.Buy(), positions[0].Buy())

	orderSell := types.Order{
		Party:     "test_party",
		Side:      types.SideSell,
		Size:      uint64(sellsize),
		Remaining: uint64(sellsize),
		Price:     num.UintZero(),
	}
	pos = e.RegisterOrder(context.TODO(), &orderSell)
	assert.Equal(t, buysize, pos.Buy())
	assert.Equal(t, sellsize, pos.Sell())
	assert.True(t, pos.Price().IsZero())
	assert.Zero(t, pos.Size())
	positions = e.Positions()
	assert.Equal(t, 1, len(positions))
	assert.Equal(t, pos.Buy(), positions[0].Buy())
	assert.Equal(t, pos.Sell(), positions[0].Sell())
}

func testUnregisterOrderSuccessful(t *testing.T) {
	const (
		buysize  int64 = 123
		sellsize int64 = 456
	)
	e := getTestEngine(t)
	orderBuy := types.Order{
		Party:     "test_party",
		Side:      types.SideBuy,
		Size:      uint64(buysize),
		Remaining: uint64(buysize),
		Price:     num.UintZero(),
	}
	pos := e.RegisterOrder(context.TODO(), &orderBuy)
	assert.Equal(t, buysize, pos.Buy())

	pos = e.UnregisterOrder(context.TODO(), &orderBuy)
	assert.Zero(t, pos.Buy())

	orderSell := types.Order{
		Party:     "test_party",
		Side:      types.SideSell,
		Size:      uint64(sellsize),
		Remaining: uint64(sellsize),
		Price:     num.UintZero(),
	}
	pos = e.RegisterOrder(context.TODO(), &orderSell)
	assert.Zero(t, pos.Buy())
	assert.Equal(t, sellsize, pos.Sell())

	pos = e.UnregisterOrder(context.TODO(), &orderSell)
	assert.Zero(t, pos.Buy())
	assert.Zero(t, pos.Sell())
}

func testUnregisterOrderUnsuccessful(t *testing.T) {
	e := getTestEngine(t)
	orderBuy := types.Order{
		Party:     "test_party",
		Side:      types.SideBuy,
		Size:      uint64(999),
		Remaining: uint64(999),
		Price:     num.UintZero(),
	}
	require.Panics(t, func() {
		_ = e.UnregisterOrder(context.TODO(), &orderBuy)
	})
}

func getTestEngine(t *testing.T) *positions.SnapshotEngine {
	t.Helper()
	broker := stubs.NewBrokerStub()

	return positions.NewSnapshotEngine(
		logging.NewTestLogger(), positions.NewDefaultConfig(),
		"test_market",
		broker,
	)
}

func TestGetOpenInterestGivenTrades(t *testing.T) {
	// A, B represents partys who already have positions
	// C, D represents partys who don't have positions (but there are entries in "trades" array that contain their trades)

	cases := []struct {
		ExistingPositions []*types.Trade
		Trades            []*types.Trade
		ExpectedOI        uint64
	}{
		// Both parties already have positions
		{ // A: + 100, B: -100 => OI: 100
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			ExpectedOI: 100,
		},
		{ // A: + 100 - 10, B: -100 + 10=> OI: 90
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 10, Price: num.UintZero()},
			},
			ExpectedOI: 90,
		},
		{
			// A: + 100 + 10, B: -100 - 10 => OI: 110
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 10, Price: num.UintZero()},
			},
			ExpectedOI: 110,
		},
		{
			// Same as above + wash trade -> should leave OI unchanged
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 10, Price: num.UintZero()},
				{Seller: "A", Buyer: "A", Size: 13, Price: num.UintZero()},
			},
			ExpectedOI: 110,
		},
		{
			// Same as above + wash trade -> should leave OI unchanged
			ExistingPositions: []*types.Trade{},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 20, Price: num.UintZero()},
				{Seller: "A", Buyer: "C", Size: 30, Price: num.UintZero()},
				{Seller: "D", Buyer: "D", Size: 40, Price: num.UintZero()},
			},
			ExpectedOI: 50,
		},
		// There at least 1 new party
		{
			// A: + 100 + 10, B: -100, C: -10 => OI: 110
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.UintZero()},
			},
			ExpectedOI: 110,
		},
		{
			// A: + 100 - 10, B: -100, C: +10 => OI: 100
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "C", Size: 10, Price: num.UintZero()},
			},
			ExpectedOI: 100,
		},
		// None of the parties have positions yet
		{
			// C: +10, D:-10 => OI: 10
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 10, Price: num.UintZero()},
			},
			ExpectedOI: 10,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 5, Price: num.UintZero()},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 5, Price: num.UintZero()},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "C", Buyer: "B", Size: 5, Price: num.UintZero()},
			},
			ExpectedOI: 205,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "C", Size: 5, Price: num.UintZero()},
			},
			ExpectedOI: 195,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 500, Price: num.UintZero()},
			},
			ExpectedOI: 500,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 5, Price: num.UintZero()},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 5, Price: num.UintZero()},
			},
			ExpectedOI: 110,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 5, Price: num.UintZero()},
			},
			ExpectedOI: 110,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 200, Price: num.UintZero()},
			},
			ExpectedOI: 300,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "A", Buyer: "C", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 300, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			ExpectedOI: 300,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "A", Buyer: "C", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 300, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			ExpectedOI: 400,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "A", Buyer: "C", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 300, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "A", Buyer: "B", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			ExpectedOI: 400,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "A", Buyer: "C", Size: 100, Price: num.UintZero()},
				{Seller: "C", Buyer: "B", Size: 300, Price: num.UintZero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "A", Buyer: "C", Size: 100, Price: num.UintZero()},
				{Seller: "D", Buyer: "A", Size: 100, Price: num.UintZero()},
				{Seller: "B", Buyer: "A", Size: 100, Price: num.UintZero()},
			},
			ExpectedOI: 400,
		},
	}

	for _, tc := range cases {
		e := getTestEngine(t)

		for _, tr := range tc.ExistingPositions {
			passive := registerOrder(e, types.SideBuy, tr.Buyer, tr.Price, tr.Size)
			aggressive := registerOrder(e, types.SideSell, tr.Seller, tr.Price, tr.Size)
			e.Update(context.Background(), tr, passive, aggressive)
		}

		oiGivenTrades := e.GetOpenInterestGivenTrades(tc.Trades)

		for _, tr := range tc.Trades {
			passive := registerOrder(e, types.SideBuy, tr.Buyer, tr.Price, tr.Size)
			aggressive := registerOrder(e, types.SideSell, tr.Seller, tr.Price, tr.Size)
			e.Update(context.Background(), tr, passive, aggressive)
		}

		// Now check it matches once those trades are registered as positions
		oiAfterUpdatingPositions := e.GetOpenInterest()
		t.Run("", func(t *testing.T) {
			require.Equal(t, tc.ExpectedOI, oiGivenTrades)
			require.Equal(t, tc.ExpectedOI, oiAfterUpdatingPositions)
		})
	}
}

type mp struct {
	size, buy, sell int64
	party           string
	price           *num.Uint
}

func (m mp) AverageEntryPrice() *num.Uint {
	return num.UintZero()
}

func (m mp) Party() string {
	return m.party
}

func (m mp) Size() int64 {
	return m.size
}

func (m mp) Buy() int64 {
	return m.buy
}

func (m mp) Sell() int64 {
	return m.sell
}

func (m mp) Price() *num.Uint {
	return m.price
}

func (m mp) ClearPotentials() {}

func (m mp) BuySumProduct() *num.Uint {
	return num.UintZero()
}

func (m mp) SellSumProduct() *num.Uint {
	return num.UintZero()
}

func (m mp) VWBuy() *num.Uint {
	return num.UintZero()
}

func (m mp) VWSell() *num.Uint {
	return num.UintZero()
}

func TestHash(t *testing.T) {
	e := getTestEngine(t)
	orders := []*types.Order{
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(100),
			Remaining: uint64(100),
			Price:     num.UintZero(),
		},
		{
			Party:     "test_party_2",
			Side:      types.SideBuy,
			Size:      uint64(200),
			Remaining: uint64(200),
			Price:     num.UintZero(),
		},
	}

	matchingPrice := num.NewUint(10000)
	tradeSize := uint64(15)
	passiveOrder := &types.Order{
		ID:        "buy_order_id",
		Party:     "test_party_3",
		Side:      types.SideBuy,
		Size:      tradeSize,
		Remaining: tradeSize,
		Price:     matchingPrice,
	}

	aggresiveOrder := &types.Order{
		ID:        "sell_order_id",
		Party:     "test_party_1",
		Side:      types.SideSell,
		Size:      tradeSize,
		Remaining: tradeSize,
		Price:     matchingPrice,
	}

	orders = append(orders, passiveOrder, aggresiveOrder)

	for _, order := range orders {
		e.RegisterOrder(context.TODO(), order)
	}

	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     matchingPrice,
		Size:      tradeSize,
		Buyer:     passiveOrder.Party,
		Seller:    aggresiveOrder.Party,
		BuyOrder:  passiveOrder.ID,
		SellOrder: aggresiveOrder.ID,
		Timestamp: time.Now().Unix(),
	}
	e.Update(context.Background(), &trade, passiveOrder, aggresiveOrder)

	hash := e.Hash()
	require.Equal(t,
		"05f6edb5f12dff7edd911da41da5962631283a01e13a717d193109454d22d10a",
		hex.EncodeToString(hash),
		"It should match against the known hash",
	)

	// compute the hash 100 times for determinism verification
	for i := 0; i < 100; i++ {
		got := e.Hash()
		require.Equal(t, hash, got)
	}
}

func registerOrder(e *positions.SnapshotEngine, side types.Side, party string, price *num.Uint, size uint64) *types.Order {
	order := &types.Order{
		Party:     party,
		Side:      side,
		Price:     price,
		Remaining: size,
	}
	e.RegisterOrder(context.TODO(), order)
	return order
}

func TestCalcVWAP(t *testing.T) {
	// no previous size, new long position acquired at 100
	require.Equal(t, num.NewUint(100), positions.CalcVWAP(num.UintZero(), 0, 10, num.NewUint(100)))

	// position decreased, not flipping sides
	require.Equal(t, num.NewUint(100), positions.CalcVWAP(num.NewUint(100), 10, -5, num.NewUint(25)))

	// position closed
	require.Equal(t, num.UintZero(), positions.CalcVWAP(num.NewUint(100), 10, -10, num.NewUint(25)))

	// no previous size, new short position acquired at 100
	require.Equal(t, num.NewUint(100), positions.CalcVWAP(num.UintZero(), 0, -10, num.NewUint(100)))

	// position decreased, not flipping sides
	require.Equal(t, num.NewUint(100), positions.CalcVWAP(num.NewUint(100), -10, 5, num.NewUint(25)))

	// position closed
	require.Equal(t, num.UintZero(), positions.CalcVWAP(num.NewUint(100), -10, 10, num.NewUint(25)))

	// long position increased => (100 * 10 + 25 * 5) / 15
	require.Equal(t, num.NewUint(75), positions.CalcVWAP(num.NewUint(100), 10, 5, num.NewUint(25)))

	// short position increased => (100 * -10 + 25 * -5) / 15
	require.Equal(t, num.NewUint(75), positions.CalcVWAP(num.NewUint(100), -10, -5, num.NewUint(25)))

	// flipping from long to short => (100 * 10 + 15 * 15)/25
	require.Equal(t, num.NewUint(49), positions.CalcVWAP(num.NewUint(100), -10, 25, num.NewUint(15)))

	// flipping from short to long => (100 * 10 + 15 * 15)/25
	require.Equal(t, num.NewUint(49), positions.CalcVWAP(num.NewUint(100), 10, -25, num.NewUint(15)))
}
