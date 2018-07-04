package main

import (
	"fmt"
	"math/rand"
	"testing"

	"vega/core"
	"vega/proto"
)

const marketId = "TEST"

func NewOrder() interface{} {
	return &msg.Order{}
}

func BenchmarkMatching(
	numberOfOrders int,
	b *testing.B,
	quiet bool,
	blockSize int,
	randSize bool,
	reportInterval int) {

	//matching.Pool.New = NewOrder

	//var times int
	//if b != nil {
	//	times = b.N
	//} else {
	//	times = 1
	//}

	if reportInterval == 0 {
		reportInterval = numberOfOrders
	}

	config := core.DefaultConfig()
	//config.Matching.Quiet = true

	vega := core.New(config)
	vega.CreateMarket(marketId)

	timestamp := uint64(0)
	for k := 0; k < b.N; k++ {
		//totalElapsed := time.Duration(0)
		//periodElapsed := totalElapsed
		//totalTrades := 0
		//periodTrades := totalTrades
		//for i := 1; i <= numberOfOrders; i++ {
		//if blockSize == 0 || (i%blockSize) == 0 {
		if rand.Intn(5) > 1 {
			timestamp++
		}
		var size uint64
		if randSize {
			size = uint64(rand.Intn(250) + 1)
		} else {
			size = 50
		}
		//start := time.Now()
		order := msg.OrderPool.Get().(*msg.Order)
		//order := &msg.Order{}
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

		//result, _ := vega.SubmitOrder(msg.Order{
		//	Market:    marketId,
		//	Party:     fmt.Sprintf("P%v", timestamp),
		//	Side:      msg.Side(rand.Intn(2)),
		//	Price:     uint64(rand.Intn(100) + 50),
		//	Size:      size,
		//	Remaining: size,
		//	Type:      msg.Order_GTC,
		//	Timestamp: timestamp,
		//})
		//_ = result
		//end := time.Now()
		//totalElapsed += end.Sub(start)
		//periodElapsed += end.Sub(start)
		//totalTrades += len(result.Trades)
		//periodTrades += len(result.Trades)

		//if !quiet && reportInterval != numberOfOrders && i%reportInterval == 0 {
		//	fmt.Printf(
		//		"(n=%v/%v) Elapsed = %v, average = %v; matched %v trades, average %v trades per order\n",
		//		i,
		//		numberOfOrders,
		//		totalElapsed,
		//		periodElapsed/time.Duration(reportInterval),
		//		periodTrades,
		//		float64(periodTrades)/float64(reportInterval))
		//	fmt.Printf(
		//		"(n=%v/%v) \n",
		//		//"(n=%v/%v) %v %v\n",
		//		i,
		//		numberOfOrders,
		//		//vega.GetMarketData(marketId),
		//		//vega.GetMarketDepth(marketId))
		//		)
		//	periodTrades = 0
		//	periodElapsed = 0
		//}
		//}

		//if !quiet {
		//	fmt.Printf(
		//		"(n=%v) Elapsed = %v, average = %v; matched %v trades, average %v trades per order\n",
		//		numberOfOrders,
		//		totalElapsed,
		//		totalElapsed/time.Duration(numberOfOrders),
		//		totalTrades,
		//		float64(totalTrades)/float64(reportInterval))
		//	fmt.Printf(
		//		"(n=%v) \n",
		//		//"(n=%v) %v %v\n",
		//		numberOfOrders,
		//		//vega.GetMarketData(marketId),
		//		//vega.GetMarketDepth(marketId)
		//		)
		//}
	}

}
