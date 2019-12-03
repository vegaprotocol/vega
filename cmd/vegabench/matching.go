package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/mock/gomock"
)

const marketID = "BTC/DEC19"

type execEngine struct {
	*execution.Engine
	ctrl         *gomock.Controller
	time         *mocks.MockTimeService
	order        *mocks.MockOrderBuf
	trade        *mocks.MockTradeBuf
	candle       *mocks.MockCandleBuf
	market       *mocks.MockMarketBuf
	party        *mocks.MockPartyBuf
	marginLevels *mocks.MockMarginLevelsBuf
}

func getExecEngine(b *testing.B, log *logging.Logger) *execEngine {
	ctrl := gomock.NewController(b)
	time := mocks.NewMockTimeService(ctrl)
	order := mocks.NewMockOrderBuf(ctrl)
	trade := mocks.NewMockTradeBuf(ctrl)
	candle := mocks.NewMockCandleBuf(ctrl)
	market := mocks.NewMockMarketBuf(ctrl)
	marketdata := mocks.NewMockMarketDataBuf(ctrl)
	party := mocks.NewMockPartyBuf(ctrl)
	accounts, _ := storage.NewAccounts(log, storage.NewDefaultConfig(""))
	accountBuf := buffer.NewAccount(accounts)
	settleBuf := buffer.NewSettlement()
	transferResponse := mocks.NewMockTransferBuf(ctrl)
	marginLevelsBuf := mocks.NewMockMarginLevelsBuf(ctrl)
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
		accountBuf,
		transferResponse,
		marketdata,
		marginLevelsBuf,
		settleBuf,
		[]types.Market{},
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
		executionEngine.order.EXPECT().Add(gomock.Any()).AnyTimes()
		executionEngine.trade.EXPECT().Add(gomock.Any())
		executionEngine.market.EXPECT().Add(gomock.Any())

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
