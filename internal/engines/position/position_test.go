package position_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/engines/position"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestUpdatePosition(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	buyer := "buyer_id"
	seller := "seller_id"
	size := int64(10)
	trade := proto.Trade{
		Id:        "trade_id",
		Market:    "market_id",
		Price:     10000,
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	engine.Update(&trade)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	for _, p := range pos {
		if p.Party() == buyer {
			assert.Equal(t, size, p.Size())
		} else {
			assert.Equal(t, -size, p.Size())
		}
	}
}

func getTestEngine(t *testing.T) *position.Engine {
	return position.New(
		position.NewDefaultConfig(
			logging.NewTestLogger(),
		),
	)
}
