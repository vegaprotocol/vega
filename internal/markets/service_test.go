package markets

import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage/mocks"
	"code.vegaprotocol.io/vega/internal/storage/newmocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
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

func TestMarketObserveDepth(t *testing.T) {
	t.Run("Observe market depth, success", testMarketObserveDepthSuccess)
}

func testMarketObserveDepthSuccess(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	mockCtrl := gomock.NewController(t)
	marketStore := newmocks.NewMockMarketStore(mockCtrl)
	orderStore := newmocks.NewMockOrderStore(mockCtrl)
	marketArg := "TSTmarket"
	// empty slice on the internal channel
	orders := []types.Order{}
	// return value of GetMarketDepth call
	depth := types.MarketDepth{
		Name: marketArg,
	}
	// ensure unsubscribe was handled properly
	wg := sync.WaitGroup{}
	wg.Add(1)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	conf := NewDefaultConfig(logger)
	marketService, err := NewMarketService(conf, marketStore, orderStore)
	assert.Nil(t, err)
	// set up calls

	// perform this write in a routine, somehow this doesn't work when we use an anonymous func in the Do argument
	writeToChannel := func(ch chan<- []types.Order) {
		ch <- orders
	}
	orderStore.EXPECT().Subscribe(gomock.Any()).Times(1).Return(uint64(1)).Do(func(ch chan<- []types.Order) {
		go writeToChannel(ch)
	})

	orderStore.EXPECT().GetMarketDepth(gomock.Any(), marketArg).Times(1).Return(&depth, nil)
	// waitgroup here ensures that unsubscribe was indeed called
	orderStore.EXPECT().Unsubscribe(uint64(1)).Times(1).Return(nil).Do(func(_ uint64) {
		wg.Done()
	})

	depthCh, ref := marketService.ObserveDepth(ctx, 0, marketArg)
	assert.Equal(t, uint64(1), ref) // should be returned straight from the orderStore mock
	// whatever was in the channel, we're expecting that to be accessible here, too
	chDepth := <-depthCh
	// cancel context here, so we can check the unsubscribe call went through as expected
	cfunc()
	assert.Equal(t, depth, *chDepth)
	wg.Wait() // wait for unsubscribe call
	// end mocks
	mockCtrl.Finish()
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
