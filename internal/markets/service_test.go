package markets

import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets/newmocks"
	"code.vegaprotocol.io/vega/internal/storage/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	Service
	ctx    context.Context
	cfunc  context.CancelFunc
	log    *logging.Logger
	ctrl   *gomock.Controller
	order  *newmocks.MockOrderStore
	market *newmocks.MockMarketStore
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	order := newmocks.NewMockOrderStore(ctrl)
	market := newmocks.NewMockMarketStore(ctrl)
	log := logging.NewLoggerFromEnv("dev")
	ctx, cfunc := context.WithCancel(context.Background())
	svc, err := NewMarketService(
		NewDefaultConfig(log),
		market,
		order,
	)
	assert.NoError(t, err)
	return &testService{
		Service: svc,
		ctx:     ctx,
		cfunc:   cfunc,
		log:     log,
		ctrl:    ctrl,
		order:   order,
		market:  market,
	}
}

func TestMarketService_CreateMarket(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	market := &types.Market{Name: "BTC/DEC19"}
	svc.market.EXPECT().Post(market).Times(1).Return(nil)

	assert.NoError(t, svc.CreateMarket(svc.ctx, market))
}

func TestMarketService_GetAll(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	markets := []*types.Market{
		{Name: "BTC/DEC19"},
		{Name: "ETH/JUN19"},
		{Name: "LTC/JAN20"},
	}
	svc.market.EXPECT().GetAll().Times(1).Return(markets, nil)

	get, err := svc.GetAll(svc.ctx)
	assert.NoError(t, err)
	assert.Equal(t, markets, get)
}

func TestMarketService_GetByName(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	markets := map[string]*types.Market{
		"BTC/DEC19": &types.Market{Name: "BTC/DEC19"},
		"ETH/JUN19": &types.Market{Name: "ETH/JUN19"},
		"LTC/JAN20": nil,
	}
	notFoundErr := errors.New("market not found")
	svc.market.EXPECT().GetByName(gomock.Any()).Times(len(markets)).DoAndReturn(func(k string) (*types.Market, error) {
		m, ok := markets[k]
		assert.True(t, ok)
		if m == nil {
			return nil, notFoundErr
		}
		return m, nil
	})
	for k, exp := range markets {
		market, err := svc.GetByName(svc.ctx, k)
		if exp != nil {
			assert.Equal(t, exp, market)
			assert.NoError(t, err)
		} else {
			assert.Nil(t, market)
			assert.Equal(t, notFoundErr, err)
		}
	}
}

func TestMarketService_GetDepth(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	market := &types.Market{Name: "BTC/DEC19"}
	depth := &types.MarketDepth{
		Name: market.Name,
	}

	svc.market.EXPECT().GetByName(market.Name).Times(1).Return(market, nil)
	svc.order.EXPECT().GetMarketDepth(svc.ctx, market.Name).Times(1).Return(depth, nil)

	got, err := svc.GetDepth(svc.ctx, market.Name)
	assert.NoError(t, err)
	assert.Equal(t, depth, got)
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

	marketConfig := NewDefaultConfig(logger)
	marketService, err := NewMarketService(marketConfig, marketStore, orderStore)
	assert.NotNil(t, marketService)
	assert.Nil(t, err)

	depth, err := marketService.GetDepth(context.Background(), market.Name)
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

func (t *testService) Finish() {
	t.cfunc()
	t.log.Sync()
	t.ctrl.Finish()
}
