package positions_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/positions"
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
		MarketID:  "market_id",
		Price:     10000,
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	positions := engine.Update(&trade)
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

func TestRemoveDistressedEmpty(t *testing.T) {
	data := []events.MarketPosition{
		mp{
			party: "test",
			size:  1,
			price: 1000,
		},
	}
	e := getTestEngine(t)
	ret := e.RemoveDistressed(data)
	assert.Empty(t, ret)
}

func TestRegisterUnregiserOrder(t *testing.T) {
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
	orderBuy := proto.Order{
		PartyID: "test_trader",
		Side:    proto.Side_Buy,
		Size:    uint64(buysize),
	}
	pos, err := e.RegisterOrder(&orderBuy)
	assert.NoError(t, err)
	assert.Equal(t, buysize, pos.Buy())
	assert.Zero(t, pos.Sell())
	assert.Zero(t, pos.Price())
	assert.Zero(t, pos.Size())
	positions := e.Positions()
	assert.Equal(t, 1, len(positions))
	assert.Equal(t, pos.Buy(), positions[0].Buy())

	orderSell := proto.Order{
		PartyID: "test_trader",
		Side:    proto.Side_Sell,
		Size:    uint64(sellsize),
	}
	pos, err = e.RegisterOrder(&orderSell)
	assert.NoError(t, err)
	assert.Equal(t, buysize, pos.Buy())
	assert.Equal(t, sellsize, pos.Sell())
	assert.Zero(t, pos.Price())
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
	orderBuy := proto.Order{
		PartyID: "test_trader",
		Side:    proto.Side_Buy,
		Size:    uint64(buysize),
	}
	pos, err := e.RegisterOrder(&orderBuy)
	assert.NoError(t, err)
	assert.Equal(t, buysize, pos.Buy())

	pos, err = e.UnregisterOrder(&orderBuy)
	assert.NoError(t, err)
	assert.Zero(t, pos.Buy())

	orderSell := proto.Order{
		PartyID: "test_trader",
		Side:    proto.Side_Sell,
		Size:    uint64(sellsize),
	}
	pos, err = e.RegisterOrder(&orderSell)
	assert.NoError(t, err)
	assert.Zero(t, pos.Buy())
	assert.Equal(t, sellsize, pos.Sell())

	pos, err = e.UnregisterOrder(&orderSell)
	assert.NoError(t, err)
	assert.Zero(t, pos.Buy())
	assert.Zero(t, pos.Sell())
}

func testUnregisterOrderUnsuccessful(t *testing.T) {
	e := getTestEngine(t)
	orderBuy := proto.Order{
		PartyID: "test_trader",
		Side:    proto.Side_Buy,
		Size:    uint64(999),
	}
	pos, err := e.UnregisterOrder(&orderBuy)
	assert.Equal(t, err, positions.ErrPositionNotFound)
	assert.Nil(t, pos)
}

func getTestEngine(t *testing.T) *positions.Engine {
	return positions.New(
		logging.NewTestLogger(), positions.NewDefaultConfig(),
	)
}

type mp struct {
	size, buy, sell int64
	party           string
	price           uint64
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

func (m mp) Price() uint64 {
	return m.price
}

func (m mp) ClearPotentials() {}
