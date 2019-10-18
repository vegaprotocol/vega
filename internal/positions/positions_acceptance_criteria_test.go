package positions_test

import (
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/settlement"
	"code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestPositionsEngineAcceptanceCriteria(t *testing.T) {
	t.Run("Open long position, trades occur increasing long position", testTradeOccurIncreaseShortAndLong)
	t.Run("Open long position, trades occur decreasing long position", testTradeOccurDecreaseShortAndLong)
	t.Run("Open short position, trades occur increasing (greater abs(size)) short position", testTradeOccurIncreaseShortAndLong)
	t.Run("Open short position, trades occur decreasing (smaller abs(size)) short position", testTradeOccurDecreaseShortAndLong)
}

func testTradeOccurIncreaseShortAndLong(t *testing.T) {
	ch := make(chan events.MarketPosition, 2)
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	// create an initial trade, so we get buyer to +size
	// and seller to -size
	buyer := "buyer_id"
	seller := "seller_id"
	size := int64(10)
	trade := proto.Trade{
		Id:        "trade_id",
		MarketID:  "market_id",
		Price:     100,
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}

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
	engine.Update(&trade, ch)
	close(ch)
	wg.Wait()
	assert.Empty(t, ch)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))
	// make sure that both positions are as expected
	// of size 10
	for _, p := range pos {
		if p.Party() == buyer {
			assert.Equal(t, size, p.Size())
		} else if p.Party() == seller {
			assert.Equal(t, -size, p.Size())
		}
	}

	// now we create a second trade for a size of 25
	size := int64(10)
	trade := proto.Trade{
		Id:        "trade_id",
		MarketID:  "market_id",
		Price:     100,
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}

}

func testTradeOccurDecreaseShortAndLong(t *testing.T) {

}
