package positions_test

import (
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/events"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/settlement"
	"code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestUpdatePosition(t *testing.T) {
	ch := make(chan events.MarketPosition, 2)
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	buyer := "buyer_id"
	seller := "seller_id"
	size := int64(10)
	trade := proto.Trade{
		Id:        "trade_id",
		MarketID:  "market_id",
		Price:     10000,
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
	engine.Update(&trade, ch)
	close(ch)
	wg.Wait()
	assert.Empty(t, ch)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))
	for _, p := range pos {
		if p.Party() == buyer {
			assert.Equal(t, size, p.Size())
		} else {
			assert.Equal(t, -size, p.Size())
		}
	}
}

func getTestEngine(t *testing.T) *positions.Engine {
	return positions.New(
		logging.NewTestLogger(), positions.NewDefaultConfig(),
	)
}
