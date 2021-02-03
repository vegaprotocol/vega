package positions_test

import (
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePosition(t *testing.T) {
	t.Run("Update position regular", testUpdatePositionRegular)
	t.Run("Update position network trade as buyer", testUpdatePositionNetworkBuy)
	t.Run("Update position network trade as seller", testUpdatePositionNetworkSell)
}

func TestGetOpenInterest(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	buyer := "buyer_id"
	buyer2 := "buyer_id2"
	seller := "seller_id"
	size := int64(10)
	trade := proto.Trade{
		Type:      proto.Trade_TYPE_DEFAULT,
		Id:        "trade_id",
		MarketId:  "market_id",
		Price:     10000,
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	_ = engine.Update(&trade)
	trade = proto.Trade{
		Type:      proto.Trade_TYPE_DEFAULT,
		Id:        "trade_id",
		MarketId:  "market_id",
		Price:     10000,
		Size:      uint64(size),
		Buyer:     buyer2,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	_ = engine.Update(&trade)
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
	buyer := "buyer_id"
	seller := "seller_id"
	size := int64(10)
	trade := proto.Trade{
		Type:      proto.Trade_TYPE_DEFAULT,
		Id:        "trade_id",
		MarketId:  "market_id",
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

func testUpdatePositionNetworkBuy(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	buyer := "network"
	seller := "seller_id"
	size := int64(10)
	trade := proto.Trade{
		Type:      proto.Trade_TYPE_DEFAULT,
		Id:        "trade_id",
		MarketId:  "market_id",
		Price:     10000,
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	positions := engine.UpdateNetwork(&trade)
	pos := engine.Positions()
	assert.Equal(t, 1, len(pos))
	assert.Equal(t, 1, len(positions))
	assert.Equal(t, seller, pos[0].Party())
	assert.Equal(t, -size, pos[0].Size())
}

func testUpdatePositionNetworkSell(t *testing.T) {
	engine := getTestEngine(t)
	assert.Empty(t, engine.Positions())
	buyer := "buyer_id"
	seller := "network"
	size := int64(10)
	trade := proto.Trade{
		Type:      proto.Trade_TYPE_DEFAULT,
		Id:        "trade_id",
		MarketId:  "market_id",
		Price:     10000,
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	positions := engine.UpdateNetwork(&trade)
	pos := engine.Positions()
	assert.Equal(t, 1, len(pos))
	assert.Equal(t, 1, len(positions))
	assert.Equal(t, buyer, pos[0].Party())
	assert.Equal(t, size, pos[0].Size())
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
		PartyId:   "test_trader",
		Side:      proto.Side_SIDE_BUY,
		Size:      uint64(buysize),
		Remaining: uint64(buysize),
	}
	pos := e.RegisterOrder(&orderBuy)
	assert.Equal(t, buysize, pos.Buy())
	assert.Zero(t, pos.Sell())
	assert.Zero(t, pos.Price())
	assert.Zero(t, pos.Size())
	positions := e.Positions()
	assert.Equal(t, 1, len(positions))
	assert.Equal(t, pos.Buy(), positions[0].Buy())

	orderSell := proto.Order{
		PartyId:   "test_trader",
		Side:      proto.Side_SIDE_SELL,
		Size:      uint64(sellsize),
		Remaining: uint64(sellsize),
	}
	pos = e.RegisterOrder(&orderSell)
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
		PartyId:   "test_trader",
		Side:      proto.Side_SIDE_BUY,
		Size:      uint64(buysize),
		Remaining: uint64(buysize),
	}
	pos := e.RegisterOrder(&orderBuy)
	assert.Equal(t, buysize, pos.Buy())

	pos, err := e.UnregisterOrder(&orderBuy)
	assert.NoError(t, err)
	assert.Zero(t, pos.Buy())

	orderSell := proto.Order{
		PartyId:   "test_trader",
		Side:      proto.Side_SIDE_SELL,
		Size:      uint64(sellsize),
		Remaining: uint64(sellsize),
	}
	pos = e.RegisterOrder(&orderSell)
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
		PartyId:   "test_trader",
		Side:      proto.Side_SIDE_BUY,
		Size:      uint64(999),
		Remaining: uint64(999),
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

func (m mp) VWBuy() uint64 {
	return 0
}

func (m mp) VWSell() uint64 {
	return 0
}

func TestHash(t *testing.T) {
	e := getTestEngine(t)
	orders := []proto.Order{
		{
			PartyId:   "test_trader_1",
			Side:      proto.Side_SIDE_BUY,
			Size:      uint64(100),
			Remaining: uint64(100),
		},
		{
			PartyId:   "test_trader_2",
			Side:      proto.Side_SIDE_BUY,
			Size:      uint64(200),
			Remaining: uint64(200),
		},
		{
			PartyId:   "test_trader_3",
			Side:      proto.Side_SIDE_BUY,
			Size:      uint64(300),
			Remaining: uint64(300),
		},
		{
			PartyId:   "test_trader_1",
			Side:      proto.Side_SIDE_SELL,
			Size:      uint64(1000),
			Remaining: uint64(1000),
		},
	}

	for _, order := range orders {
		e.RegisterOrder(&order)
	}

	trade := proto.Trade{
		Type:      proto.Trade_TYPE_DEFAULT,
		Id:        "trade_id",
		MarketId:  "market_id",
		Price:     10000,
		Size:      uint64(15),
		Buyer:     "test_trader_3",
		Seller:    "test_trader_1",
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	e.Update(&trade)

	hash := e.Hash()
	require.Equal(t,
		"7cbb54f5ecf8be4378b6380361d3f8f425c6c0ec2c36728eda03a162a3d0d676",
		hex.EncodeToString(hash),
		"It should match against the known hash",
	)

	// compute the hash 100 times for determinism verification
	for i := 0; i < 100; i++ {
		got := e.Hash()
		require.Equal(t, hash, got)
	}
}
