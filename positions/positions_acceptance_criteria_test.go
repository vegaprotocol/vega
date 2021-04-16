package positions_test

import (
	"fmt"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"
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
	// Opening and closing positions for multiple traders, maintains position size for all open (non-zero) positions
	t.Run("Does not change position size for a wash trade (buyer = seller)", testWashTradeDoNotChangePosition)

	//No active buy orders, a new buy order is added to the order book
	t.Run("Active buy orders, a new buy order is added to the order book", testNewOrderAddedToTheBook)
	t.Run("Active sell orders, a new sell order is added to the order book", testNewOrderAddedToTheBook)
	t.Run("Active buy order, an order initiated by another trader causes a partial amount of the existing buy order to trade.", testNewTradePartialAmountOfExistingOrderTraded)
	t.Run("Active sell order, an order initiated by another trader causes a partial amount of the existing sell order to trade.", testNewTradePartialAmountOfExistingOrderTraded)
	t.Run("Active buy order, an order initiated by another trader causes the full amount of the existing buy order to trade.", testTradeCauseTheFullAmountOfOrderToTrade)
	t.Run("Active sell order, an order initiated by another trader causes the full amount of the existing sell order to trade.", testTradeCauseTheFullAmountOfOrderToTrade)
	t.Run("Active buy orders, an existing order is cancelled", testOrderCancelled)
	t.Run("Active sell orders, an existing order is cancelled", testOrderCancelled)

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
		trade               types.Trade
		expectedSizeTraderA int64
		expectedSizeTraderB int64
	}{
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     buyer,
				Seller:    seller,
				BuyOrder:  "buy_order_id",
				SellOrder: "sell_order_id",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +10,
			expectedSizeTraderB: -10,
		},
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id",
				MarketId:  "market_id",
				Price:     100,
				Size:      25,
				Buyer:     buyer,
				Seller:    seller,
				BuyOrder:  "buy_order_id",
				SellOrder: "sell_order_id",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +35,
			expectedSizeTraderB: -35,
		},
	}

	for _, c := range cases {
		// call an update on the positions with the trade
		registerOrder(engine, types.Side_SIDE_BUY, c.trade.Buyer, c.trade.Price, c.trade.Size)
		registerOrder(engine, types.Side_SIDE_SELL, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(&c.trade)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == buyer {
				assert.Equal(t, c.expectedSizeTraderA, p.Size())
			} else if p.Party() == seller {
				assert.Equal(t, c.expectedSizeTraderB, p.Size())
			}
		}
	}
}

func testTradeOccurDecreaseShortAndLong(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	traderA := "trader_a"
	traderB := "trader_b"
	cases := []struct {
		trade               types.Trade
		expectedSizeTraderA int64
		expectedSizeTraderB int64
	}{
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_i1",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     traderA,
				Seller:    traderB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +10,
			expectedSizeTraderB: -10,
		},
		// inverse buyer and seller, so it should reduce both position of 5
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id2",
				MarketId:  "market_id",
				Price:     100,
				Size:      5,
				Buyer:     traderB,
				Seller:    traderA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +5,
			expectedSizeTraderB: -5,
		},
	}

	for _, c := range cases {
		// call an update on the positions with the trade
		registerOrder(engine, types.Side_SIDE_BUY, c.trade.Buyer, c.trade.Price, c.trade.Size)
		registerOrder(engine, types.Side_SIDE_SELL, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(&c.trade)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == traderA {
				assert.Equal(t, c.expectedSizeTraderA, p.Size())
			} else if p.Party() == traderB {
				assert.Equal(t, c.expectedSizeTraderB, p.Size())
			}
		}
	}
}

func testTradeOccurClosingShortAndLong(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	traderA := "trader_a"
	traderB := "trader_b"
	cases := []struct {
		trade               types.Trade
		expectedSizeTraderA int64
		expectedSizeTraderB int64
	}{
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_i1",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     traderA,
				Seller:    traderB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +10,
			expectedSizeTraderB: -10,
		},
		// inverse buyer and seller, so it should reduce both position of 5
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id2",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     traderB,
				Seller:    traderA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: 0,
			expectedSizeTraderB: 0,
		},
	}

	for _, c := range cases {
		registerOrder(engine, types.Side_SIDE_BUY, c.trade.Buyer, c.trade.Price, c.trade.Size)
		registerOrder(engine, types.Side_SIDE_SELL, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(&c.trade)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == traderA {
				assert.Equal(t, c.expectedSizeTraderA, p.Size())
			} else if p.Party() == traderB {
				assert.Equal(t, c.expectedSizeTraderB, p.Size())
			}
		}
	}
}

func testTradeOccurShortBecomeLongAndLongBecomeShort(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	traderA := "trader_a"
	traderB := "trader_b"
	cases := []struct {
		trade               types.Trade
		expectedSizeTraderA int64
		expectedSizeTraderB int64
	}{
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_i1",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     traderA,
				Seller:    traderB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +10,
			expectedSizeTraderB: -10,
		},
		// inverse buyer and seller, so it should reduce both position of 5
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id2",
				MarketId:  "market_id",
				Price:     100,
				Size:      15,
				Buyer:     traderB,
				Seller:    traderA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: -5,
			expectedSizeTraderB: +5,
		},
	}

	for _, c := range cases {
		registerOrder(engine, types.Side_SIDE_BUY, c.trade.Buyer, c.trade.Price, c.trade.Size)
		registerOrder(engine, types.Side_SIDE_SELL, c.trade.Seller, c.trade.Price, c.trade.Size)
		// call an update on the positions with the trade
		positions := engine.Update(&c.trade)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == traderA {
				assert.Equal(t, c.expectedSizeTraderA, p.Size())
			} else if p.Party() == traderB {
				assert.Equal(t, c.expectedSizeTraderB, p.Size())
			}
		}
	}
}

func testNoOpenPositionsTradeOccurOpenLongAndShortPosition(t *testing.T) {
	engine := getTestEngine(t)
	traderA := "trader_a"
	traderB := "trader_b"
	c := struct {
		trade               types.Trade
		expectedSizeTraderA int64
		expectedSizeTraderB int64
	}{
		trade: types.Trade{
			Type:      types.Trade_TYPE_DEFAULT,
			Id:        "trade_i1",
			MarketId:  "market_id",
			Price:     100,
			Size:      10,
			Buyer:     traderA,
			Seller:    traderB,
			BuyOrder:  "buy_order_id1",
			SellOrder: "sell_order_id1",
			Timestamp: time.Now().Unix(),
		},
		expectedSizeTraderA: +10,
		expectedSizeTraderB: -10,
	}

	// ensure there is no open positions in the engine
	assert.Empty(t, engine.Positions())

	// now create a trade an make sure the positions are created an correct
	registerOrder(engine, types.Side_SIDE_BUY, c.trade.Buyer, c.trade.Price, c.trade.Size)
	registerOrder(engine, types.Side_SIDE_SELL, c.trade.Seller, c.trade.Price, c.trade.Size)
	positions := engine.Update(&c.trade)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))

	// check size of positions
	for _, p := range pos {
		if p.Party() == traderA {
			assert.Equal(t, c.expectedSizeTraderA, p.Size())
		} else if p.Party() == traderB {
			assert.Equal(t, c.expectedSizeTraderB, p.Size())
		}
	}

}

func testOpenPosTradeOccurCloseThanOpenPositioAgain(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	traderA := "trader_a"
	traderB := "trader_b"
	traderC := "trader_c"
	cases := []struct {
		trade               types.Trade
		expectedSizeTraderA int64
		expectedSizeTraderB int64
		expectedSizeTraderC int64
		posSize             int
	}{
		// first trade between A and B, open a new position
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_i1",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     traderA,
				Seller:    traderB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +10,
			expectedSizeTraderB: -10,
			expectedSizeTraderC: 0,
			posSize:             2,
		},
		// second trade between A and C, open C close A
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id2",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     traderC,
				Seller:    traderA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: 0,
			expectedSizeTraderB: -10,
			expectedSizeTraderC: 10,
			posSize:             3,
		},
		// last trade between A and B again, re-open A, decrease B
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id3",
				MarketId:  "market_id",
				Price:     100,
				Size:      3,
				Buyer:     traderB,
				Seller:    traderA,
				BuyOrder:  "buy_order_id3",
				SellOrder: "sell_order_id3",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: -3,
			expectedSizeTraderB: -7,
			expectedSizeTraderC: 10,
			posSize:             3,
		},
	}

	for _, c := range cases {
		registerOrder(engine, types.Side_SIDE_BUY, c.trade.Buyer, c.trade.Price, c.trade.Size)
		registerOrder(engine, types.Side_SIDE_SELL, c.trade.Seller, c.trade.Price, c.trade.Size)
		positions := engine.Update(&c.trade)
		pos := engine.Positions()
		assert.Equal(t, c.posSize, len(pos), fmt.Sprintf("all pos trade: %v", c.trade.Id))
		assert.Equal(t, 2, len(positions), fmt.Sprintf("chan trade: %v", c.trade.Id))
		fmt.Printf("positions: %v\n", positions)

		// check size of positions
		for _, p := range pos {
			if p.Party() == traderA {
				assert.Equal(t, c.expectedSizeTraderA, p.Size())
			} else if p.Party() == traderB {
				assert.Equal(t, c.expectedSizeTraderB, p.Size())
			} else if p.Party() == traderC {
				assert.Equal(t, c.expectedSizeTraderC, p.Size())
			}
		}
	}

}

func testWashTradeDoNotChangePosition(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	traderA := "trader_a"
	traderB := "trader_b"
	cases := []struct {
		trade               types.Trade
		expectedSizeTraderA int64
		expectedSizeTraderB int64
	}{
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_i1",
				MarketId:  "market_id",
				Price:     100,
				Size:      10,
				Buyer:     traderA,
				Seller:    traderB,
				BuyOrder:  "buy_order_id1",
				SellOrder: "sell_order_id1",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +10,
			expectedSizeTraderB: -10,
		},
		// trader A trade with himsefl, no positions changes
		{
			trade: types.Trade{
				Type:      types.Trade_TYPE_DEFAULT,
				Id:        "trade_id2",
				MarketId:  "market_id",
				Price:     100,
				Size:      30,
				Buyer:     traderA,
				Seller:    traderA,
				BuyOrder:  "buy_order_id2",
				SellOrder: "sell_order_id2",
				Timestamp: time.Now().Unix(),
			},
			expectedSizeTraderA: +10,
			expectedSizeTraderB: -10,
		},
	}

	for _, c := range cases {
		registerOrder(engine, types.Side_SIDE_BUY, c.trade.Buyer, c.trade.Price, c.trade.Size)
		registerOrder(engine, types.Side_SIDE_SELL, c.trade.Seller, c.trade.Price, c.trade.Size)
		// call an update on the positions with the trade
		positions := engine.Update(&c.trade)
		pos := engine.Positions()
		assert.Equal(t, 2, len(pos))
		assert.Equal(t, 2, len(positions))

		// check size of positions
		for _, p := range pos {
			if p.Party() == traderA {
				assert.Equal(t, c.expectedSizeTraderA, p.Size())
			} else if p.Party() == traderB {
				assert.Equal(t, c.expectedSizeTraderB, p.Size())
			}
		}
	}
}

func testNewOrderAddedToTheBook(t *testing.T) {
	engine := getTestEngine(t)
	traderA := "trader_a"
	traderB := "trader_b"
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
				PartyId:   traderA,
				Side:      types.Side_SIDE_BUY,
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
				PartyId:   traderB,
				Side:      types.Side_SIDE_SELL,
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
				PartyId:   traderA,
				Side:      types.Side_SIDE_BUY,
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
				PartyId:   traderB,
				Side:      types.Side_SIDE_SELL,
			},
			expectedBuy:  0,
			expectedSell: 21,
			expectedSize: 0,
		},
	}

	// no potions exists at the moment:
	assert.Empty(t, engine.Positions())

	for _, c := range cases {
		pos := engine.RegisterOrder(&c.order)
		assert.Equal(t, c.expectedBuy, pos.Buy())
		assert.Equal(t, c.expectedSell, pos.Sell())
		assert.Equal(t, c.expectedSize, pos.Size())
	}
}

func testNewTradePartialAmountOfExistingOrderTraded(t *testing.T) {
	engine := getTestEngine(t)
	traderA := "trader_a"
	traderB := "trader_b"
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
				PartyId:   traderA,
				Side:      types.Side_SIDE_BUY,
			},
			{
				Size:      16,
				Remaining: 16,
				PartyId:   traderB,
				Side:      types.Side_SIDE_SELL,
			},
		},
		expects: map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}{
			traderA: {
				expectedBuy:  10,
				expectedSell: 0,
				expectedSize: 0,
			},
			traderB: {
				expectedBuy:  0,
				expectedSell: 16,
				expectedSize: 0,
			},
		},
	}

	// no potions exists at the moment:
	assert.Empty(t, engine.Positions())

	for i, c := range cases.orders {
		engine.RegisterOrder(&c)
		// ensure we have 1 position with 1 potential buy of size 10 for traderA
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
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade_i1",
		MarketId:  "market_id",
		Price:     100,
		Size:      3,
		Buyer:     traderA,
		Seller:    traderB,
		BuyOrder:  "buy_order_id1",
		SellOrder: "sell_order_id1",
		Timestamp: time.Now().Unix(),
	}

	// add the trade
	// call an update on the positions with the trade
	positions := engine.Update(&trade)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))

	// check size of positions
	for _, p := range pos {
		if p.Party() == traderA {
			assert.Equal(t, int64(3), p.Size())
			assert.Equal(t, int64(7), p.Buy())
		} else if p.Party() == traderB {
			assert.Equal(t, int64(-3), p.Size())
			assert.Equal(t, int64(13), p.Sell())
		}
	}
}

func testTradeCauseTheFullAmountOfOrderToTrade(t *testing.T) {
	engine := getTestEngine(t)
	traderA := "trader_a"
	traderB := "trader_b"
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
				PartyId:   traderA,
				Side:      types.Side_SIDE_BUY,
			},
			{
				Size:      10,
				Remaining: 10,
				PartyId:   traderB,
				Side:      types.Side_SIDE_SELL,
			},
		},
		expects: map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}{
			traderA: {
				expectedBuy:  10,
				expectedSell: 0,
				expectedSize: 0,
			},
			traderB: {
				expectedBuy:  0,
				expectedSell: 10,
				expectedSize: 0,
			},
		},
	}

	// no potions exists at the moment:
	assert.Empty(t, engine.Positions())

	for i, c := range cases.orders {
		engine.RegisterOrder(&c)
		// ensure we have 1 position with 1 potential buy of size 10 for traderA
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
		Type:      types.Trade_TYPE_DEFAULT,
		Id:        "trade_i1",
		MarketId:  "market_id",
		Price:     100,
		Size:      10,
		Buyer:     traderA,
		Seller:    traderB,
		BuyOrder:  "buy_order_id1",
		SellOrder: "sell_order_id1",
		Timestamp: time.Now().Unix(),
	}

	// add the trade
	// call an update on the positions with the trade
	positions := engine.Update(&trade)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))

	// check size of positions
	for _, p := range pos {
		if p.Party() == traderA {
			assert.Equal(t, int64(10), p.Size())
			assert.Equal(t, int64(0), p.Buy())
		} else if p.Party() == traderB {
			assert.Equal(t, int64(-10), p.Size())
			assert.Equal(t, int64(0), p.Sell())
		}
	}
}

func testOrderCancelled(t *testing.T) {
	engine := getTestEngine(t)
	traderA := "trader_a"
	traderB := "trader_b"
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
				PartyId:   traderA,
				Side:      types.Side_SIDE_BUY,
			},
			{
				Size:      10,
				Remaining: 10,
				PartyId:   traderB,
				Side:      types.Side_SIDE_SELL,
			},
		},
		expects: map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}{
			traderA: {
				expectedBuy:  10,
				expectedSell: 0,
				expectedSize: 0,
			},
			traderB: {
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
		engine.RegisterOrder(&c)
		// ensure we have 1 position with 1 potential buy of size 10 for traderA
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
				PartyId:   traderA,
				Side:      types.Side_SIDE_BUY,
			},
			{
				Size:      10,
				Remaining: 10,
				PartyId:   traderB,
				Side:      types.Side_SIDE_SELL,
			},
		},
		expects: map[string]struct {
			expectedBuy  int64
			expectedSell int64
			expectedSize int64
		}{
			traderA: {
				expectedBuy:  0,
				expectedSell: 0,
				expectedSize: 0,
			},
			traderB: {
				expectedBuy:  0,
				expectedSell: 0,
				expectedSize: 0,
			},
		},
	}

	// first add the orders
	for _, c := range cases.orders {
		_ = engine.UnregisterOrder(&c)
	}

	// test everything is back to 0 once orders are unregistered
	pos := engine.Positions()
	for _, v := range pos {
		assert.Equal(t, cases.expects[v.Party()].expectedBuy, v.Buy())
		assert.Equal(t, cases.expects[v.Party()].expectedSell, v.Sell())
		assert.Equal(t, cases.expects[v.Party()].expectedSize, v.Size())
	}
}
