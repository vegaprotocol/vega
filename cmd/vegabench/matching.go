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

	if reportInterval == 0 {
		reportInterval = numberOfOrders
	}

	config := core.DefaultConfig()

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

		order := msg.OrderPool.Get().(*msg.Order)
		order.Market = marketId
		order.Party = fmt.Sprintf("P%v", timestamp)
		order.Side = msg.Side(rand.Intn(2))
		order.Price = uint64(rand.Intn(100) + 50)
		order.Size = size
		order.Remaining = size
		order.Type = msg.Order_GTC
		order.Timestamp = timestamp

		oconfirm, oerr := vega.SubmitOrder(order)
		if oerr == 0 {
			oconfirm.Release()
		}
	}

}
