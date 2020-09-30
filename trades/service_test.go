package trades_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/trades/mocks"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*trades.Svc
	ctx   context.Context
	cfunc context.CancelFunc
	log   *logging.Logger
	ctrl  *gomock.Controller
	trade *mocks.MockTradeStore
	pos   *mocks.MockPositionsPlugin
}

func getTestService(t *testing.T) *testService {
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	trade := mocks.NewMockTradeStore(ctrl)
	pos := mocks.NewMockPositionsPlugin(ctrl)
	log := logging.NewTestLogger()
	svc, err := trades.NewService(
		log,
		trades.NewDefaultConfig(),
		trade,
		pos,
	)
	assert.NoError(t, err)
	return &testService{
		Svc:   svc,
		ctx:   ctx,
		cfunc: cfunc,
		log:   log,
		ctrl:  ctrl,
		trade: trade,
	}
}

func TestGetByMarket(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	cfg := svc.Config
	cfg.Level.Level = logging.DebugLevel
	svc.ReloadConf(cfg)
	cfg.Level.Level = logging.InfoLevel
	svc.ReloadConf(cfg)

	market := "BTC/DEC19"
	invalid := "LTC/DEC19"
	expErr := errors.New("phobos communications link interrupted")
	expect := []*types.Trade{
		{Type: types.Trade_TYPE_DEFAULT, Id: "A", MarketID: market, Price: 100},
		{Type: types.Trade_TYPE_DEFAULT, Id: "B", MarketID: market, Price: 200},
		{Type: types.Trade_TYPE_DEFAULT, Id: "C", MarketID: market, Price: 300},
	}

	ui0, ui1, uiDefault := uint64(0), uint64(1), uint64(svc.Config.PageSizeDefault)
	svc.trade.EXPECT().GetByMarket(svc.ctx, market, ui0, uiDefault, false).Times(1).Return(expect, nil)
	svc.trade.EXPECT().GetByMarket(svc.ctx, invalid, ui1, uiDefault, false).Times(1).Return(nil, expErr)

	success, noErr := svc.GetByMarket(svc.ctx, market, 0, 0, false)
	assert.NoError(t, noErr)
	assert.Equal(t, expect, success)

	fail, err := svc.GetByMarket(svc.ctx, invalid, 1, 0, false)
	assert.Nil(t, fail)
	assert.Equal(t, expErr, err)

	_, err = svc.GetByMarket(svc.ctx, market, 0, uint64(svc.Config.PageSizeMaximum+1), false)
	assert.True(t, strings.Contains(err.Error(), "invalid pagination limit"))
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
			{Type: types.Trade_TYPE_DEFAULT, Id: "A", Buyer: partyA, Seller: partyB, Price: 100},
			{Type: types.Trade_TYPE_DEFAULT, Id: "B", Buyer: partyB, Seller: partyA, Price: 200},
		},
		partyB: []*types.Trade{
			{Type: types.Trade_TYPE_DEFAULT, Id: "C", Buyer: partyB, Seller: partyA, Price: 100},
			{Type: types.Trade_TYPE_DEFAULT, Id: "D", Buyer: partyA, Seller: partyB, Price: 200},
		},
		invalid: nil,
	}
	ui0, uiDefault := uint64(0), uint64(svc.Config.PageSizeDefault)
	svc.trade.EXPECT().GetByParty(svc.ctx, gomock.Any(), ui0, uiDefault, false, nil).Times(len(expect)).DoAndReturn(func(_ context.Context, party string, _ uint64, _ uint64, _ bool, _ *string) ([]*types.Trade, error) {
		trades, ok := expect[party]
		assert.True(t, ok)
		if trades == nil {
			return nil, expErr
		}
		return trades, nil
	})

	for party, exp := range expect {
		trades, err := svc.GetByParty(svc.ctx, party, 0, 0, false, nil)
		if exp == nil {
			assert.Nil(t, trades)
			assert.Equal(t, expErr, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, exp, trades)
		}
	}

	_, err := svc.GetByParty(svc.ctx, partyA, 0, uint64(svc.Config.PageSizeMaximum+1), false, nil)
	assert.True(t, strings.Contains(err.Error(), "invalid pagination limit"))
}

func TestObserveTrades(t *testing.T) {
	t.Run("Observe trades - no filters, successfully push to channel", testObserveTradesSuccess)
	t.Run("Observe trades - no filters, no write to channel (retry path)", testObserveTradesNoWrite)
	t.Run("Observe trades - filter, partial results returned", testObserveTradesFilterSuccess)
	t.Run("Observe trades - filters, no results returned", testObserveTradesFilterNone)
}

func testObserveTradesSuccess(t *testing.T) {
	ref := uint64(1)
	market := "BTC/DEC19"
	buyer, seller := "buyerID", "sellerID"
	trades := []types.Trade{
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade1",
			MarketID: market,
			Price:    1000,
			Size:     1,
			Buyer:    buyer,
			Seller:   seller,
		},
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade2",
			MarketID: market,
			Price:    1200,
			Size:     2,
			Buyer:    buyer,
			Seller:   seller,
		},
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	writeF := func(ch chan<- []types.Trade) {
		ch <- trades
	}
	svc := getTestService(t)
	defer svc.Finish()
	ctx, cfunc := context.WithCancel(svc.ctx)
	svc.trade.EXPECT().Subscribe(gomock.Any()).Times(1).DoAndReturn(func(ch chan<- []types.Trade) uint64 {
		go writeF(ch)
		return ref
	})
	svc.trade.EXPECT().Unsubscribe(ref).Times(1).Return(nil).Do(func(_ uint64) {
		wg.Done()
	})
	ch, rref := svc.ObserveTrades(ctx, 0, nil, nil)
	// wait for data on channel
	gotTrades := <-ch
	// ensure we got the data we expected
	assert.Equal(t, ref, rref)
	assert.Equal(t, trades, gotTrades)
	// unsubscript
	cfunc()
	// ensure unsubscribe was indeed called before returning
	wg.Wait()
}

func testObserveTradesNoWrite(t *testing.T) {
	ref := uint64(1)
	market := "BTC/DEC19"
	buyer, seller := "buyerID", "sellerID"
	trades := []types.Trade{
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade1",
			MarketID: market,
			Price:    1000,
			Size:     1,
			Buyer:    buyer,
			Seller:   seller,
		},
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade2",
			MarketID: market,
			Price:    1200,
			Size:     2,
			Buyer:    buyer,
			Seller:   seller,
		},
	}
	done := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	writeF := func(ch chan<- []types.Trade) {
		ch <- trades
		wg.Done()
	}
	svc := getTestService(t)
	defer svc.Finish()
	ctx, cfunc := context.WithCancel(svc.ctx)
	svc.trade.EXPECT().Subscribe(gomock.Any()).Times(1).DoAndReturn(func(ch chan<- []types.Trade) uint64 {
		go writeF(ch)
		return ref
	})
	svc.trade.EXPECT().Unsubscribe(ref).Times(1).Return(nil).Do(func(_ uint64) {
		done <- struct{}{}
	})
	ch, rref := svc.ObserveTrades(ctx, 0, nil, nil)
	// do not read channel
	wg.Wait()
	// cancel context, write will not happen to channel
	cfunc()
	// ensure unsubscribe was called (and channels were closed)
	<-done
	// wait for data on channel
	gotTrades := <-ch
	// ensure we got the data we expected
	assert.Equal(t, ref, rref)
	assert.Nil(t, gotTrades)
}

func testObserveTradesFilterSuccess(t *testing.T) {
	ref := uint64(1)
	market := "BTC/DEC19"
	filterMarket := "foobar"
	buyer, seller := "buyerID", "sellerID"
	trades := []types.Trade{
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade1",
			MarketID: market,
			Price:    1000,
			Size:     1,
			Buyer:    buyer,
			Seller:   seller,
		},
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade2",
			MarketID: market,
			Price:    1200,
			Size:     2,
			Buyer:    buyer,
			Seller:   seller,
		},
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade3",
			MarketID: filterMarket,
			Price:    1200,
			Size:     2,
			Buyer:    buyer,
			Seller:   seller,
		},
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	writeF := func(ch chan<- []types.Trade) {
		ch <- trades
	}
	svc := getTestService(t)
	defer svc.Finish()
	ctx, cfunc := context.WithCancel(svc.ctx)
	svc.trade.EXPECT().Subscribe(gomock.Any()).Times(1).DoAndReturn(func(ch chan<- []types.Trade) uint64 {
		go writeF(ch)
		return ref
	})
	svc.trade.EXPECT().Unsubscribe(ref).Times(1).Return(nil).Do(func(_ uint64) {
		wg.Done()
	})
	// filter by market and party
	ch, rref := svc.ObserveTrades(ctx, 1, &filterMarket, &buyer)
	// wait for data on channel
	gotTrades := <-ch
	// ensure we got the data we expected
	assert.Equal(t, ref, rref)
	assert.Equal(t, 1, len(gotTrades))
	assert.Equal(t, filterMarket, gotTrades[0].MarketID)
	// unsubscript
	cfunc()
	// ensure unsubscribe was indeed called before returning
	wg.Wait()
}

func testObserveTradesFilterNone(t *testing.T) {
	ref := uint64(1)
	market := "BTC/DEC19"
	filterMarket := "foobar"
	buyer, seller := "buyerID", "sellerID"
	trades := []types.Trade{
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade1",
			MarketID: market,
			Price:    1000,
			Size:     1,
			Buyer:    buyer,
			Seller:   seller,
		},
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade2",
			MarketID: market,
			Price:    1200,
			Size:     2,
			Buyer:    buyer,
			Seller:   seller,
		},
		{
			Type:     types.Trade_TYPE_DEFAULT,
			Id:       "trade3",
			MarketID: filterMarket,
			Price:    1200,
			Size:     2,
			Buyer:    buyer,
			Seller:   seller,
		},
	}
	done := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	writeF := func(ch chan<- []types.Trade) {
		ch <- trades
		done <- struct{}{}
	}
	svc := getTestService(t)
	defer svc.Finish()
	ctx, cfunc := context.WithCancel(svc.ctx)
	svc.trade.EXPECT().Subscribe(gomock.Any()).Times(1).DoAndReturn(func(ch chan<- []types.Trade) uint64 {
		go writeF(ch)
		return ref
	})
	svc.trade.EXPECT().Unsubscribe(ref).Times(1).Return(nil).Do(func(_ uint64) {
		wg.Done()
	})
	// filter by specific market, and use market as party, no results will be returned
	ch, rref := svc.ObserveTrades(ctx, 0, &filterMarket, &market)
	// wait for data on channel
	<-done
	// ensure unsubscribe is called
	cfunc()
	// ensure unsubscribe was indeed called before returning
	wg.Wait()
	gotTrades := <-ch
	// ensure we got the data we expected
	assert.Equal(t, ref, rref)
	assert.Empty(t, gotTrades)
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
//	var tradeService = NewService()
//
//	vega := &core.Vega{}
//	tradeService.Init(vega, &tradeStore)
//
//	tradeStore.On("GetByOrderId", market, orderId, datastore.GetParams{Limit: datastore.GetParamsLimitDefault}).Return([]datastore.Trade{
//		{Trade: types.Trade{Type: types.Trade_TYPE_DEFAULT, Id: "A", MarketID: market, Price: 1}, OrderId: orderId},
//		{Trade: types.Trade{Type: types.Trade_TYPE_DEFAULT, Id: "B", MarketID: market, Price: 2}, OrderId: orderId},
//		{Trade: types.Trade{Type: types.Trade_TYPE_DEFAULT, Id: "C", MarketID: market, Price: 3}, OrderId: orderId},
//		{Trade: types.Trade{Type: types.Trade_TYPE_DEFAULT, Id: "D", MarketID: market, Price: 4}, OrderId: orderId},
//		{Trade: types.Trade{Type: types.Trade_TYPE_DEFAULT, Id: "E", MarketID: market, Price: 5}, OrderId: orderId},
//		{Trade: types.Trade{Type: types.Trade_TYPE_DEFAULT, Id: "F", MarketID: market, Price: 6}, OrderId: orderId},
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
//		Order: types.Order{Id: orderId, MarketID: market},
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
