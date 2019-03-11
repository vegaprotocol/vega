package main

import (
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/matching"
	types "code.vegaprotocol.io/vega/proto"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	mockStorage "code.vegaprotocol.io/vega/internal/storage/mocks"
	mockVegaTime "code.vegaprotocol.io/vega/internal/vegatime/mocks"

	"github.com/stretchr/testify/mock"
)

const marketID = "BTC/DEC19"

func BenchmarkMatching(
	numberOfOrders int,
	b *testing.B,
	quiet bool,
	randSize bool,
	reportDuration string) {

	times := 1
	if b != nil {
		b.ReportAllocs()
		times = b.N
	}
	duration, err := time.ParseDuration(reportDuration)
	if err != nil {
		panic(err)
	}
	durationNano := duration.Nanoseconds()

	for k := 0; k < times; k++ {
		totalElapsed := time.Duration(0)
		// totalTrades := 0

		periodElapsed := time.Duration(0)
		// periodTrades := 0

		timeService := &mockVegaTime.Service{}
		orderStore := &mockStorage.OrderStore{}
		tradeStore := &mockStorage.TradeStore{}
		candleStore := &mockStorage.CandleStore{}
		marketStore := &mockStorage.MarketStore{}

		// Refer to the proto package by its real name, not by its alias "types".
		candleStore.On("AddTradeToBuffer", mock.AnythingOfType("proto.Trade")).Return(nil)

		orderStore.On("Post", mock.AnythingOfType("proto.Order")).Return(nil)
		orderStore.On("Put", mock.AnythingOfType("proto.Order")).Return(nil)

		tradeStore.On("Post", mock.AnythingOfType("*proto.Trade")).Return(nil)

		marketStore.On("Post", mock.AnythingOfType("*proto.Market")).Return(nil)

		logger := logging.NewLoggerFromEnv("dev")
		logger.SetLevel(logging.InfoLevel)
		defer logger.Sync()

		// Matching engine (trade matching)
		matchingConfig := matching.NewDefaultConfig(logger)
		matchingEngine := matching.NewMatchingEngine(matchingConfig)

		// Execution engine (broker operation of markets at runtime etc)
		eec := execution.NewDefaultConfig(logger)
		executionEngine := execution.NewExecutionEngine(eec, matchingEngine,
			timeService, orderStore, tradeStore, candleStore, marketStore)

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
			if oe == nil {
				oc.Release()
			}
			timetaken := end.Sub(start)
			totalElapsed += timetaken
			periodElapsed += timetaken

			if periodElapsed.Nanoseconds() > durationNano {
				if !quiet {
					fmt.Printf(
						"(%5.2f%%) Elapsed = %s, average = %v\n",
						float32(o)/float32(numberOfOrders)*100.0,
						totalElapsed.Round(time.Second).String(),
						totalElapsed/time.Duration(k*numberOfOrders+o),
					)
				}
				periodElapsed = time.Duration(0)
				// periodTrades = 0
			}
		}

		if !quiet {
			fmt.Printf(
				"(n=%d) Elapsed = %s, average = %v\n",
				numberOfOrders,
				totalElapsed.Round(time.Second).String(),
				totalElapsed/time.Duration(numberOfOrders))
			// fmt.Printf(
			// 	"(n=%v) %v %v\n",
			// 	numberOfOrders,
			// 	vega.GetMarketData(marketId),
			// 	vega.GetMarketDepth(marketId))
		}
	}
}
