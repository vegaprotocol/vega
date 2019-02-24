package markets

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"vega/internal/storage/mocks"
	types "vega/proto"
	"github.com/pkg/errors"
	"vega/internal/logging"
)

func TestMarketService_NewService(t *testing.T) {
	orderStore := &mocks.OrderStore{}
	marketStore := &mocks.MarketStore{}

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)
}

func TestMarketService_CreateMarket(t *testing.T) {
	orderStore := &mocks.OrderStore{}
	marketStore := &mocks.MarketStore{}
	market := &types.Market{Name: "BTC/DEC19"}
	marketStore.On("Post", market).Return(nil)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewConfig(logger)
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
		{Name: "BTC/DEC19"},
		{Name: "ETH/JUN19"},
		{Name: "LTC/JAN20"},
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	markets, err := marketService.GetAll(context.Background())
	assert.Nil(t, err)
	assert.Len(t, markets, 3)
	assert.Equal(t, "BTC/DEC19", markets[0].Name)
	assert.Equal(t, "ETH/JUN19", markets[1].Name)
	assert.Equal(t, "LTC/JAN20", markets[2].Name)
}

func TestMarketService_GetByName(t *testing.T) {
	orderStore := &mocks.OrderStore{}
	marketStore := &mocks.MarketStore{}
	marketStore.On("GetByName", "BTC/DEC19").Return(&types.Market{
		Name: "BTC/DEC19",
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	market, err := marketService.GetByName(context.Background(), "BTC/DEC19")
	assert.Nil(t, err)
	assert.Equal(t, "BTC/DEC19", market.Name)
}

func TestMarketService_GetDepth(t *testing.T) {
	market := &types.Market{Name: "BTC/DEC19"}
	orderStore := &mocks.OrderStore{}
	orderStore.On("GetMarketDepth", market.Name).Return(&types.MarketDepth{
		Name: market.Name,
	}, nil)
	marketStore := &mocks.MarketStore{}
	marketStore.On("GetByName", market.Name).Return(&types.Market{
		Name: market.Name,
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	depth, err := marketService.GetDepth(context.Background(), market.Name)
	assert.Nil(t, err)
	assert.NotNil(t, depth)
}


func TestMarketService_GetDepthNonExistentMarket(t *testing.T) {
	market := &types.Market{Name: "BTC/DEC18"}
	orderStore := &mocks.OrderStore{}
	orderStore.On("GetMarketDepth", "BTC/DEC18").Return(nil,
		errors.New("market does not exist"))

	marketStore := &mocks.MarketStore{}
	marketStore.On("GetByName", market.Name).Return(nil,
		errors.New("market does not exist")).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	marketConfig := NewConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	depth, err := marketService.GetDepth(context.Background(), market.Name)
	assert.NotNil(t, err)
	assert.Nil(t, depth)
}

//func TestMarketService_ObserveMarkets(t *testing.T) {
//	// todo(cdm) observing markets service test
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
//	// todo(cdm) observing market depth service test
//	//ctx := context.Background()
//	//context.WithCancel(ctx, func())
//	//
//	//marketService.ObserveDepth(context.Background(), )
//	//
//
//	assert.True(t, false)
//}
