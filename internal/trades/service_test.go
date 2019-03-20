package trades

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/storage/mocks"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/logging"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

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

func TestNewTradeService(t *testing.T) {
	config := storageConfig(t)
	storage.FlushStores(config)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	tradeStore, err := storage.NewTradeStore(config)
	defer tradeStore.Close()
	assert.Nil(t, err)
	assert.NotNil(t, tradeStore)

	riskStore, err := storage.NewRiskStore(config)
	defer riskStore.Close()
	assert.Nil(t, err)
	assert.NotNil(t, tradeStore)

	tradeConfig := NewDefaultConfig(logger)
	newTradeService, err := NewTradeService(tradeConfig, tradeStore, riskStore)
	assert.Nil(t, err)
	assert.NotNil(t, newTradeService)
}

func TestTradeService_GetByMarket(t *testing.T) {
	ctx := context.Background()

	market := "BTC/DEC19"
	invalid := "LTC/DEC19"

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	tradeStore := &mocks.TradeStore{}
	riskStore := &mocks.RiskStore{}

	tradeConfig := NewDefaultConfig(logger)
	tradeService, err := NewTradeService(tradeConfig, tradeStore, riskStore)
	assert.Nil(t, err)
	assert.NotNil(t, tradeService)

	// Scenario 1: valid market has n trades
	tradeStore.On("GetByMarket", ctx, market, &filtering.TradeQueryFilters{}).Return([]*types.Trade{
		{Id: "A", Market: market, Price: 100},
		{Id: "B", Market: market, Price: 200},
		{Id: "C", Market: market, Price: 300},
	}, nil).Once()

	tradeSet, err := tradeService.GetByMarket(ctx, market, &filtering.TradeQueryFilters{})
	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 3, len(tradeSet))
	tradeStore.AssertExpectations(t)

	// Scenario 2: invalid market returns an error
	tradeStore.On("GetByMarket", ctx, invalid, &filtering.TradeQueryFilters{}).Return(nil,
		errors.New("phobos communications link interrupted")).Once()

	tradeSet, err = tradeService.GetByMarket(ctx, invalid, &filtering.TradeQueryFilters{})
	assert.NotNil(t, err)
	assert.Nil(t, tradeSet)
}

func TestTradeService_GetByParty(t *testing.T) {
	ctx := context.Background()

	partyA := "ramsey"
	partyB := "barney"
	invalid := "chris"

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	tradeStore := &mocks.TradeStore{}
	riskStore := &mocks.RiskStore{}
	tradeConfig := NewDefaultConfig(logger)
	tradeService, err := NewTradeService(tradeConfig, tradeStore, riskStore)
	assert.Nil(t, err)
	assert.NotNil(t, tradeService)

	// Scenario 1: valid market has n trades
	tradeStore.On("GetByParty", ctx, partyA, &filtering.TradeQueryFilters{}).Return([]*types.Trade{
		{Id: "A", Buyer: partyA, Seller: partyB, Price: 100},
		{Id: "B", Buyer: partyB, Seller: partyA, Price: 200},
	}, nil).Once()

	tradeSet, err := tradeService.GetByParty(context.Background(), partyA, &filtering.TradeQueryFilters{})
	assert.Nil(t, err)
	assert.NotNil(t, tradeSet)
	assert.Equal(t, 2, len(tradeSet))
	tradeStore.AssertExpectations(t)

	// Scenario 2: invalid market returns an error
	tradeStore.On("GetByParty", ctx, invalid, &filtering.TradeQueryFilters{}).Return(nil,
		errors.New("phobos communications link interrupted")).Once()

	tradeSet, err = tradeService.GetByParty(context.Background(), invalid, &filtering.TradeQueryFilters{})
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
