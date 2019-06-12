package markets_test

import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/markets/mocks"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*markets.Svc
	ctx    context.Context
	cfunc  context.CancelFunc
	log    *logging.Logger
	ctrl   *gomock.Controller
	order  *mocks.MockOrderStore
	market *mocks.MockMarketStore
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	order := mocks.NewMockOrderStore(ctrl)
	market := mocks.NewMockMarketStore(ctrl)
	log := logging.NewTestLogger()
	ctx, cfunc := context.WithCancel(context.Background())
	svc, err := markets.NewService(
		log,
		storcfg.NewDefaultMarketsConfig("/tmp"),
		market,
		order,
	)
	assert.NoError(t, err)
	return &testService{
		Svc:    svc,
		ctx:    ctx,
		cfunc:  cfunc,
		log:    log,
		ctrl:   ctrl,
		order:  order,
		market: market,
	}
}

func TestMarketService_CreateMarket(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	market := &types.Market{Id: "BTC/DEC19"}
	svc.market.EXPECT().Post(market).Times(1).Return(nil)

	assert.NoError(t, svc.CreateMarket(svc.ctx, market))
}

func TestMarketService_GetAll(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	markets := []*types.Market{
		{Id: "BTC/DEC19"},
		{Id: "ETH/JUN19"},
		{Id: "LTC/JAN20"},
	}
	svc.market.EXPECT().GetAll().Times(1).Return(markets, nil)

	get, err := svc.GetAll(svc.ctx)
	assert.NoError(t, err)
	assert.Equal(t, markets, get)
}

func TestMarketService_GetByID(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	markets := map[string]*types.Market{
		"BTC/DEC19": &types.Market{Id: "BTC/DEC19"},
		"ETH/JUN19": &types.Market{Id: "ETH/JUN19"},
		"LTC/JAN20": nil,
	}
	notFoundErr := errors.New("market not found")
	svc.market.EXPECT().GetByID(gomock.Any()).Times(len(markets)).DoAndReturn(func(k string) (*types.Market, error) {
		m, ok := markets[k]
		assert.True(t, ok)
		if m == nil {
			return nil, notFoundErr
		}
		return m, nil
	})
	for k, exp := range markets {
		market, err := svc.GetByID(svc.ctx, k)
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
	market := &types.Market{Id: "BTC/DEC19"}
	depth := &types.MarketDepth{
		MarketID: market.Id,
	}

	svc.market.EXPECT().GetByID(market.Id).Times(1).Return(market, nil)
	svc.order.EXPECT().GetMarketDepth(svc.ctx, market.Id).Times(1).Return(depth, nil)

	got, err := svc.GetDepth(svc.ctx, market.Id)
	assert.NoError(t, err)
	assert.Equal(t, depth, got)
}

func TestMarketService_GetDepthNonExistentMarket(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	market := &types.Market{Id: "BTC/DEC18"}
	notFoundErr := errors.New("market does not exist")

	svc.market.EXPECT().GetByID(market.Id).Times(1).Return(nil, notFoundErr)

	depth, err := svc.GetDepth(svc.ctx, market.Id)
	assert.Equal(t, notFoundErr, err)
	assert.Nil(t, depth)
}

func TestMarketObserveDepth(t *testing.T) {
	t.Run("Observe market depth, success", testMarketObserveDepthSuccess)
}

func testMarketObserveDepthSuccess(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	marketArg := "TSTmarket"
	// empty slice on the internal channel
	orders := []types.Order{}
	// return value of GetMarketDepth call
	depth := types.MarketDepth{
		MarketID: marketArg,
	}
	// ensure unsubscribe was handled properly
	wg := sync.WaitGroup{}
	wg.Add(1)

	// perform this write in a routine, somehow this doesn't work when we use an anonymous func in the Do argument
	writeToChannel := func(ch chan<- []types.Order) {
		ch <- orders
	}
	svc.order.EXPECT().Subscribe(gomock.Any()).Times(1).Return(uint64(1)).Do(func(ch chan<- []types.Order) {
		go writeToChannel(ch)
	})

	svc.order.EXPECT().GetMarketDepth(gomock.Any(), marketArg).Times(1).Return(&depth, nil)
	// waitgroup here ensures that unsubscribe was indeed called
	svc.order.EXPECT().Unsubscribe(uint64(1)).Times(1).Return(nil).Do(func(_ uint64) {
		wg.Done()
	})

	depthCh, ref := svc.ObserveDepth(svc.ctx, 0, marketArg)
	assert.Equal(t, uint64(1), ref) // should be returned straight from the orderStore mock
	// whatever was in the channel, we're expecting that to be accessible here, too
	chDepth := <-depthCh
	// cancel context here, so we can check the unsubscribe call went through as expected
	svc.cfunc()
	assert.Equal(t, depth, *chDepth)
	wg.Wait() // wait for unsubscribe call
}

func (t *testService) Finish() {
	t.cfunc()
	t.log.Sync()
	t.ctrl.Finish()
}
