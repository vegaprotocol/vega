package gql

import (
	"context"
	"testing"

	types "vega/proto"

	"vega/api"
	"vega/internal/filtering"
	"vega/internal/logging"

	mockCandle "vega/internal/candles/mocks"
	mockMarket "vega/internal/markets/mocks"
	mockOrder "vega/internal/orders/mocks"
	mockParty "vega/internal/parties/mocks"
	mockTrade "vega/internal/trades/mocks"
	mockTime "vega/internal/vegatime/mocks"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewResolverRoot_ConstructAndResolve(t *testing.T) {
	root := buildTestResolverRoot()
	assert.NotNil(t, root)

	partyResolver := root.Party()
	assert.NotNil(t, partyResolver)

	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	depthResolver := root.MarketDepth()
	assert.NotNil(t, depthResolver)

	candleResolver := root.Candle()
	assert.NotNil(t, candleResolver)

	orderResolver := root.Order()
	assert.NotNil(t, orderResolver)

	tradeResolver := root.Trade()
	assert.NotNil(t, tradeResolver)

	vegaResolver := root.Vega()
	assert.NotNil(t, vegaResolver)

	priceLevelResolver := root.PriceLevel()
	assert.NotNil(t, priceLevelResolver)

	mutationResolver := root.Mutation()
	assert.NotNil(t, mutationResolver)

	positionResolver := root.Position()
	assert.NotNil(t, positionResolver)

	queryResolver := root.Query()
	assert.NotNil(t, queryResolver)

	subsResolver := root.Subscription()
	assert.NotNil(t, subsResolver)
}

func TestNewResolverRoot_QueryResolver(t *testing.T) {
	root := buildTestResolverRoot()
	assert.NotNil(t, root)

	queryResolver := root.Query()
	assert.NotNil(t, queryResolver)

	ctx := context.Background()
	vega, err := queryResolver.Vega(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, vega)
}

func TestNewResolverRoot_VegaResolver(t *testing.T) {

	ctx := context.Background()

	mockTradeService := &mockTrade.Service{}
	mockOrderService := &mockOrder.Service{}
	mockCandleService := &mockCandle.Service{}
	mockMarketService := &mockMarket.Service{}
	mockPartyService := &mockParty.Service{}
	mockTimeService := &mockTime.Service{}

	mockMarketService.On("GetByName", ctx, "BTC/DEC19").Return(&types.Market{Name:"BTC/DEC19"},
		nil).On("GetByName", ctx, "ETH/USD18").Return(nil,
			errors.New("market does not exist")).On("GetByName",
				ctx, "errorMarket").Return(nil, errors.New("market does not exist"))

	mockOrderService.On("GetMarkets", ctx).Return(
		[]string{"BTC/DEC19"}, nil,
	).Times(3)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	config := api.NewDefaultConfig(logger)
	root := NewResolverRoot(config, mockOrderService, mockTradeService,
		mockCandleService, mockTimeService, mockMarketService, mockPartyService)

	assert.NotNil(t, root)
	vegaResolver := root.Vega()
	assert.NotNil(t, vegaResolver)

	vega := &Vega{}
	name := "BTC/DEC19"
	markets, err := vegaResolver.Markets(ctx, vega, &name)
	assert.Nil(t, err)
	assert.NotNil(t, markets)
	assert.Len(t, markets, 1)

	name = "ETH/USD18"
	markets, err = vegaResolver.Markets(ctx, vega, &name)
	assert.Error(t, err)
	assert.Nil(t, markets)

	markets, err = vegaResolver.Markets(ctx, vega, nil)
	assert.Error(t, err)
	assert.Nil(t, markets)

	mockOrderService.On("GetMarkets", ctx).Return(
		[]string{}, errors.New("proton drive not ready"),
	).Once()

	name = "errorMarket"
	markets, err = vegaResolver.Markets(ctx, vega, &name)
	assert.NotNil(t, err)

	name = "barney"
	parties, err := vegaResolver.Parties(ctx, vega, &name)
	assert.Nil(t, err)
	assert.NotNil(t, parties)
	assert.Len(t, parties, 1)

	parties, err = vegaResolver.Parties(ctx, vega, nil)
	assert.Error(t, err)
	assert.Nil(t, parties)
}

func TestNewResolverRoot_MarketResolver(t *testing.T) {
	ctx := context.Background()

	mockTradeService := &mockTrade.Service{}
	mockOrderService := &mockOrder.Service{}
	mockCandleService := &mockCandle.Service{}
	mockMarketService := &mockMarket.Service{}
	mockPartyService := &mockParty.Service{}
	mockTimeService := &mockTime.Service{}


	mockMarketService.On("GetByName", ctx, "BTC/DEC19").Return(&types.Market{Name:"BTC/DEC19"},
		nil).On("GetByName", ctx, "errorMarket").Return(nil,
			errors.New("market does not exist"))


	mockOrderService.On("GetMarkets", ctx).Return(
		[]string{"testMarket", "BTC/DEC19"}, nil,
	).Times(3)

	depth := types.MarketDepth{
		Name: "BTC/DEC19",
	}
	mockOrderService.On("GetMarketDepth", ctx, "BTC/DEC19").Return(
		depth, nil,
	).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()
	config := api.NewDefaultConfig(logger)

	root := NewResolverRoot(config, mockOrderService, mockTradeService,
		mockCandleService, mockTimeService, mockMarketService, mockPartyService)

	assert.NotNil(t, root)
	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	marketId := "BTC/DEC19"
	market := &Market{
		Name: marketId,
	}

	// DEPTH
	//depth, err := marketResolver.Depth(ctx, market)
	//assert.Nil(t, err)
	//assert.NotNil(t, depth)
	//assert.Equal(t, marketId, depth.Name)
	//
	//mockOrderService.On("GetMarketDepth", ctx, btcDec18).Return(
	//	nil, errors.New("phobos transport system overload"),
	//).Once()
	//
	//depth, err = marketResolver.Depth(ctx, market)
	//assert.Error(t, err)

	// ORDERS

	mockOrderService.On("GetByMarket", ctx, marketId, &filtering.OrderQueryFilters{}).Return(
		[]*types.Order{
			{
				Id:        "order-id-1",
				Price:     1000,
				Timestamp: 1,
			},
			{
				Id:        "order-id-2",
				Price:     2000,
				Timestamp: 2,
			},
		}, nil,
	).Once()

	orders, err := marketResolver.Orders(ctx, market, nil, nil, nil, nil)
	assert.NotNil(t, orders)
	assert.Nil(t, err)
	assert.Len(t, orders, 2)
}

func buildTestResolverRoot() *resolverRoot {

	mockTradeService := &mockTrade.Service{}
	mockOrderService := &mockOrder.Service{}
	mockCandleService := &mockCandle.Service{}
	mockMarketService := &mockMarket.Service{}
	mockPartyService := &mockParty.Service{}
	mockTimeService := &mockTime.Service{}

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()
	config := api.NewDefaultConfig(logger)

	return NewResolverRoot(config, mockOrderService, mockTradeService,
		mockCandleService, mockTimeService, mockMarketService, mockPartyService)
}
