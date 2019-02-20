package main

import (
	"fmt"
	"math/rand"
	"testing"
	"vega/internal/execution"
	"vega/internal/matching"
	types "vega/proto"

	"vega/internal/logging"
	mockStorage "vega/internal/storage/mocks"
	mockVegaTime "vega/internal/vegatime/mocks"
)

const marketId = "BTC/JAN21"

func BenchmarkMatching(
	numberOfOrders int,
	b *testing.B,
	randSize bool,
	reportInterval int) {

	b.ReportAllocs()
	if reportInterval == 0 {
		reportInterval = numberOfOrders
	}

	timeService := &mockVegaTime.Service{}
	orderStore := &mockStorage.OrderStore{}
	tradeStore := &mockStorage.TradeStore{}

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	// Matching engine (todo) create these inside execution engine based on config
	matchingConfig := matching.NewConfig(logger)
	matchingEngine := matching.NewMatchingEngine(matchingConfig)
	matchingEngine.CreateMarket(marketId)

	// Execution engine (broker operation of markets at runtime etc)
	eec := execution.NewConfig(logger)
	executionEngine := execution.NewExecutionEngine(eec, matchingEngine, timeService, orderStore, tradeStore)

	var timestamp int64
	for k := 0; k < b.N; k++ {
		if rand.Intn(5) > 1 {
			timestamp++
		}
		var size uint64
		if randSize {
			size = uint64(rand.Intn(250) + 1)
		} else {
			size = 50
		}

		order := types.OrderPool.Get().(*types.Order)
		order.Market = marketId
		order.Party = fmt.Sprintf("P%v", timestamp)
		order.Side = types.Side(rand.Intn(2))
		order.Price = uint64(rand.Intn(100) + 50)
		order.Size = size
		order.Remaining = size
		order.Type = types.Order_GTC
		order.Timestamp = timestamp

		oc, oe := executionEngine.SubmitOrder(order)
		if oe == 0 {
			oc.Release()
		}
		result, _ := executionEngine.SubmitOrder(&types.Order{
			Market:    marketId,
			Party:     fmt.Sprintf("P%v", timestamp),
			Side:      types.Side(rand.Intn(2)),
			Price:     uint64(rand.Intn(100) + 50),
			Size:      size,
			Remaining: size,
			Type:      types.Order_GTC,
			Timestamp: timestamp,
		})
		_ = result

	}
}
