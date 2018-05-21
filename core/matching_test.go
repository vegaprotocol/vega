package core

import (
	"testing"
	"fmt"
	"time"
	"math/rand"
	"vega/proto"
)

func BenchmarkMatching100(b *testing.B) {
	benchmarkMatching(100, b, true, 1, true, 0)
}

func BenchmarkMatching1000(b *testing.B) {
	benchmarkMatching(1000, b, true, 1, true, 0)
}

func BenchmarkMatching10000(b *testing.B) {
	benchmarkMatching(10000, b, true, 1, true, 0)
}

func BenchmarkMatching100000(b *testing.B) {
	benchmarkMatching(100000, b, true, 1, true, 0)
}


func BenchmarkMatching100Allocated(b *testing.B) {
	benchmarkMatching(100, b, true, 0, true, 0)
}

func BenchmarkMatching1000Allocated(b *testing.B) {
	benchmarkMatching(1000, b, true, 0, true, 0)
}

func BenchmarkMatching10000Allocated(b *testing.B) {
	benchmarkMatching(10000, b, true, 0, true, 0)
}

func BenchmarkMatching100000Allocated(b *testing.B) {
	benchmarkMatching(100000, b, true, 0, true, 0)
}

func BenchmarkMatching100Uniform(b *testing.B) {
	benchmarkMatching(100, b, true, 1, false, 0)
}

func BenchmarkMatching1000Uniform(b *testing.B) {
	benchmarkMatching(1000, b, true, 1, false, 0)
}

func BenchmarkMatching10000Uniform(b *testing.B) {
	benchmarkMatching(10000, b, true, 1, false, 0)
}

func BenchmarkMatching100000Uniform(b *testing.B) {
	benchmarkMatching(100000, b, true, 1, false, 0)
}


const marketId = "TEST"

func benchmarkMatching(
	numberOfOrders int,
	b *testing.B,
	quiet bool,
	blockSize int,
	randSize bool,
	reportInterval int) {

	var times int
	if b != nil {
		times = b.N
	} else {
		times = 1
	}

	if reportInterval == 0 {
		reportInterval = numberOfOrders
	}

	config := DefaultConfig()
	config.Matching.Quiet = true

	for k := 0; k < times; k++ {
		vega := New(config)
		vega.CreateMarket(marketId)
		totalElapsed := time.Duration(0)
		periodElapsed := totalElapsed
		totalTrades := 0
		periodTrades := totalTrades
		timestamp := uint64(0)
		for i := 1; i <= numberOfOrders; i++ {
			if blockSize == 0 || (i%blockSize) == 0 {
				timestamp++
			}
			var size uint64
			if randSize {
				size = uint64(rand.Intn(250) + 1)
			} else {
				size = 50
			}
			start := time.Now()
			result, _ := vega.SubmitOrder(msg.Order{
				Market:    marketId,
				Party:     fmt.Sprintf("P%v", timestamp),
				Side:      msg.Side(rand.Intn(2)),
				Price:     uint64(rand.Intn(100) + 50),
				Size:      size,
				Remaining: size,
				Type:      msg.Order_GTC,
				Timestamp: timestamp,
			})
			end := time.Now()
			totalElapsed += end.Sub(start)
			periodElapsed += end.Sub(start)
			totalTrades += len(result.Trades)
			periodTrades += len(result.Trades)

			if !quiet && reportInterval != numberOfOrders && i%reportInterval == 0 {
				fmt.Printf(
					"(n=%v/%v) Elapsed = %v, average = %v; matched %v trades, average %v trades per order\n",
					i,
					numberOfOrders,
					totalElapsed,
					periodElapsed/time.Duration(reportInterval),
					periodTrades,
					float64(periodTrades)/float64(reportInterval))
				fmt.Printf(
					"(n=%v/%v) %v %v\n",
					i,
					numberOfOrders,
					vega.GetMarketData(marketId),
					vega.GetMarketDepth(marketId))
				periodTrades = 0
				periodElapsed = 0
			}
		}

		if !quiet {
			fmt.Printf(
				"(n=%v) Elapsed = %v, average = %v; matched %v trades, average %v trades per order\n",
				numberOfOrders,
				totalElapsed,
				totalElapsed/time.Duration(numberOfOrders),
				totalTrades,
				float64(totalTrades)/float64(reportInterval))
			fmt.Printf(
				"(n=%v) %v %v\n",
				numberOfOrders,
				vega.GetMarketData(marketId),
				vega.GetMarketDepth(marketId))
		}
	}

}
