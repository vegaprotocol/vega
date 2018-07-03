package main

import (
	"fmt"
	"math/rand"
	"testing"
	"vega/core"
	"vega/proto"
)

const marketId = "TEST"

func BenchmarkMatching(
	numberOfOrders int,
	b *testing.B,
	quiet bool,
	blockSize int,
	randSize bool,
	reportInterval int) {

		b.ReportAllocs()

	if reportInterval == 0 {
		reportInterval = numberOfOrders
	}

	config := core.DefaultConfig()
	config.Matching.Quiet = true

	vega := core.New(config)
	vega.CreateMarket(marketId)
	timestamp := uint64(0)
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
		_ = result

	}

}
