package gql

import (
	"testing"
	"vega/api/mocks"
	"vega/api"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"context"
	"github.com/pkg/errors"
	"vega/filters"
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
	mockTradeService := &mocks.TradeService{}
	mockOrderService := &mocks.OrderService{}
	mockCandleService := &mocks.CandleService{}

	mockOrderService.On("GetMarkets", ctx).Return(
		[]string {"BTC/DEC18"}, nil,
	).Times(3)

	var tradeService api.TradeService
	var orderService api.OrderService
	var candleService api.CandleService
	tradeService = mockTradeService
	orderService = mockOrderService
	candleService = mockCandleService

	root := NewResolverRoot(orderService, tradeService, candleService)
	assert.NotNil(t, root)
	vegaResolver := root.Vega()
	assert.NotNil(t, vegaResolver)

	vega := &Vega{}
	name := "BTC/DEC18"
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
		[]string {}, errors.New("proton drive not ready"),
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
	mockTradeService := &mocks.TradeService{}
	mockOrderService := &mocks.OrderService{}
	mockCandleService := &mocks.CandleService{}

	btcDec18 := "BTC/DEC18"
	mockOrderService.On("GetMarkets", ctx).Return(
		[]string{"testMarket", btcDec18}, nil,
	).Times(3)

	depth := msg.MarketDepth{
		Name: btcDec18,
	}
	mockOrderService.On("GetMarketDepth", ctx, btcDec18).Return(
		depth, nil,
	).Once()

	var tradeService api.TradeService
	var orderService api.OrderService
	var candleService api.CandleService
	tradeService = mockTradeService
	orderService = mockOrderService
	candleService = mockCandleService
	
	root := NewResolverRoot(orderService, tradeService, candleService)
	assert.NotNil(t, root)
	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	market := &Market{
		Name: btcDec18,
	}

	// DEPTH

	depth, err := marketResolver.Depth(ctx, market)
	assert.Nil(t, err)
	assert.NotNil(t, depth)
	assert.Equal(t, btcDec18, depth.Name)

	//mockOrderService.On("GetMarketDepth", ctx, btcDec18).Return(
	//	nil, errors.New("phobos transport system overload"),
	//).Once()
	//
	//depth, err = marketResolver.Depth(ctx, market)
	//assert.Error(t, err)

	// ORDERS

	mockOrderService.On("GetByMarket", ctx, btcDec18, &filters.OrderQueryFilters{}).Return(
		[]*msg.Order{
			&msg.Order{
				Id: "order-id-1",
				Price: 1000,
				Timestamp: 1,
			},
			&msg.Order{
				Id: "order-id-2",
				Price: 2000,
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
	var tradeService api.TradeService
	var orderService api.OrderService
	var candleService api.CandleService
	tradeService = &mocks.TradeService{}
	orderService = &mocks.OrderService{}
	candleService = &mocks.CandleService{}
	return NewResolverRoot(orderService, tradeService, candleService)
}