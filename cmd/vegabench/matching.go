package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
	"vega/internal/execution"
	"vega/internal/matching"
	types "vega/proto"

	"vega/internal/logging"
	mockStorage "vega/internal/storage/mocks"
	mockVegaTime "vega/internal/vegatime/mocks"

	"github.com/stretchr/testify/mock"
)

const marketID = "BTC/DEC19"

func BenchmarkMatching(
	numberOfOrders int,
	b *testing.B,
	quiet bool,
	randSize bool,
	reportInterval int) {

	times := 1
	if b != nil {
		b.ReportAllocs()
		times = b.N
	}
	if reportInterval == 0 {
		reportInterval = numberOfOrders
	}

	for k := 0; k < times; k++ {
		totalElapsed := time.Duration(0)

		timeService := &mockVegaTime.Service{}
		orderStore := &mockStorage.OrderStore{}
		tradeStore := &mockStorage.TradeStore{}
		candleStore := &mockStorage.CandleStore{}

		// Refer to the proto package by its real name, not by its alias "types".
		candleStore.On("AddTradeToBuffer", mock.AnythingOfType("proto.Trade")).Return(nil)

		orderStore.On("Post", mock.AnythingOfType("proto.Order")).Return(nil)
		orderStore.On("Put", mock.AnythingOfType("proto.Order")).Return(nil)

		tradeStore.On("Post", mock.AnythingOfType("*proto.Trade")).Return(nil)

		logger := logging.NewLoggerFromEnv("dev")
		logger.SetLevel(logging.InfoLevel, false)
		defer logger.Sync()

		// Matching engine (todo) create these inside execution engine based on config
		matchingConfig := matching.NewConfig(logger)
		matchingEngine := matching.NewMatchingEngine(matchingConfig)

		// Execution engine (broker operation of markets at runtime etc)
		eec := execution.NewConfig(logger)
		executionEngine := execution.NewExecutionEngine(eec, matchingEngine,
			timeService, orderStore, tradeStore, candleStore)

		var timestamp int64
		for o := 0; o < numberOfOrders; o++ {
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
			order.Market = marketID
			order.Party = fmt.Sprintf("P%v", timestamp)
			order.Side = types.Side(rand.Intn(2))
			order.Price = uint64(rand.Intn(100) + 50)
			order.Size = size
			order.Remaining = size
			order.Type = types.Order_GTC
			order.Timestamp = uint64(timestamp)

			start := time.Now()
			oc, oe := executionEngine.SubmitOrder(order)
			end := time.Now()
			if oe == 0 {
				oc.Release()
			}
			totalElapsed += end.Sub(start)
		}

		if !quiet {
			fmt.Printf(
				"(n=%v) Elapsed = %v, average = %v\n",
				numberOfOrders,
				totalElapsed,
				totalElapsed/time.Duration(numberOfOrders))
			// fmt.Printf(
			// 	"(n=%v) %v %v\n",
			// 	numberOfOrders,
			// 	vega.GetMarketData(marketId),
			// 	vega.GetMarketDepth(marketId))
		}
	}
}
