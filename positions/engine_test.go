package positions_test

import (
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/types"

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
	var (
		buyer         = "buyer_id"
		buyer2        = "buyer_id2"
		seller        = "seller_id"
		size   uint64 = 10
		price         = num.NewUint(10000)
	)
	engine.RegisterOrder(&types.Order{
		Party:     buyer,
		Remaining: size,
		Price:     price,
		Side:      types.SideBuy,
	})
	engine.RegisterOrder(&types.Order{
		Party:     buyer2,
		Remaining: size,
		Price:     price,
		Side:      types.SideBuy,
	})
	engine.RegisterOrder(&types.Order{
		Party:     seller,
		Remaining: size * 2,
		Price:     price,
		Side:      types.SideSell,
	})

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
	_ = engine.Update(&trade)
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
	var (
		buyer         = "buyer_id"
		seller        = "seller_id"
		size   uint64 = 10
		price         = num.NewUint(10000)
	)
	engine.RegisterOrder(&types.Order{
		Party:     buyer,
		Remaining: size,
		Price:     price,
		Side:      types.SideBuy,
	})
	engine.RegisterOrder(&types.Order{
		Party:     seller,
		Remaining: size,
		Price:     price,
		Side:      types.SideSell,
	})

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
	positions := engine.Update(&trade)
	pos := engine.Positions()
	assert.Equal(t, 2, len(pos))
	assert.Equal(t, 2, len(positions))
	for _, p := range pos {
		if p.Party() == buyer {
			assert.Equal(t, int64(size), p.Size())
		} else {
			assert.Equal(t, int64(-size), p.Size())
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
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     num.NewUint(10000),
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	registerOrder(engine, types.SideSell, seller, num.NewUint(10000), uint64(size))
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
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     num.NewUint(10000),
		Size:      uint64(size),
		Buyer:     buyer,
		Seller:    seller,
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	registerOrder(engine, types.SideBuy, buyer, num.NewUint(10000), uint64(size))
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
		Price:     num.Zero(),
	}
	pos := e.RegisterOrder(&orderBuy)
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
		Price:     num.Zero(),
	}
	pos = e.RegisterOrder(&orderSell)
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
		Price:     num.Zero(),
	}
	pos := e.RegisterOrder(&orderBuy)
	assert.Equal(t, buysize, pos.Buy())

	pos = e.UnregisterOrder(&orderBuy)
	assert.Zero(t, pos.Buy())

	orderSell := types.Order{
		Party:     "test_party",
		Side:      types.SideSell,
		Size:      uint64(sellsize),
		Remaining: uint64(sellsize),
		Price:     num.Zero(),
	}
	pos = e.RegisterOrder(&orderSell)
	assert.Zero(t, pos.Buy())
	assert.Equal(t, sellsize, pos.Sell())

	pos = e.UnregisterOrder(&orderSell)
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
		Price:     num.Zero(),
	}
	require.Panics(t, func() {
		_ = e.UnregisterOrder(&orderBuy)
	})
}

func getTestEngine(t *testing.T) *positions.Engine {
	t.Helper()
	return positions.New(
		logging.NewTestLogger(), positions.NewDefaultConfig(),
		"test_market",
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
				{Seller: "B", Buyer: "A", Size: 100, Price: num.Zero()},
			},
			ExpectedOI: 100,
		},
		{ // A: + 100 - 10, B: -100 + 10=> OI: 90
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 10, Price: num.Zero()},
			},
			ExpectedOI: 90,
		},
		{ // A: + 100 + 10, B: -100 - 10 => OI: 110
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 10, Price: num.Zero()},
			},
			ExpectedOI: 110,
		},

		// There at least 1 new party
		{ // A: + 100 + 10, B: -100, C: -10 => OI: 110
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.Zero()},
			},
			ExpectedOI: 110,
		},
		{ // A: + 100 - 10, B: -100, C: +10 => OI: 100
			ExistingPositions: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "C", Size: 10, Price: num.Zero()},
			},
			ExpectedOI: 100,
		},

		// None of the parties have positions yet
		{ // C: +10, D:-10 => OI: 10
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 10, Price: num.Zero()},
			},
			ExpectedOI: 10,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 5, Price: num.Zero()},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 5, Price: num.Zero()},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "C", Buyer: "B", Size: 5, Price: num.Zero()},
			},
			ExpectedOI: 205,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "C", Size: 5, Price: num.Zero()},
			},
			ExpectedOI: 195,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 500, Price: num.Zero()},
			},
			ExpectedOI: 500,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 100, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "D", Buyer: "C", Size: 5, Price: num.Zero()},
			},
			ExpectedOI: 200,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 5, Price: num.Zero()},
			},
			ExpectedOI: 110,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "B", Buyer: "A", Size: 5, Price: num.Zero()},
			},
			ExpectedOI: 110,
		},
		{
			ExistingPositions: []*types.Trade{
				{Seller: "C", Buyer: "A", Size: 10, Price: num.Zero()},
				{Seller: "C", Buyer: "B", Size: 100, Price: num.Zero()},
			},
			Trades: []*types.Trade{
				{Seller: "A", Buyer: "B", Size: 200, Price: num.Zero()},
			},
			ExpectedOI: 300,
		},
	}

	for _, tc := range cases {
		e := getTestEngine(t)

		for _, tr := range tc.ExistingPositions {
			registerOrder(e, types.SideBuy, tr.Buyer, tr.Price, tr.Size)
			registerOrder(e, types.SideSell, tr.Seller, tr.Price, tr.Size)
			e.Update(tr)
		}

		oiGivenTrades := e.GetOpenInterestGivenTrades(tc.Trades)

		for _, tr := range tc.Trades {
			registerOrder(e, types.SideBuy, tr.Buyer, tr.Price, tr.Size)
			registerOrder(e, types.SideSell, tr.Seller, tr.Price, tr.Size)
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
	price           *num.Uint
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

func (m mp) VWBuy() *num.Uint {
	return num.Zero()
}

func (m mp) VWSell() *num.Uint {
	return num.Zero()
}

func TestHash(t *testing.T) {
	e := getTestEngine(t)
	orders := []types.Order{
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(100),
			Remaining: uint64(100),
			Price:     num.Zero(),
		},
		{
			Party:     "test_party_2",
			Side:      types.SideBuy,
			Size:      uint64(200),
			Remaining: uint64(200),
			Price:     num.Zero(),
		},
		{
			Party:     "test_party_3",
			Side:      types.SideBuy,
			Size:      uint64(300),
			Remaining: uint64(300),
			Price:     num.Zero(),
		},
		{
			Party:     "test_party_1",
			Side:      types.SideSell,
			Size:      uint64(1000),
			Remaining: uint64(1000),
			Price:     num.Zero(),
		},
	}

	for _, order := range orders {
		e.RegisterOrder(&order)
	}

	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     num.NewUint(10000),
		Size:      uint64(15),
		Buyer:     "test_party_3",
		Seller:    "test_party_1",
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	e.Update(&trade)

	hash := e.Hash()
	require.Equal(t,
		"6ad38d53e438c8a0c5d67f6304c1e29e436b24491a72934e3b748bd1e21768e4",
		hex.EncodeToString(hash),
		"It should match against the known hash",
	)

	// compute the hash 100 times for determinism verification
	for i := 0; i < 100; i++ {
		got := e.Hash()
		require.Equal(t, hash, got)
	}
}

func registerOrder(e *positions.Engine, side types.Side, party string, price *num.Uint, size uint64) {
	e.RegisterOrder(&types.Order{
		Party:     party,
		Side:      side,
		Price:     price,
		Remaining: size,
	})
}
