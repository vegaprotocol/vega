package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/execution/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
)

const marketID = "BTC/DEC19"

type execEngine struct {
	*execution.Engine
	ctrl   *gomock.Controller
	time   *mocks.MockTimeService
	order  *mocks.MockOrderStore
	trade  *mocks.MockTradeStore
	candle *mocks.MockCandleStore
	market *mocks.MockMarketStore
	party  *mocks.MockPartyStore
}

func getExecEngine(b *testing.B, log *logging.Logger) *execEngine {
	ctrl := gomock.NewController(b)
	time := mocks.NewMockTimeService(ctrl)
	order := mocks.NewMockOrderStore(ctrl)
	trade := mocks.NewMockTradeStore(ctrl)
	candle := mocks.NewMockCandleStore(ctrl)
	market := mocks.NewMockMarketStore(ctrl)
	party := mocks.NewMockPartyStore(ctrl)
	accounts, _ := storage.NewAccounts(log, storage.NewDefaultConfig(""))
	executionConfig := execution.NewDefaultConfig("")

	engine := execution.NewEngine(
		log,
		executionConfig,
		time,
		order,
		trade,
		candle,
		market,
		party,
		accounts,
	)
	return &execEngine{
		Engine: engine,
		ctrl:   ctrl,
		time:   time,
		order:  order,
		trade:  trade,
		candle: candle,
		market: market,
		party:  party,
	}
}

func benchmarkMatching(
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

		logger := logging.NewDevLogger()
		logger.SetLevel(logging.InfoLevel)

		// Execution engine (broker operation of markets at runtime etc)
		executionEngine := getExecEngine(b, logger)
		executionEngine.order.EXPECT().Post(gomock.Any()).Return(nil)
		executionEngine.order.EXPECT().Put(gomock.Any()).Return(nil)
		executionEngine.trade.EXPECT().Post(gomock.Any()).Return(nil)
		executionEngine.market.EXPECT().Post(gomock.Any()).Return(nil)

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

			order := &types.Order{
				MarketID:    marketID,
				PartyID:     fmt.Sprintf("P%v", timestamp),
				Side:        types.Side(rand.Intn(2)),
				Price:       uint64(rand.Intn(100) + 50),
				Size:        size,
				Remaining:   size,
				TimeInForce: types.Order_GTC,
				CreatedAt:   timestamp,
			}
			start := vegatime.Now()
			executionEngine.SubmitOrder(order)
			end := vegatime.Now()
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
		executionEngine.ctrl.Finish()
		logger.Sync()
	}
}
