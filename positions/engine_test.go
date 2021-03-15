package positions_test

import (
	"encoding/hex"
	"testing"
	"time"

	"code.vegatypes.ol.io/vega/events"
	types "code.vegatypes.ol.io/vega/types.

	"code.vegatypes.ol.io/vega/logging"
	"code.vegatypes.ol.io/vega/positions"

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
	trade := types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
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
	trade = types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
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
	trade := types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
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
	trade := types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
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
	trade := types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
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
		PartyId:   "test_trader",
		Side:      types.Side_SIDE_BUY,
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

	orderSell := types.Order{
		PartyId:   "test_trader",
		Side:      types.Side_SIDE_SELL,
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
	orderBuy := types.Order{
		PartyId:   "test_trader",
		Side:      types.Side_SIDE_BUY,
		Size:      uint64(buysize),
		Remaining: uint64(buysize),
	}
	pos := e.RegisterOrder(&orderBuy)
	assert.Equal(t, buysize, pos.Buy())

	pos, err := e.UnregisterOrder(&orderBuy)
	assert.NoError(t, err)
	assert.Zero(t, pos.Buy())

	orderSell := types.Order{
		PartyId:   "test_trader",
		Side:      types.Side_SIDE_SELL,
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
	orderBuy := types.Order{
		PartyId:   "test_trader",
		Side:      types.Side_SIDE_BUY,
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

func TestGetOpenInterestGivenTrades(t *testing.T) {
	//A, B represents traders who already have positions
	//C, D represents traders who don't have positions (but there are entries in "trades" array that contain their trades)

	cases := []struct {
		ExistingPositions []*types.Trade
		Trades            []*types.Trade
		ExpectedOI        uint64
	}{
		//Both parties already have positions
		{ //A: + 100, B: -100 => OI: 100
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100},
			},
			ExpectedOI: 100,
		},
		{ //A: + 100 - 10, B: -100 + 10=> OI: 90
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 10},
			},
			ExpectedOI: 90,
		},
		{ //A: + 100 + 10, B: -100 - 10 => OI: 110
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 10},
			},
			ExpectedOI: 110,
		},

		// There at least 1 new party
		{ //A: + 100 + 10, B: -100, C: -10 => OI: 110
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10},
			},
			ExpectedOI: 110,
		},
		{ //A: + 100 - 10, B: -100, C: +10 => OI: 100
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "C", Size: 10},
			},
			ExpectedOI: 100,
		},

		//None of the parties have positions yet
		{ //C: +10, D:-10 => OI: 10
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 10},
			},
			ExpectedOI: 10,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 5},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 5},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "C", Buyer: "B", Size: 5},
			},
			ExpectedOI: 205,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "C", Size: 5},
			},
			ExpectedOI: 195,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 500},
			},
			ExpectedOI: 500,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 5},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 5},
			},
			ExpectedOI: 110,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 5},
			},
			ExpectedOI: 110,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10},
				{Seller: "C", Buyer: "B", Size: 100},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 200},
			},
			ExpectedOI: 300,
		},
	}

	for _, tc := range cases {
		e := getTestEngine(t)

		for _, tr := range tc.ExistingPositions {
			e.Update(tr)
		}

		oiGivenTrades := e.GetOpenInterestGivenTrades(tc.Trades)

		for _, tr := range tc.Trades {
			e.Update(tr)
		}

		// Now check it matches ones those trades are registered as positions
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
	orders := []types.Order{
		{
			PartyId:   "test_trader_1",
			Side:      types.Side_SIDE_BUY,
			Size:      uint64(100),
			Remaining: uint64(100),
		},
		{
			PartyId:   "test_trader_2",
			Side:      types.Side_SIDE_BUY,
			Size:      uint64(200),
			Remaining: uint64(200),
		},
		{
			PartyId:   "test_trader_3",
			Side:      types.Side_SIDE_BUY,
			Size:      uint64(300),
			Remaining: uint64(300),
		},
		{
			PartyId:   "test_trader_1",
			Side:      types.Side_SIDE_SELL,
			Size:      uint64(1000),
			Remaining: uint64(1000),
		},
	}

	for _, order := range orders {
		e.RegisterOrder(&order)
	}

	trade := types.Trade{
		Type:      types.Trade_TYPE_DEFAULT,
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
