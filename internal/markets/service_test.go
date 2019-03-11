package markets

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestMarketService_NewService(t *testing.T) {
	orderStore := &mocks.OrderStore{}
	marketStore := &mocks.MarketStore{}

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewDefaultConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)
}

func TestMarketService_CreateMarket(t *testing.T) {
	orderStore := &mocks.OrderStore{}
	marketStore := &mocks.MarketStore{}
	market := &types.Market{Id: "BTC/DEC19"}
	marketStore.On("Post", market).Return(nil)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewDefaultConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	err = marketService.CreateMarket(context.Background(), market)
	assert.Nil(t, err)
}

func TestMarketService_GetAll(t *testing.T) {
	orderStore := &mocks.OrderStore{}
	marketStore := &mocks.MarketStore{}
	marketStore.On("GetAll").Return([]*types.Market{
		{Id: "BTC/DEC19"},
		{Id: "ETH/JUN19"},
		{Id: "LTC/JAN20"},
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewDefaultConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	markets, err := marketService.GetAll(context.Background())
	assert.Nil(t, err)
	assert.Len(t, markets, 3)
	assert.Equal(t, "BTC/DEC19", markets[0].Id)
	assert.Equal(t, "ETH/JUN19", markets[1].Id)
	assert.Equal(t, "LTC/JAN20", markets[2].Id)
}

func TestMarketService_GetByName(t *testing.T) {
	orderStore := &mocks.OrderStore{}
	marketStore := &mocks.MarketStore{}
	marketStore.On("GetByID", "BTC/DEC19").Return(&types.Market{
		Id: "BTC/DEC19",
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewDefaultConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	market, err := marketService.GetByID(context.Background(), "BTC/DEC19")
	assert.Nil(t, err)
	assert.Equal(t, "BTC/DEC19", market.Id)
}

func TestMarketService_GetDepth(t *testing.T) {
	market := &types.Market{Id: "BTC/DEC19"}
	orderStore := &mocks.OrderStore{}

	orderStore.On("GetMarketDepth", context.Background(), market.Id).Return(&types.MarketDepth{
		Name: market.Id,
	}, nil)
	marketStore := &mocks.MarketStore{}
	marketStore.On("GetByID", market.Id).Return(&types.Market{
		Id: market.Id,
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewDefaultConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	depth, err := marketService.GetDepth(context.Background(), market.Id)
	assert.Nil(t, err)
	assert.NotNil(t, depth)
}

func TestMarketService_GetDepthNonExistentMarket(t *testing.T) {
	market := &types.Market{Id: "BTC/DEC18"}
	orderStore := &mocks.OrderStore{}
	orderStore.On("GetMarketDepth", "BTC/DEC18").Return(nil,
		errors.New("market does not exist"))

	marketStore := &mocks.MarketStore{}
	marketStore.On("GetByID", market.Id).Return(nil,
		errors.New("market does not exist")).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewDefaultConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	depth, err := marketService.GetDepth(context.Background(), market.Id)
	assert.NotNil(t, err)
	assert.Nil(t, depth)
}

//func TestMarketService_ObserveMarkets(t *testing.T) {
//	// todo: observing markets service test (gitlab.com/vega-protocol/trading-core/issues/166)
//	assert.True(t, false)
//}

//func TestMarketService_ObserveDepth(t *testing.T) {
//
//	orderStore := &mocks.OrderStore{}
//	marketStore := &mocks.MarketStore{}
//
//	logger := logging.NewLoggerFromEnv("dev")
//	defer logger.Sync()
//
//	marketConfig := NewConfig(logger)
//	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
//	assert.NotNil(t, marketService)
//	assert.Nil(t, err)
//
//	// todo: observing market depth service test (gitlab.com/vega-protocol/trading-core/issues/166)
//	//ctx := context.Background()
//	//context.WithCancel(ctx, func())
//	//
//	//marketService.ObserveDepth(context.Background(), )
//	//
//
//	assert.True(t, false)
//}
