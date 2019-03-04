package gql

import (
	context "context"
	"flag"
	"fmt"
	"testing"
	"time"

	"vega/internal"
	"vega/internal/blockchain"
	bcmocks "vega/internal/blockchain/mocks"
	"vega/internal/candles"
	"vega/internal/logging"
	"vega/internal/markets"
	"vega/internal/orders"
	"vega/internal/parties"
	"vega/internal/storage"
	"vega/internal/trades"
	"vega/internal/vegatime"
	types "vega/proto"

	"github.com/stretchr/testify/assert"
)

var datadir = flag.String("vega.datadir", "", "directory containing badger data files")

func TestMarketResolver(t *testing.T) {
	if datadir == nil || len(*datadir) <= 0 {
		t.Fatal("missing vega.datadir argument")
	}

	ctx := context.Background()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	tctx := buildResolver(t, logger)
	defer tctx.Close()

	marketResolver := tctx.root.Market()
	assert.NotNil(t, marketResolver)

	marketID := "BTC/DEC19"
	market := &Market{
		Name: marketID,
	}

	// orders

	t.Run("Get orders by market", func(t *testing.T) {
		orders, err := marketResolver.Orders(ctx, market, nil, nil, nil, nil)
		assert.NotNil(t, orders)
		assert.Nil(t, err)
		for _, v := range orders {
			assert.Equal(t, marketID, v.Market)
		}
	})

	t.Run("Get orders by market with limit", func(t *testing.T) {
		orders, err := marketResolver.Orders(ctx, market, nil, nil, nil, intptr(5))
		assert.NotNil(t, orders)
		assert.Nil(t, err)
		assert.Len(t, orders, 5)
	})

	t.Run("Get orders by market with nil market", func(t *testing.T) {
		orders, err := marketResolver.Orders(ctx, nil, nil, nil, nil, nil)
		assert.Nil(t, orders)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilMarket, err)
	})

	t.Run("Get orders by market with unknown market", func(t *testing.T) {
		badmarket := &Market{
			Name: "ETH/DEC99",
		}
		orders, err := marketResolver.Orders(ctx, badmarket, nil, nil, nil, nil)
		assert.Nil(t, orders)
		assert.NotNil(t, err)
		assert.Equal(t, "market ETH/DEC99 not found in store", err.Error())
	})

	t.Run("Get Orders by market from last with skip", func(t *testing.T) {
		// get 1 last order
		orders1, err := marketResolver.Orders(ctx, market, nil, nil, nil, intptr(1))
		assert.NotNil(t, orders1)
		assert.Nil(t, err)
		assert.Len(t, orders1, 1)

		// get 1 last order skip 1 so get 2 from end
		orders2, err := marketResolver.Orders(ctx, market, nil, intptr(1), nil, intptr(1))
		assert.NotNil(t, orders2)
		assert.Nil(t, err)
		assert.Len(t, orders2, 1)
		assert.NotEqual(t, orders1[0], orders2[0])
	})

	t.Run("Get Orders by market first", func(t *testing.T) {
		// get 1 first order
		orders1, err := marketResolver.Orders(ctx, market, nil, nil, intptr(1), nil)
		assert.NotNil(t, orders1)
		assert.Nil(t, err)
		assert.Len(t, orders1, 1)

		// get 1 last order
		orders2, err := marketResolver.Orders(ctx, market, nil, nil, nil, intptr(1))
		assert.NotNil(t, orders2)
		assert.Nil(t, err)
		assert.Len(t, orders2, 1)
		assert.NotEqual(t, orders1[0], orders2[0])
	})

	t.Run("Get Orders by market from first with skip", func(t *testing.T) {
		// get 1 first order
		orders1, err := marketResolver.Orders(ctx, market, nil, nil, intptr(1), nil)
		assert.NotNil(t, orders1)
		assert.Nil(t, err)
		assert.Len(t, orders1, 1)

		// get 1 first order skip 1 so get 2 from end
		orders2, err := marketResolver.Orders(ctx, market, nil, intptr(1), intptr(1), nil)
		assert.NotNil(t, orders2)
		assert.Nil(t, err)
		assert.Len(t, orders2, 1)
		assert.NotEqual(t, orders1[0], orders2[0])
	})

	// trades

	t.Run("Get trades by market", func(t *testing.T) {
		trades, err := marketResolver.Trades(ctx, market, nil, nil, nil, nil)
		assert.NotNil(t, trades)
		assert.Nil(t, err)
		for _, v := range trades {
			assert.Equal(t, marketID, v.Market)
		}
	})

	t.Run("Get trades by market with limit", func(t *testing.T) {
		trades, err := marketResolver.Trades(ctx, market, nil, nil, nil, intptr(5))
		assert.NotNil(t, trades)
		assert.Nil(t, err)
		assert.Len(t, trades, 5)
	})

	t.Run("Get trades by market with nil market", func(t *testing.T) {
		trades, err := marketResolver.Trades(ctx, nil, nil, nil, nil, nil)
		assert.Nil(t, trades)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilMarket, err)
	})

	t.Run("Get trades by market with unknown market", func(t *testing.T) {
		badmarket := &Market{
			Name: "ETH/DEC99",
		}
		trades, err := marketResolver.Trades(ctx, badmarket, nil, nil, nil, nil)
		assert.Nil(t, trades)
		assert.NotNil(t, err)
		assert.Equal(t, "market ETH/DEC99 not found in store", err.Error())
	})

	t.Run("Get Trades by market from last with skip", func(t *testing.T) {
		// get 1 last trade
		trades1, err := marketResolver.Trades(ctx, market, nil, nil, nil, intptr(1))
		assert.NotNil(t, trades1)
		assert.Nil(t, err)
		assert.Len(t, trades1, 1)

		// get 1 last trade skip 1 so get 2 from end
		trades2, err := marketResolver.Trades(ctx, market, nil, intptr(1), nil, intptr(1))
		assert.NotNil(t, trades2)
		assert.Nil(t, err)
		assert.Len(t, trades2, 1)
		assert.NotEqual(t, trades1[0], trades2[0])
	})

	t.Run("Get Trades by market first", func(t *testing.T) {
		// get 1 first trade
		trades1, err := marketResolver.Trades(ctx, market, nil, nil, intptr(1), nil)
		assert.NotNil(t, trades1)
		assert.Nil(t, err)
		assert.Len(t, trades1, 1)

		// get 1 last trade
		trades2, err := marketResolver.Trades(ctx, market, nil, nil, nil, intptr(1))
		assert.NotNil(t, trades2)
		assert.Nil(t, err)
		assert.Len(t, trades2, 1)
		assert.NotEqual(t, trades1[0], trades2[0])
	})

	t.Run("Get Trades by market from first with skip", func(t *testing.T) {
		// get 1 first trade
		trades1, err := marketResolver.Trades(ctx, market, nil, nil, intptr(1), nil)
		assert.NotNil(t, trades1)
		assert.Nil(t, err)
		assert.Len(t, trades1, 1)

		// get 1 first trade skip 1 so get 2 from end
		trades2, err := marketResolver.Trades(ctx, market, nil, intptr(1), intptr(1), nil)
		assert.NotNil(t, trades2)
		assert.Nil(t, err)
		assert.Len(t, trades2, 1)
		assert.NotEqual(t, trades1[0], trades2[0])
	})

	// depth

	t.Run("Get Market Depth", func(t *testing.T) {
		t.Skip()
		depth, err := marketResolver.Depth(ctx, market)
		assert.Nil(t, err)
		t.Log(depth)
	})

	t.Run("Get Market Depth with n market", func(t *testing.T) {
		_, err := marketResolver.Depth(ctx, nil)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilMarket)
	})

	// candles

	t.Run("Get Candles nil market", func(t *testing.T) {
		ts := fmt.Sprintf("%v", time.Now().UnixNano())
		candles, err := marketResolver.Candles(ctx, nil, ts, IntervalI15M)
		assert.Nil(t, candles)
		assert.NotNil(t, err)
		assert.Equal(t, err, ErrNilMarket)
	})

	t.Run("Get Candles unknown market", func(t *testing.T) {
		ts := fmt.Sprintf("%v", time.Now().UnixNano())
		badmarket := &Market{
			Name: "ETH/DEC99",
		}
		candles, err := marketResolver.Candles(ctx, badmarket, ts, IntervalI15M)
		assert.Nil(t, candles)
		assert.Nil(t, err)
	})

	t.Run("Get Candles by market invalid timestamp", func(t *testing.T) {
		invalidts := "notavalidts"
		candles, err := marketResolver.Candles(ctx, market, invalidts, IntervalI15M)
		assert.Nil(t, candles)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "error converting notavalidts into a valid timestamp")
	})

	t.Run("Get Candles by market require nano format", func(t *testing.T) {
		notnanots := fmt.Sprintf("%v", time.Now().Unix())
		candles, err := marketResolver.Candles(ctx, market, notnanots, IntervalI15M)
		assert.Nil(t, candles)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "timestamp should be in epoch+nanoseconds format, eg. 1545158175835902621")

	})

	t.Run("Get Candle by market no candles", func(t *testing.T) {
		ts := fmt.Sprintf("%v", time.Now().UnixNano())
		candles, err := marketResolver.Candles(ctx, market, ts, IntervalI15M)
		assert.Nil(t, candles)
		assert.Nil(t, err)
		assert.Len(t, candles, 0)
	})

	t.Run("Get Candle by market with valid timestamp", func(t *testing.T) {
		t.Skip()
		ts := fmt.Sprintf(
			"%v", time.Date(2019, time.February, 26, 0, 0, 0, 0, time.UTC).UnixNano())
		candles, err := marketResolver.Candles(ctx, market, ts, IntervalI15M)
		assert.NotNil(t, candles)
		assert.Nil(t, err)
		assert.Len(t, candles, 0)
	})

}

func TestPartyResolver(t *testing.T) {
	if datadir == nil || len(*datadir) <= 0 {
		t.Fatal("missing vega.datadir argument")
	}

	ctx := context.Background()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	tctx := buildResolver(t, logger)
	defer tctx.Close()

	partyResolver := tctx.root.Party()
	assert.NotNil(t, partyResolver)

	partyID := "localjoe"
	party := &Party{
		Name: partyID,
	}

	// orders

	t.Run("Get Orders for party", func(t *testing.T) {
		orders, err := partyResolver.Orders(ctx, party, nil, nil, nil, nil)
		assert.NotNil(t, orders)
		assert.Nil(t, err)
		for _, v := range orders {
			assert.Equal(t, v.Party, party.Name)
		}
	})

	t.Run("Get Orders for party with limit", func(t *testing.T) {
		orders, err := partyResolver.Orders(ctx, party, nil, nil, nil, intptr(5))
		assert.NotNil(t, orders)
		assert.Nil(t, err)
		assert.Len(t, orders, 5)
	})

	t.Run("Get Orders for party with nil party", func(t *testing.T) {
		orders, err := partyResolver.Orders(ctx, nil, nil, nil, nil, intptr(5))
		assert.Nil(t, orders)
		assert.NotNil(t, err, ErrNilParty)
	})

	t.Run("Get Orders for party with unknown party", func(t *testing.T) {
		badparty := &Party{
			Name: "lolparty",
		}
		orders, err := partyResolver.Orders(ctx, badparty, nil, nil, nil, intptr(5))
		assert.Nil(t, orders)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "party lolparty not found in store")
	})

	t.Run("Get Orders by party from last with skip", func(t *testing.T) {
		// get 1 last order
		orders1, err := partyResolver.Orders(ctx, party, nil, nil, nil, intptr(1))
		assert.NotNil(t, orders1)
		assert.Nil(t, err)
		assert.Len(t, orders1, 1)

		// get 1 last order skip 1 so get 2 from end
		orders2, err := partyResolver.Orders(ctx, party, nil, intptr(1), nil, intptr(1))
		assert.NotNil(t, orders2)
		assert.Nil(t, err)
		assert.Len(t, orders2, 1)
		assert.NotEqual(t, orders1[0], orders2[0])
	})

	t.Run("Get Orders by party first", func(t *testing.T) {
		// get 1 first order
		orders1, err := partyResolver.Orders(ctx, party, nil, nil, intptr(1), nil)
		assert.NotNil(t, orders1)
		assert.Nil(t, err)
		assert.Len(t, orders1, 1)

		// get 1 last order
		orders2, err := partyResolver.Orders(ctx, party, nil, nil, nil, intptr(1))
		assert.NotNil(t, orders2)
		assert.Nil(t, err)
		assert.Len(t, orders2, 1)
		assert.NotEqual(t, orders1[0], orders2[0])
	})

	t.Run("Get Orders by party from first with skip", func(t *testing.T) {
		// get 1 first order
		orders1, err := partyResolver.Orders(ctx, party, nil, nil, intptr(1), nil)
		assert.NotNil(t, orders1)
		assert.Nil(t, err)
		assert.Len(t, orders1, 1)

		// get 1 first order skip 1 so get 2 from end
		orders2, err := partyResolver.Orders(ctx, party, nil, intptr(1), intptr(1), nil)
		assert.NotNil(t, orders2)
		assert.Nil(t, err)
		assert.Len(t, orders2, 1)
		assert.NotEqual(t, orders1[0], orders2[0])
	})

	// trades

	t.Run("Get trades by party", func(t *testing.T) {
		trades, err := partyResolver.Trades(ctx, party, nil, nil, nil, nil)
		assert.NotNil(t, trades)
		assert.Nil(t, err)
	})

	t.Run("Get trades by party with limit", func(t *testing.T) {
		trades, err := partyResolver.Trades(ctx, party, nil, nil, nil, intptr(5))
		assert.NotNil(t, trades)
		assert.Nil(t, err)
		assert.Len(t, trades, 5)
	})

	t.Run("Get trades by party with nil party", func(t *testing.T) {
		trades, err := partyResolver.Trades(ctx, nil, nil, nil, nil, nil)
		assert.Nil(t, trades)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilParty, err)
	})

	t.Run("Get trades by party with unknown party", func(t *testing.T) {
		badparty := &Party{
			Name: "lolparty",
		}
		trades, err := partyResolver.Trades(ctx, badparty, nil, nil, nil, nil)
		assert.Nil(t, trades)
		assert.NotNil(t, err)
		assert.Equal(t, "party lolparty not found in store", err.Error())
	})

	t.Run("Get Trades by party from last with skip", func(t *testing.T) {
		// get 1 last trade
		trades1, err := partyResolver.Trades(ctx, party, nil, nil, nil, intptr(1))
		assert.NotNil(t, trades1)
		assert.Nil(t, err)
		assert.Len(t, trades1, 1)

		// get 1 last trade skip 1 so get 2 from end
		trades2, err := partyResolver.Trades(ctx, party, nil, intptr(1), nil, intptr(1))
		assert.NotNil(t, trades2)
		assert.Nil(t, err)
		assert.Len(t, trades2, 1)
		assert.NotEqual(t, trades1[0], trades2[0])
	})

	t.Run("Get Trades by party first", func(t *testing.T) {
		// get 1 first trade
		trades1, err := partyResolver.Trades(ctx, party, nil, nil, intptr(1), nil)
		assert.NotNil(t, trades1)
		assert.Nil(t, err)
		assert.Len(t, trades1, 1)

		// get 1 last trade
		trades2, err := partyResolver.Trades(ctx, party, nil, nil, nil, intptr(1))
		assert.NotNil(t, trades2)
		assert.Nil(t, err)
		assert.Len(t, trades2, 1)
		assert.NotEqual(t, trades1[0], trades2[0])
	})

	t.Run("Get Trades by party from first with skip", func(t *testing.T) {
		// get 1 first trade
		trades1, err := partyResolver.Trades(ctx, party, nil, nil, intptr(1), nil)
		assert.NotNil(t, trades1)
		assert.Nil(t, err)
		assert.Len(t, trades1, 1)

		// get 1 first trade skip 1 so get 2 from end
		trades2, err := partyResolver.Trades(ctx, party, nil, intptr(1), intptr(1), nil)
		assert.NotNil(t, trades2)
		assert.Nil(t, err)
		assert.Len(t, trades2, 1)
		assert.NotEqual(t, trades1[0], trades2[0])
	})

	// positions

	t.Run("Get Position by party", func(t *testing.T) {
		positions, err := partyResolver.Positions(ctx, party)
		assert.NotNil(t, positions)
		assert.Nil(t, err)
		assert.Len(t, positions, 1)
	})

	t.Run("Get positions by party with nil party", func(t *testing.T) {
		positions, err := partyResolver.Positions(ctx, nil)
		assert.Nil(t, positions)
		assert.NotNil(t, err)
		assert.Equal(t, ErrNilParty, err)
	})

	t.Run("Get positions by party with unknown party", func(t *testing.T) {
		badparty := &Party{
			Name: "lolparty",
		}
		positions, err := partyResolver.Positions(ctx, badparty)
		assert.Nil(t, positions)
		assert.NotNil(t, err)
		assert.Equal(t, "party lolparty not found in store", err.Error())
	})

}

func intptr(i int) *int {
	return &i
}

type testCtx struct {
	root             *resolverRoot
	orderstore       storage.OrderStore
	ordersrv         orders.Service
	blockchainClient blockchain.Client
	timesrv          vegatime.Service
	riskstore        storage.RiskStore
	tradestore       storage.TradeStore
	tradesrv         trades.Service
	candlestore      storage.CandleStore
	candlesrv        candles.Service
	marketstore      storage.MarketStore
	marketsrv        markets.Service
	partystore       storage.PartyStore
	partysrv         parties.Service
}

func (tctx *testCtx) Close() {
	tctx.orderstore.Close()
	tctx.riskstore.Close()
	tctx.tradestore.Close()
	tctx.candlestore.Close()
	tctx.marketstore.Close()
	tctx.partystore.Close()
}

func buildResolver(t *testing.T, logger *logging.Logger) *testCtx {
	config, err := internal.NewDefaultConfig(logger, *datadir)
	assert.Nil(t, err)

	// time service
	timesrv := vegatime.NewTimeService(config.Time)

	// order service
	bcmock := &bcmocks.Client{}
	orderstore, err := storage.NewOrderStore(config.Storage)
	assert.Nil(t, err)

	ordersrv, err := orders.NewOrderService(config.Orders, orderstore, timesrv, bcmock)
	assert.Nil(t, err)

	// risk store
	riskstore, err := storage.NewRiskStore(config.Storage)
	assert.Nil(t, err)

	// trade store
	tradestore, err := storage.NewTradeStore(config.Storage)
	assert.Nil(t, err)

	// trade service
	tradesrv, err := trades.NewTradeService(config.Trades, tradestore, riskstore)
	assert.Nil(t, err)

	// candle store
	candlestore, err := storage.NewCandleStore(config.Storage)
	assert.Nil(t, err)

	// candle service
	candlesrv, err := candles.NewCandleService(config.Candles, candlestore)
	assert.Nil(t, err)

	// market store
	marketstore, err := storage.NewMarketStore(config.Storage)
	assert.Nil(t, err)

	marketsrv, err := markets.NewMarketService(config.Markets, marketstore, orderstore)
	assert.Nil(t, err)

	// party store
	partystore, err := storage.NewPartyStore(config.Storage)
	assert.Nil(t, err)

	partysrv, err := parties.NewPartyService(config.Parties, partystore)
	assert.Nil(t, err)

	root := NewResolverRoot(config.API, ordersrv, tradesrv,
		candlesrv, timesrv, marketsrv, partysrv)

	assert.NotNil(t, root)

	// default values

	marketstore.Post(&types.Market{
		Name: "BTC/DEC19",
	})

	partystore.Post(&types.Party{
		Name: "localjoe",
	})

	return &testCtx{
		root:             root,
		orderstore:       orderstore,
		ordersrv:         ordersrv,
		blockchainClient: bcmock,
		timesrv:          timesrv,
		riskstore:        riskstore,
		tradestore:       tradestore,
		tradesrv:         tradesrv,
		candlestore:      candlestore,
		candlesrv:        candlesrv,
		marketstore:      marketstore,
		marketsrv:        marketsrv,
		partystore:       partystore,
		partysrv:         partysrv,
	}
}
