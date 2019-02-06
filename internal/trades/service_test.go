package trades

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"vega/internal/storage"
	"vega/internal/storage/mocks"
	"vega/msg"
	"vega/internal/filtering"
	"github.com/pkg/errors"
)

// storageConfig specifies that the badger files are kept in a different
// directory when the candle service tests run. This is useful as when
// all the unit tests are run for the project they can be run in parallel.
func storageConfig() *storage.Config {
	storeConfig := storage.NewTestConfig()
	storeConfig.CandleStoreDirPath = "../storage/tmp/candlestore-h2n4k"
	storeConfig.OrderStoreDirPath = "../storage/tmp/orderstore-h2n4k"
	storeConfig.TradeStoreDirPath = "../storage/tmp/tradestore-h2n4k"
	return storeConfig
}

func TestNewTradeService(t *testing.T) {
	config := storageConfig()
	storage.FlushStores(config)
	
	tradeStore, err := storage.NewTradeStore(config)
	defer tradeStore.Close()
	assert.Nil(t, err)
	assert.NotNil(t, tradeStore)

	riskStore, err := storage.NewRiskStore(config)
	defer riskStore.Close()
	assert.Nil(t, err)
	assert.NotNil(t, tradeStore)

	var newTradeService = NewTradeService(tradeStore, riskStore)
	assert.NotNil(t, newTradeService)
}

func TestTradeService_GetByMarket(t *testing.T) {
	market := "BTC/DEC19"
	invalid := "LTC/DEC19"

	tradeStore := &mocks.TradeStore{}
	riskStore := &mocks.RiskStore{}
	tradeService := NewTradeService(tradeStore, riskStore)
	assert.NotNil(t, tradeService)

	// Scenario 1: valid market has n trades
	tradeStore.On("GetByMarket", market, &filtering.TradeQueryFilters{}).Return([]*msg.Trade{
		{Id: "A", Market: market, Price: 100},
		{Id: "B", Market: market, Price: 200},
		{Id: "C", Market: market, Price: 300},
	}, nil).Once()

	var tradeSet, err = tradeService.GetByMarket(market, &filtering.TradeQueryFilters{})
	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 3, len(tradeSet))
	tradeStore.AssertExpectations(t)

	// Scenario 2: invalid market returns an error
	tradeStore.On("GetByMarket", invalid, &filtering.TradeQueryFilters{}).Return(nil,
		errors.New("phobos communications link interrupted")).Once()

	tradeSet, err = tradeService.GetByMarket(invalid, &filtering.TradeQueryFilters{})
	assert.NotNil(t, err)
	assert.Nil(t, tradeSet)
}

func TestTradeService_GetByParty(t *testing.T) {
	partyA := "ramsey"
	partyB := "barney"
	invalid := "chris"

	tradeStore := &mocks.TradeStore{}
	riskStore := &mocks.RiskStore{}
	tradeService := NewTradeService(tradeStore, riskStore)
	assert.NotNil(t, tradeService)

	// Scenario 1: valid market has n trades
	tradeStore.On("GetByParty", partyA, &filtering.TradeQueryFilters{}).Return([]*msg.Trade{
		{Id: "A", Buyer: partyA, Seller: partyB, Price: 100},
		{Id: "B", Buyer: partyB, Seller: partyA, Price: 200},
	}, nil).Once()

	var tradeSet, err = tradeService.GetByParty(partyA, &filtering.TradeQueryFilters{})
	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 2, len(tradeSet))
	tradeStore.AssertExpectations(t)

	// Scenario 2: invalid market returns an error
	tradeStore.On("GetByMarket", invalid, &filtering.TradeQueryFilters{}).Return(nil,
		errors.New("phobos communications link interrupted")).Once()

	tradeSet, err = tradeService.GetByParty(invalid, &filtering.TradeQueryFilters{})
	assert.NotNil(t, err)
	assert.Nil(t, tradeSet)
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
//		{Trade: msg.Trade{Id: "A", Market: market, Price: 1}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "B", Market: market, Price: 2}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "C", Market: market, Price: 3}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "D", Market: market, Price: 4}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "E", Market: market, Price: 5}, OrderId: orderId},
//		{Trade: msg.Trade{Id: "F", Market: market, Price: 6}, OrderId: orderId},
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
//		Order: msg.Order{Id: orderId, Market: market},
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
