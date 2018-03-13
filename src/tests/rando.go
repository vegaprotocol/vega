package tests

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"vega/src/core"
	"vega/src/proto"
)

func BenchmarkMatching(numberOfOrders int, b *testing.B, quiet bool, blockSize int, randSize bool) {

	var times int
	if b != nil {
		times = b.N
	} else {
		times = 1
	}

	config := core.DefaultConfig()
	config.Matching.Quiet = true

	for k := 0; k < times; k++ {
		vega := core.New(config)
		vega.CreateMarket("BTC/DEC18")
		totalElapsed := time.Duration(0)
		totalTrades := 0
		timestamp := uint64(0)
		for i := 0; i < numberOfOrders; i++ {
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
				Market:    "BTC/DEC18",
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
			totalTrades += len(result.Trades)
		}
		if !quiet {
			fmt.Printf("(n=%v) Elapsed = %v, average = %v; matched %v trades, average %v per order.",
				numberOfOrders,
				totalElapsed,
				totalElapsed/time.Duration(numberOfOrders),
				totalTrades,
				float64(totalTrades)/float64(numberOfOrders))
		}
	}

}