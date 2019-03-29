package trades

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/trades/newmocks"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/logging"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	Service
	ctx   context.Context
	cfunc context.CancelFunc
	log   *logging.Logger
	ctrl  *gomock.Controller
	trade *newmocks.MockTradeStore
	risk  *newmocks.MockRiskStore
}

func getTestService(t *testing.T) *testService {
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	trade := newmocks.NewMockTradeStore(ctrl)
	risk := newmocks.NewMockRiskStore(ctrl)
	log := logging.NewLoggerFromEnv("dev")
	svc, err := NewTradeService(
		NewDefaultConfig(log),
		trade,
		risk,
	)
	assert.NoError(t, err)
	return &testService{
		Service: svc,
		ctx:     ctx,
		cfunc:   cfunc,
		log:     log,
		ctrl:    ctrl,
		trade:   trade,
		risk:    risk,
	}
}

// storageConfig specifies that the badger files are kept in a different
// directory when the candle service tests run. This is useful as when
// all the unit tests are run for the project they can be run in parallel.
func storageConfig(t *testing.T) *storage.Config {
	storeConfig, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}
	storeConfig.LogPositionStoreDebug = false

	return storeConfig
}

func TestGetByMarket(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	market := "BTC/DEC19"
	invalid := "LTC/DEC19"
	expErr := errors.New("phobos communications link interrupted")
	expect := []*types.Trade{
		{Id: "A", Market: market, Price: 100},
		{Id: "B", Market: market, Price: 200},
		{Id: "C", Market: market, Price: 300},
	}

	svc.trade.EXPECT().GetByMarket(svc.ctx, market, nil).Times(1).Return(expect, nil)
	svc.trade.EXPECT().GetByMarket(svc.ctx, invalid, nil).Times(1).Return(nil, expErr)

	success, noErr := svc.GetByMarket(svc.ctx, market, nil)
	assert.NoError(t, noErr)
	assert.Equal(t, expect, success)

	fail, err := svc.GetByMarket(svc.ctx, invalid, nil)
	assert.Nil(t, fail)
	assert.Equal(t, expErr, err)
}

func TestTradeService_GetByParty(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	expErr := errors.New("phobos communications link interrupted")

	partyA := "ramsey"
	partyB := "barney"
	invalid := "chris"

	expect := map[string][]*types.Trade{
		partyA: []*types.Trade{
			{Id: "A", Buyer: partyA, Seller: partyB, Price: 100},
			{Id: "B", Buyer: partyB, Seller: partyA, Price: 200},
		},
		partyB: []*types.Trade{
			{Id: "C", Buyer: partyB, Seller: partyA, Price: 100},
			{Id: "D", Buyer: partyA, Seller: partyB, Price: 200},
		},
		invalid: nil,
	}
	svc.trade.EXPECT().GetByParty(svc.ctx, gomock.Any(), nil).Times(len(expect)).DoAndReturn(func(_ context.Context, party string, _ *filtering.TradeQueryFilters) ([]*types.Trade, error) {
		trades, ok := expect[party]
		assert.True(t, ok)
		if trades == nil {
			return nil, expErr
		}
		return trades, nil
	})

	for party, exp := range expect {
		trades, err := svc.GetByParty(svc.ctx, party, nil)
		if exp == nil {
			assert.Nil(t, trades)
			assert.Equal(t, expErr, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, exp, trades)
		}
	}
}

func (t *testService) Finish() {
	t.log.Sync()
	t.cfunc()
	t.ctrl.Finish()
}

//func TestTradeService_GetAllTradesForOrderOnMarket(t *testing.T) {
//	var market = ServiceTestMarket
//	var orderId = "12345"
//
//	var ctx = context.Background()
//	var tradeStore = mocks.TradeStore{}
//	var tradeService = NewTradeService()
//
//	vega := &core.Vega{}
//	tradeService.Init(vega, &tradeStore)
//
//	tradeStore.On("GetByOrderId", market, orderId, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Trade{
//		{Trade: types.Trade{Id: "A", Market: market, Price: 1}, OrderId: orderId},
//		{Trade: types.Trade{Id: "B", Market: market, Price: 2}, OrderId: orderId},
//		{Trade: types.Trade{Id: "C", Market: market, Price: 3}, OrderId: orderId},
//		{Trade: types.Trade{Id: "D", Market: market, Price: 4}, OrderId: orderId},
//		{Trade: types.Trade{Id: "E", Market: market, Price: 5}, OrderId: orderId},
//		{Trade: types.Trade{Id: "F", Market: market, Price: 6}, OrderId: orderId},
//	}, nil).Once()
//
//	var tradeSet, err = tradeService.GetTradesForOrder(ctx, market, orderId, datastore.GetParamsLimitDefault)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, tradeSet)
//	assert.Equal(t, 6, len(tradeSet))
//	tradeStore.AssertExpectations(t)
//}
//
//func TestOrderService_GetOrderById(t *testing.T) {
//	var market = ServiceTestMarket
//	var orderId = "12345"
//
//	var ctx = context.Background()
//	var orderStore = mocks.OrderStore{}
//	var orderService = NewOrderService()
//
//	vega := &core.Vega{}
//	orderService.Init(vega, &orderStore)
//
//	orderStore.On("Get", market, orderId).Return(datastore.Order{
//		Order: types.Order{Id: orderId, Market: market},
//	}, nil)
//
//	var order, err = orderService.GetById(ctx, market, orderId)
//
//	assert.Nil(t, err)
//	assert.NotNil(t, order)
//	assert.Equal(t, orderId, order.Id)
//	orderStore.AssertExpectations(t)
//
//}
