package gql

import (
	"context"
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/api/endpoints/gql/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/monitoring"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewResolverRoot_ConstructAndResolve(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
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
	root := buildTestResolverRoot(t)
	defer root.Finish()
	assert.NotNil(t, root)

	queryResolver := root.Query()
	assert.NotNil(t, queryResolver)

	ctx := context.Background()
	vega, err := queryResolver.Vega(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, vega)
}

func TestNewResolverRoot_VegaResolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()

	notExistsErr := errors.New("market does not exist")
	markets := map[string]*types.Market{
		"BTC/DEC19": &types.Market{
			Name: "BTC/DEC19",
		},
		"ETH/USD18": nil,
	}

	root.market.EXPECT().GetByName(gomock.Any(), gomock.Any()).Times(len(markets)).DoAndReturn(func(_ context.Context, k string) (*types.Market, error) {
		m, ok := markets[k]
		assert.True(t, ok)
		if m == nil {
			return nil, notExistsErr
		}
		return m, nil
	})

	vegaResolver := root.Vega()
	assert.NotNil(t, vegaResolver)

	vega := &Vega{}
	name := "BTC/DEC19"
	vMarkets, err := vegaResolver.Markets(ctx, vega, &name)
	assert.Nil(t, err)
	assert.NotNil(t, vMarkets)
	assert.Len(t, vMarkets, 1)

	name = "ETH/USD18"
	vMarkets, err = vegaResolver.Markets(ctx, vega, &name)
	assert.Error(t, err)
	assert.Nil(t, vMarkets)

	vMarkets, err = vegaResolver.Markets(ctx, vega, nil)
	assert.Error(t, err)
	assert.Nil(t, vMarkets)

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
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()

	notExistsErr := errors.New("market does not exist")
	markets := map[string]*types.Market{
		"BTC/DEC19": &types.Market{
			Name: "BTC/DEC19",
		},
	}
	marketId := "BTC/DEC19"
	market := &Market{
		Name: marketId,
	}

	root.market.EXPECT().GetByName(gomock.Any(), gomock.Any()).Times(len(markets)).DoAndReturn(func(_ context.Context, k string) (*types.Market, error) {
		m, ok := markets[k]
		assert.True(t, ok)
		if m == nil {
			return nil, notExistsErr
		}
		return m, nil
	})
	root.order.EXPECT().GetByMarket(gomock.Any(), marketId, gomock.Any()).Times(1).Return([]*types.Order{
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
	}, nil)

	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	orders, err := marketResolver.Orders(ctx, market, nil, nil, nil, nil)
	assert.NotNil(t, orders)
	assert.Nil(t, err)
	assert.Len(t, orders, 2)
}

type testResolver struct {
	*resolverRoot
	log    *logging.Logger
	ctrl   *gomock.Controller
	order  *mocks.MockOrderService
	trade  *mocks.MockTradeService
	candle *mocks.MockCandleService
	market *mocks.MockMarketService
}

func buildTestResolverRoot(t *testing.T) *testResolver {
	ctrl := gomock.NewController(t)
	log := logging.NewLoggerFromEnv("dev")
	conf := api.NewDefaultConfig(log)
	order := mocks.NewMockOrderService(ctrl)
	trade := mocks.NewMockTradeService(ctrl)
	candle := mocks.NewMockCandleService(ctrl)
	market := mocks.NewMockMarketService(ctrl)
	statusChecker := &monitoring.Status{}
	resolver := NewResolverRoot(
		conf,
		order,
		trade,
		candle,
		market,
		statusChecker,
	)
	return &testResolver{
		resolverRoot: resolver,
		log:          log,
		ctrl:         ctrl,
		order:        order,
		trade:        trade,
		candle:       candle,
		market:       market,
	}
}

func (t *testResolver) Finish() {
	t.log.Sync()
	t.ctrl.Finish()
}
