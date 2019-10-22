package positions_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/settlement"
	"code.vegaprotocol.io/vega/proto"
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
			trade: proto.Trade{
				Id:        "trade_id",
				MarketID:  "market_id",
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
			trade: proto.Trade{
				Id:        "trade_id",
				MarketID:  "market_id",
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
		ch := make(chan events.MarketPosition, 2)
		wg := sync.WaitGroup{}
		positions := make([]settlement.MarketPosition, 0, 2)
		wg.Add(1)
		go func() {
			for p := range ch {
				positions = append(positions, p)
			}
			wg.Done()
		}()
		// call an update on the positions with the trade
		engine.Update(&c.trade, ch)
		close(ch)
		wg.Wait()
		assert.Empty(t, ch)
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
			trade: proto.Trade{
				Id:        "trade_i1",
				MarketID:  "market_id",
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
			trade: proto.Trade{
				Id:        "trade_id2",
				MarketID:  "market_id",
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
		ch := make(chan events.MarketPosition, 2)
		wg := sync.WaitGroup{}
		positions := make([]settlement.MarketPosition, 0, 2)
		wg.Add(1)
		go func() {
			for p := range ch {
				positions = append(positions, p)
			}
			wg.Done()
		}()
		// call an update on the positions with the trade
		engine.Update(&c.trade, ch)
		close(ch)
		wg.Wait()
		assert.Empty(t, ch)
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
			trade: proto.Trade{
				Id:        "trade_i1",
				MarketID:  "market_id",
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
			trade: proto.Trade{
				Id:        "trade_id2",
				MarketID:  "market_id",
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
		ch := make(chan events.MarketPosition, 2)
		wg := sync.WaitGroup{}
		positions := make([]settlement.MarketPosition, 0, 2)
		wg.Add(1)
		go func() {
			for p := range ch {
				positions = append(positions, p)
			}
			wg.Done()
		}()
		// call an update on the positions with the trade
		engine.Update(&c.trade, ch)
		close(ch)
		wg.Wait()
		assert.Empty(t, ch)
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
			trade: proto.Trade{
				Id:        "trade_i1",
				MarketID:  "market_id",
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
			trade: proto.Trade{
				Id:        "trade_id2",
				MarketID:  "market_id",
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
		ch := make(chan events.MarketPosition, 2)
		wg := sync.WaitGroup{}
		positions := make([]settlement.MarketPosition, 0, 2)
		wg.Add(1)
		go func() {
			for p := range ch {
				positions = append(positions, p)
			}
			wg.Done()
		}()
		// call an update on the positions with the trade
		engine.Update(&c.trade, ch)
		close(ch)
		wg.Wait()
		assert.Empty(t, ch)
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
		trade: proto.Trade{
			Id:        "trade_i1",
			MarketID:  "market_id",
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
	ch := make(chan events.MarketPosition, 2)
	wg := sync.WaitGroup{}
	positions := make([]settlement.MarketPosition, 0, 2)
	wg.Add(1)
	go func() {
		for p := range ch {
			positions = append(positions, p)
		}
		wg.Done()
	}()
	// call an update on the positions with the trade
	engine.Update(&c.trade, ch)
	close(ch)
	wg.Wait()
	assert.Empty(t, ch)
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
			trade: proto.Trade{
				Id:        "trade_i1",
				MarketID:  "market_id",
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
			trade: proto.Trade{
				Id:        "trade_id2",
				MarketID:  "market_id",
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
			trade: proto.Trade{
				Id:        "trade_id3",
				MarketID:  "market_id",
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
		ch := make(chan events.MarketPosition, 2)
		wg := sync.WaitGroup{}
		positions := make([]settlement.MarketPosition, 0, 2)
		wg.Add(1)
		go func() {
			for p := range ch {
				positions = append(positions, p)
			}
			wg.Done()
		}()
		// call an update on the positions with the trade
		engine.Update(&c.trade, ch)
		close(ch)
		wg.Wait()
		assert.Empty(t, ch)
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
			trade: proto.Trade{
				Id:        "trade_i1",
				MarketID:  "market_id",
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
		// trader A trade with himsef, no positions changes
		{
			trade: proto.Trade{
				Id:        "trade_id2",
				MarketID:  "market_id",
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
		ch := make(chan events.MarketPosition, 2)
		wg := sync.WaitGroup{}
		positions := make([]settlement.MarketPosition, 0, 2)
		wg.Add(1)
		go func() {
			for p := range ch {
				positions = append(positions, p)
			}
			wg.Done()
		}()
		// call an update on the positions with the trade
		engine.Update(&c.trade, ch)
		close(ch)
		wg.Wait()
		assert.Empty(t, ch)
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
