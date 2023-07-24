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

package positions_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"
)

func TestPositionsEngineAcceptanceCriteria(t *testing.T) {
	t.Run("Open long position, trades occur increasing long position", testTradeOccurIncreaseShortAndLong)
	t.Run("Open long position, trades occur decreasing long position", testTradeOccurDecreaseShortAndLong)
	t.Run("Open short position, trades occur increasing (greater abs(size)) short position", testTradeOccurIncreaseShortAndLong)
	t.Run("Open short position, trades occur decreasing (smaller abs(size)) short position", testTradeOccurDecreaseShortAndLong)
	t.Run("Open short position, trades occur taking position to zero (closing it)", testTradeOccurClosingShortAndLong)
	t.Run("Open long position, trades occur taking position to zero (closing it)", testTradeOccurClosingShortAndLong)
	t.Run("Open short position, trades occur closing the short position and opening a long position", testTradeOccurShortBecomeLongAndLongBecomeShort)
	t.Run("Open long position, trades occur closing the long position and opening a short position", testTradeOccurShortBecomeLongAndLongBecomeShort)
	t.Run("No open position, trades occur opening a long position", testNoOpenPositionsTradeOccurOpenLongAndShortPosition)
	t.Run("No open position, trades occur opening a short position", testNoOpenPositionsTradeOccurOpenLongAndShortPosition)
	t.Run("Open position, trades occur that close it (take it to zero), in a separate transaction, trades occur and that open a new position", testOpenPosTradeOccurCloseThanOpenPositioAgain)
	// NOTE: this will not be tested, as we do not remove a position from the engine when it reach 0
	// Opening and closing positions for multiple partys, maintains position size for all open (non-zero) positions
	t.Run("Does not change position size for a wash trade (buyer = seller)", testWashTradeDoNotChangePosition)

	// No active buy orders, a new buy order is added to the order book
	t.Run("Active buy orders, a new buy order is added to the order book", testNewOrderAddedToTheBook)
	t.Run("Active sell orders, a new sell order is added to the order book", testNewOrderAddedToTheBook)
	t.Run("Active buy order, an order initiated by another party causes a partial amount of the existing buy order to trade.", testNewTradePartialAmountOfExistingOrderTraded)
	t.Run("Active sell order, an order initiated by another party causes a partial amount of the existing sell order to trade.", testNewTradePartialAmountOfExistingOrderTraded)
	t.Run("Active buy order, an order initiated by another party causes the full amount of the existing buy order to trade.", testTradeCauseTheFullAmountOfOrderToTrade)
	t.Run("Active sell order, an order initiated by another party causes the full amount of the existing sell order to trade.", testTradeCauseTheFullAmountOfOrderToTrade)
	t.Run("Active buy orders, an existing order is cancelled", testOrderCancelled)
	t.Run("Active sell orders, an existing order is cancelled", testOrderCancelled)
	t.Run("Aggressive order gets partially filled", testNewTradePartialAmountOfIncomingOrderTraded)

	// NOTE: these next tests needs the integration test to be ran
	// Active buy orders, an existing buy order is amended which increases its size.
	// Active buy orders, an existing buy order is amended which decreases its size.
	// Active buy orders, an existing buy order's price is amended such that it trades a partial amount.
	// Active buy orders, an existing buy order's price is amended such that it trades in full.
	// Active buy orders, an existing order expires
}

func testTradeOccurIncreaseShortAndLong(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	buyer := "buyer_id"
	seller := "seller_id"
	cases := []struct {
		trade              types.Trade
		expectedSizePartyA int64
		expectedSizePartyB int64
	}{
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     buyer,
				Seller:    seller,
				BuyOrder:  "buy_order_id",
				SellOrder: "sell_order_id",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +10,
			expectedSizePartyB: -10,
		},
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      25,
				Buyer:     buyer,
				Seller:    seller,
				BuyOrder:  "buy_order_id",
				SellOrder: "sell_order_id",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +35,
			expectedSizePartyB: -35,
		},
	}

	for _, c := range cases {
		// call an update on the positions with the trade
		passive := registerOrder(engine, types.SideBuy, c.trade.Buyer, c.trade.Price, c.trade.Size)
		aggressive := registerOrder(engine, types.SideSell, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(context.Background(), &c.trade, passive, aggressive)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == buyer {
				assert.Equal(t, c.expectedSizePartyA, p.Size())
			} else if p.Party() == seller {
				assert.Equal(t, c.expectedSizePartyB, p.Size())
			}
		}
	}
}

func testTradeOccurDecreaseShortAndLong(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	partyA := "party_a"
	partyB := "party_b"
	cases := []struct {
		trade              types.Trade
		expectedSizePartyA int64
		expectedSizePartyB int64
	}{
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_i1",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     partyA,
				Seller:    partyB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +10,
			expectedSizePartyB: -10,
		},
		// inverse buyer and seller, so it should reduce both position of 5
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id2",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      5,
				Buyer:     partyB,
				Seller:    partyA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +5,
			expectedSizePartyB: -5,
		},
	}

	for _, c := range cases {
		// call an update on the positions with the trade
		passive := registerOrder(engine, types.SideBuy, c.trade.Buyer, c.trade.Price, c.trade.Size)
		aggressive := registerOrder(engine, types.SideSell, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(context.Background(), &c.trade, passive, aggressive)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == partyA {
				assert.Equal(t, c.expectedSizePartyA, p.Size())
			} else if p.Party() == partyB {
				assert.Equal(t, c.expectedSizePartyB, p.Size())
			}
		}
	}
}

func testTradeOccurClosingShortAndLong(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	partyA := "party_a"
	partyB := "party_b"
	cases := []struct {
		trade              types.Trade
		expectedSizePartyA int64
		expectedSizePartyB int64
	}{
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_i1",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     partyA,
				Seller:    partyB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +10,
			expectedSizePartyB: -10,
		},
		// inverse buyer and seller, so it should reduce both position of 5
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id2",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     partyB,
				Seller:    partyA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: 0,
			expectedSizePartyB: 0,
		},
	}

	for _, c := range cases {
		passive := registerOrder(engine, types.SideBuy, c.trade.Buyer, c.trade.Price, c.trade.Size)
		aggressive := registerOrder(engine, types.SideSell, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(context.Background(), &c.trade, passive, aggressive)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == partyA {
				assert.Equal(t, c.expectedSizePartyA, p.Size())
			} else if p.Party() == partyB {
				assert.Equal(t, c.expectedSizePartyB, p.Size())
			}
		}
	}
}

func testTradeOccurShortBecomeLongAndLongBecomeShort(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	partyA := "party_a"
	partyB := "party_b"
	cases := []struct {
		trade              types.Trade
		expectedSizePartyA int64
		expectedSizePartyB int64
	}{
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_i1",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     partyA,
				Seller:    partyB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +10,
			expectedSizePartyB: -10,
		},
		// inverse buyer and seller, so it should reduce both position of 5
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id2",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      15,
				Buyer:     partyB,
				Seller:    partyA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: -5,
			expectedSizePartyB: +5,
		},
	}

	for _, c := range cases {
		passive := registerOrder(engine, types.SideBuy, c.trade.Buyer, c.trade.Price, c.trade.Size)
		aggressive := registerOrder(engine, types.SideSell, c.trade.Seller, c.trade.Price, c.trade.Size)
		// call an update on the positions with the trade
		positions := engine.Update(context.Background(), &c.trade, passive, aggressive)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == partyA {
				assert.Equal(t, c.expectedSizePartyA, p.Size())
			} else if p.Party() == partyB {
				assert.Equal(t, c.expectedSizePartyB, p.Size())
			}
		}
	}
}

func testNoOpenPositionsTradeOccurOpenLongAndShortPosition(t *testing.T) {
	engine := getTestEngine(t)
	partyA := "party_a"
	partyB := "party_b"
	c := struct {
		trade              types.Trade
		expectedSizePartyA int64
		expectedSizePartyB int64
	}{
		trade: types.Trade{
			Type:      types.TradeTypeDefault,
			ID:        "trade_i1",
			MarketID:  "market_id",
			Price:     num.NewUint(100),
			Size:      10,
			Buyer:     partyA,
			Seller:    partyB,
			BuyOrder:  "buy_order_id1",
			SellOrder: "sell_order_id1",
			Timestamp: time.Now().Unix(),
		},
		expectedSizePartyA: +10,
		expectedSizePartyB: -10,
	}

	// ensure there is no open positions in the engine
	assert.Empty(t, engine.Positions())

	// now create a trade an make sure the positions are created an correct
	passive := registerOrder(engine, types.SideBuy, c.trade.Buyer, c.trade.Price, c.trade.Size)
	aggressive := registerOrder(engine, types.SideSell, c.trade.Seller, c.trade.Price, c.trade.Size)
	positions := engine.Update(context.Background(), &c.trade, passive, aggressive)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))

	// check size of positions
	for _, p := range pos {
		if p.Party() == partyA {
			assert.Equal(t, c.expectedSizePartyA, p.Size())
		} else if p.Party() == partyB {
			assert.Equal(t, c.expectedSizePartyB, p.Size())
		}
	}
}

func testOpenPosTradeOccurCloseThanOpenPositioAgain(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	partyA := "party_a"
	partyB := "party_b"
	partyC := "party_c"
	cases := []struct {
		trade              types.Trade
		expectedSizePartyA int64
		expectedSizePartyB int64
		expectedSizePartyC int64
		posSize            int
	}{
		// first trade between A and B, open a new position
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_i1",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     partyA,
				Seller:    partyB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +10,
			expectedSizePartyB: -10,
			expectedSizePartyC: 0,
			posSize:            2,
		},
		// second trade between A and C, open C close A
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id2",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     partyC,
				Seller:    partyA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: 0,
			expectedSizePartyB: -10,
			expectedSizePartyC: 10,
			posSize:            3,
		},
		// last trade between A and B again, re-open A, decrease B
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id3",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      3,
				Buyer:     partyB,
				Seller:    partyA,
				BuyOrder:  "buy_order_id3",
				SellOrder: "sell_order_id3",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: -3,
			expectedSizePartyB: -7,
			expectedSizePartyC: 10,
			posSize:            3,
		},
	}

	for _, c := range cases {
		passive := registerOrder(engine, types.SideBuy, c.trade.Buyer, c.trade.Price, c.trade.Size)
		aggressive := registerOrder(engine, types.SideSell, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(context.Background(), &c.trade, passive, aggressive)
		pos := engine.Positions()
		assert.Equal(t, c.posSize, len(pos), fmt.Sprintf("all pos trade: %v", c.trade.ID))
		assert.Equal(t, 2, len(positions), fmt.Sprintf("chan trade: %v", c.trade.ID))

		// check size of positions
		for _, p := range pos {
			if p.Party() == partyA {
				assert.Equal(t, c.expectedSizePartyA, p.Size())
			} else if p.Party() == partyB {
				assert.Equal(t, c.expectedSizePartyB, p.Size())
			} else if p.Party() == partyC {
				assert.Equal(t, c.expectedSizePartyC, p.Size())
			}
		}
	}
}

func testWashTradeDoNotChangePosition(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	partyA := "party_a"
	partyB := "party_b"
	cases := []struct {
		trade              types.Trade
		expectedSizePartyA int64
		expectedSizePartyB int64
	}{
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_i1",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      10,
				Buyer:     partyA,
				Seller:    partyB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +10,
			expectedSizePartyB: -10,
		},
		// party A trade with himsefl, no positions changes
		{
			trade: types.Trade{
				Type:      types.TradeTypeDefault,
				ID:        "trade_id2",
				MarketID:  "market_id",
				Price:     num.NewUint(100),
				Size:      30,
				Buyer:     partyA,
				Seller:    partyA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizePartyA: +10,
			expectedSizePartyB: -10,
		},
	}

	for _, c := range cases {
		passive := registerOrder(engine, types.SideBuy, c.trade.Buyer, c.trade.Price, c.trade.Size)
		aggressive := registerOrder(engine, types.SideSell, c.trade.Seller, c.trade.Price, c.trade.Size)
		// call an update on the positions with the trade
		positions := engine.Update(context.Background(), &c.trade, passive, aggressive)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == partyA {
				assert.Equal(t, c.expectedSizePartyA, p.Size())
			} else if p.Party() == partyB {
				assert.Equal(t, c.expectedSizePartyB, p.Size())
			}
		}
	}
}

func testNewOrderAddedToTheBook(t *testing.T) {
	engine := getTestEngine(t)
	partyA := "party_a"
	partyB := "party_b"
	cases := []struct {
		order        types.Order
		expectedBuy  int64
		expectedSell int64
		expectedSize int64
	}{
		{
			// add an original buy order for A
			order: types.Order{
				Size:      10,
				Remaining: 10,
				Party:     partyA,
				Side:      types.SideBuy,
				Price:     num.UintZero(),
			},
			expectedBuy:  10,
			expectedSell: 0,
			expectedSize: 0,
		},
		{
			// add and original sell order for B
			order: types.Order{
				Size:      16,
				Remaining: 16,
				Party:     partyB,
				Side:      types.SideSell,
				Price:     num.UintZero(),
			},
			expectedBuy:  0,
			expectedSell: 16,
			expectedSize: 0,
		},
		{
			// update buy order for A
			order: types.Order{
				Size:      17,
				Remaining: 17,
				Party:     partyA,
				Side:      types.SideBuy,
				Price:     num.UintZero(),
			},
			expectedBuy:  27,
			expectedSell: 0,
			expectedSize: 0,
		},
		{
			// update sell order for B
			order: types.Order{
				Size:      5,
				Remaining: 5,
				Party:     partyB,
				Side:      types.SideSell,
				Price:     num.UintZero(),
			},
			expectedBuy:  0,
			expectedSell: 21,
			expectedSize: 0,
		},
	}

	// no potions exists at the moment:
	assert.Empty(t, engine.Positions())

	for _, c := range cases {
		pos := engine.RegisterOrder(context.TODO(), &c.order)
		assert.Equal(t, c.expectedBuy, pos.Buy())
		assert.Equal(t, c.expectedSell, pos.Sell())
		assert.Equal(t, c.expectedSize, pos.Size())
	}
}

func testNewTradePartialAmountOfExistingOrderTraded(t *testing.T) {
	engine := getTestEngine(t)
	partyA := "party_a"
	partyB := "party_b"
	matchingPrice := num.NewUint(100)
	tradeSize := uint64(3)

	passive := &types.Order{
		Size:      7 + tradeSize,
		Remaining: 7 + tradeSize,
		Party:     partyA,
		Side:      types.SideBuy,
		Price:     matchingPrice,
	}

	aggressive := &types.Order{
		Size:      tradeSize,
		Remaining: tradeSize,
		Party:     partyB,
		Side:      types.SideSell,
		Price:     matchingPrice,
	}

	cases := struct {
		orders  []*types.Order
		expects map[string]struct {
			expectedBuy    int64
			expectedSell   int64
			expectedSize   int64
			expectedVwBuy  *num.Uint
			expectedVwSell *num.Uint
		}
	}{
		orders: []*types.Order{
			passive,
			aggressive,
			{
				Size:      16 - tradeSize,
				Remaining: 16 - tradeSize,
				Party:     partyB,
				Side:      types.SideSell,
				Price:     num.NewUint(1000),
			},
		},
		expects: map[string]struct {
			expectedBuy    int64
			expectedSell   int64
			expectedSize   int64
			expectedVwBuy  *num.Uint
			expectedVwSell *num.Uint
		}{
			partyA: {
				expectedBuy:    10,
				expectedSell:   0,
				expectedSize:   0,
				expectedVwBuy:  passive.Price,
				expectedVwSell: num.UintZero(),
			},
			partyB: {
				expectedBuy:    0,
				expectedSell:   16,
				expectedSize:   0,
				expectedVwBuy:  num.UintOne(),
				expectedVwSell: num.NewUint(831), // 831.25
			},
		},
	}

	// no positions exists at the moment:
	assert.Empty(t, engine.Positions())

	for _, c := range cases.orders {
		engine.RegisterOrder(context.TODO(), c)
	}
	pos := engine.Positions()
	assert.Len(t, pos, len(cases.expects))
	for _, v := range pos {
		assert.Equal(t, cases.expects[v.Party()].expectedBuy, v.Buy())
		assert.Equal(t, cases.expects[v.Party()].expectedSell, v.Sell())
		assert.Equal(t, cases.expects[v.Party()].expectedSize, v.Size())
	}

	// add a trade for a size of 3,
	// potential buy should be 7, size 3
	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_i1",
		MarketID:  "market_id",
		Price:     num.NewUint(100),
		Size:      3,
		Buyer:     partyA,
		Seller:    partyB,
		BuyOrder:  "buy_order_id1",
		SellOrder: "sell_order_id1",
		Timestamp: time.Now().Unix(),
	}

	// add the trade
	// call an update on the positions with the trade
	positions := engine.Update(context.Background(), &trade, passive, aggressive)
	pos = engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))

	// check size of positions
	for _, p := range pos {
		if p.Party() == partyA {
			assert.Equal(t, int64(3), p.Size())
			assert.Equal(t, int64(7), p.Buy())
			assert.Equal(t, cases.orders[0].Price, p.VWBuy())
			assert.Equal(t, num.UintZero(), p.VWSell())
		} else if p.Party() == partyB {
			assert.Equal(t, int64(-3), p.Size())
			assert.Equal(t, int64(13), p.Sell())
			assert.Equal(t, num.UintZero(), p.VWBuy())
			assert.Equal(t, cases.orders[len(cases.orders)-1].Price, p.VWSell())
		}
	}
}

func testNewTradePartialAmountOfIncomingOrderTraded(t *testing.T) {
	engine := getTestEngine(t)
	partyA := "party_a"
	partyB := "party_b"
	matchingPrice := num.NewUint(100)
	tradeSize := uint64(3)

	passive := &types.Order{
		Size:      tradeSize,
		Remaining: tradeSize,
		Party:     partyB,
		Side:      types.SideSell,
		Price:     matchingPrice,
	}

	aggressive := &types.Order{
		Size:      5 + tradeSize,
		Remaining: 5 + tradeSize,
		Party:     partyA,
		Side:      types.SideBuy,
		Price:     matchingPrice,
	}

	cases := struct {
		orders  []*types.Order
		expects map[string]struct {
			expectedBuy    int64
			expectedSell   int64
			expectedSize   int64
			expectedVwBuy  *num.Uint
			expectedVwSell *num.Uint
		}
	}{
		orders: []*types.Order{
			{
				Size:      16,
				Remaining: 16,
				Party:     partyB,
				Side:      types.SideSell,
				Price:     num.NewUint(1000),
			},
			passive,
			aggressive,
		},
		expects: map[string]struct {
			expectedBuy    int64
			expectedSell   int64
			expectedSize   int64
			expectedVwBuy  *num.Uint
			expectedVwSell *num.Uint
		}{
			partyA: {
				expectedBuy:    8,
				expectedSell:   0,
				expectedSize:   0,
				expectedVwBuy:  aggressive.Price,
				expectedVwSell: num.UintZero(),
			},
			partyB: {
				expectedBuy:    0,
				expectedSell:   19,
				expectedSize:   0,
				expectedVwBuy:  num.UintOne(),
				expectedVwSell: num.NewUint(857), // 857.8947368421
			},
		},
	}

	// no positions exists at the moment:
	assert.Empty(t, engine.Positions())

	for _, c := range cases.orders {
		engine.RegisterOrder(context.TODO(), c)
	}
	pos := engine.Positions()
	assert.Len(t, pos, len(cases.expects))
	for _, v := range pos {
		assert.Equal(t, cases.expects[v.Party()].expectedBuy, v.Buy())
		assert.Equal(t, cases.expects[v.Party()].expectedSell, v.Sell())
		assert.Equal(t, cases.expects[v.Party()].expectedSize, v.Size())
	}

	// add a trade for a size of 3,
	// potential buy should be 5, size 3
	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_i1",
		MarketID:  "market_id",
		Price:     num.NewUint(100),
		Size:      3,
		Buyer:     partyA,
		Seller:    partyB,
		BuyOrder:  "buy_order_id1",
		SellOrder: "sell_order_id1",
		Timestamp: time.Now().Unix(),
	}

	// add the trade
	// call an update on the positions with the trade
	positions := engine.Update(context.Background(), &trade, passive, aggressive)
	pos = engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))

	// check size of positions
	for _, p := range pos {
		if p.Party() == partyA {
			assert.Equal(t, int64(3), p.Size())
			assert.Equal(t, int64(5), p.Buy())
			assert.Equal(t, matchingPrice, p.VWBuy())
			assert.Equal(t, num.UintZero(), p.VWSell())
		} else if p.Party() == partyB {
			assert.Equal(t, int64(-3), p.Size())
			assert.Equal(t, int64(16), p.Sell())
			assert.Equal(t, num.UintZero(), p.VWBuy())
			assert.Equal(t, cases.orders[0].Price, p.VWSell())
		}
	}
}

func testTradeCauseTheFullAmountOfOrderToTrade(t *testing.T) {
	engine := getTestEngine(t)
	partyA := "party_a"
	partyB := "party_b"
	cases := struct {
		orders  []types.Order
		expects map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}
	}{
		orders: []types.Order{
			{
				Size:      10,
				Remaining: 10,
				Party:     partyA,
				Side:      types.SideBuy,
				Price:     num.UintZero(),
			},
			{
				Size:      10,
				Remaining: 10,
				Party:     partyB,
				Side:      types.SideSell,
				Price:     num.UintZero(),
			},
		},
		expects: map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}{
			partyA: {
				expectedBuy:  10,
				expectedSell: 0,
				expectedSize: 0,
			},
			partyB: {
				expectedBuy:  0,
				expectedSell: 10,
				expectedSize: 0,
			},
		},
	}

	// no potions exists at the moment:
	assert.Empty(t, engine.Positions())

	for i, c := range cases.orders {
		engine.RegisterOrder(context.TODO(), &c)
		// ensure we have 1 position with 1 potential buy of size 10 for partyA
		pos := engine.Positions()
		assert.Len(t, pos, i+1)
		for _, v := range pos {
			assert.Equal(t, cases.expects[v.Party()].expectedBuy, v.Buy())
			assert.Equal(t, cases.expects[v.Party()].expectedSell, v.Sell())
			assert.Equal(t, cases.expects[v.Party()].expectedSize, v.Size())
		}
	}
	// add a trade for a size of 3,
	// potential buy should be 7, size 3
	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_i1",
		MarketID:  "market_id",
		Price:     num.NewUint(100),
		Size:      10,
		Buyer:     partyA,
		Seller:    partyB,
		BuyOrder:  "buy_order_id1",
		SellOrder: "sell_order_id1",
		Timestamp: time.Now().Unix(),
	}

	// add the trade
	// call an update on the positions with the trade
	positions := engine.Update(context.Background(), &trade, &cases.orders[0], &cases.orders[1])
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))

	// check size of positions
	for _, p := range pos {
		if p.Party() == partyA {
			assert.Equal(t, int64(10), p.Size())
			assert.Equal(t, int64(0), p.Buy())
		} else if p.Party() == partyB {
			assert.Equal(t, int64(-10), p.Size())
			assert.Equal(t, int64(0), p.Sell())
		}
	}
}

func testOrderCancelled(t *testing.T) {
	engine := getTestEngine(t)
	partyA := "party_a"
	partyB := "party_b"
	cases := struct {
		orders  []types.Order
		expects map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}
	}{
		orders: []types.Order{
			{
				Size:      10,
				Remaining: 10,
				Party:     partyA,
				Side:      types.SideBuy,
				Price:     num.UintZero(),
			},
			{
				Size:      10,
				Remaining: 10,
				Party:     partyB,
				Side:      types.SideSell,
				Price:     num.UintZero(),
			},
		},
		expects: map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}{
			partyA: {
				expectedBuy:  10,
				expectedSell: 0,
				expectedSize: 0,
			},
			partyB: {
				expectedBuy:  0,
				expectedSell: 10,
				expectedSize: 0,
			},
		},
	}

	// no potions exists at the moment:
	assert.Empty(t, engine.Positions())

	// first add the orders
	for i, c := range cases.orders {
		engine.RegisterOrder(context.TODO(), &c)
		// ensure we have 1 position with 1 potential buy of size 10 for partyA
		pos := engine.Positions()
		assert.Len(t, pos, i+1)
		for _, v := range pos {
			assert.Equal(t, cases.expects[v.Party()].expectedBuy, v.Buy())
			assert.Equal(t, cases.expects[v.Party()].expectedSell, v.Sell())
			assert.Equal(t, cases.expects[v.Party()].expectedSize, v.Size())
		}
	}

	// then remove them
	cases = struct {
		orders  []types.Order
		expects map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}
	}{
		orders: []types.Order{
			{
				Size:      10,
				Remaining: 10,
				Party:     partyA,
				Side:      types.SideBuy,
				Price:     num.UintZero(),
			},
			{
				Size:      10,
				Remaining: 10,
				Party:     partyB,
				Side:      types.SideSell,
				Price:     num.UintZero(),
			},
		},
		expects: map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}{
			partyA: {
				expectedBuy:  0,
				expectedSell: 0,
				expectedSize: 0,
			},
			partyB: {
				expectedBuy:  0,
				expectedSell: 0,
				expectedSize: 0,
			},
		},
	}

	// first add the orders
	for _, c := range cases.orders {
		_ = engine.UnregisterOrder(context.TODO(), &c)
	}

	// test everything is back to 0 once orders are unregistered
	pos := engine.Positions()
	for _, v := range pos {
		assert.Equal(t, cases.expects[v.Party()].expectedBuy, v.Buy())
		assert.Equal(t, cases.expects[v.Party()].expectedSell, v.Sell())
		assert.Equal(t, cases.expects[v.Party()].expectedSize, v.Size())
	}
}
