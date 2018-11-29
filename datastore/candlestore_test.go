package datastore

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"fmt"
	"os"
	"time"
)

const candleStoreDir string = "../tmp/candlestore-test"

func FlushCandleStore() {
	err := os.RemoveAll(candleStoreDir)
	if err != nil {
		fmt.Printf("UNABLE TO FLUSH DB: %s\n", err.Error())
	}
}

func TestCandleGenerator_Generate(t *testing.T) {
	FlushCandleStore()
	candleStore := NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	// t0 = 2018-11-13T11:01:14Z
	t0 := uint64(1542106874000000000)
	t0Seconds := int64(1542106874)
	t0NanoSeconds := int64(000000000)

	fmt.Printf("t0 = %s\n", time.Unix(t0Seconds, t0NanoSeconds).Format(time.RFC3339))

	var trades = []*msg.Trade{
		{Id: "1", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},

		{Id: "3", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute)},
		{Id: "4", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 20 * time.Second)},
	}

	candleStore.StartNewBuffer(testMarket, t0)
	for idx := range trades {
		candleStore.AddTradeToBuffer(trades[idx].Market, *trades[idx])
	}
	candleStore.GenerateCandlesFromBuffer(testMarket)
	fmt.Printf("Candles GenerateCandlesFromBuffer\n")

	candles := candleStore.GetCandles(testMarket, t0, msg.Interval_I1M)
	fmt.Printf("Candles fetched for t0 and 1m: %+v\n", candles)

	assert.Equal(t, 2, len(candles))
	fmt.Printf("%s", time.Unix(1542106860,000000000).Format(time.RFC3339))
	assert.Equal(t, uint64(1542106860000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(200), candles[0].Volume)

	assert.Equal(t, uint64(1542106920000000000), candles[1].Timestamp)
	assert.Equal(t, uint64(100), candles[1].High)
	assert.Equal(t, uint64(100), candles[1].Low)
	assert.Equal(t, uint64(100), candles[1].Open)
	assert.Equal(t, uint64(100), candles[1].Close)
	assert.Equal(t, uint64(200), candles[1].Volume)

	candles = candleStore.GetCandles(testMarket, t0 + uint64(1 * time.Minute), msg.Interval_I1M)
	fmt.Printf("Candles fetched for t0 and 1m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106920000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(200), candles[0].Volume)

	candles = candleStore.GetCandles(testMarket, t0 + uint64(1 * time.Minute), msg.Interval_I5M)
	fmt.Printf("Candles fetched for t0 and 5m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)


	fmt.Printf("\n\nALL GOOD MAN\n\n")
	//------------------- generate empty candles-------------------------//

	currentVegaTime := uint64(t0) + uint64(2 * time.Minute)
	candleStore.StartNewBuffer(testMarket, currentVegaTime)
	candleStore.GenerateCandlesFromBuffer(testMarket)

	candles = candleStore.GetCandles(testMarket, t0, msg.Interval_I1M)
	fmt.Printf("Candles fetched for t0 and 1m: %+v\n", candles)

	assert.Equal(t, 3, len(candles))
	assert.Equal(t, uint64(1542106860000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(200), candles[0].Volume)

	assert.Equal(t, uint64(1542106920000000000), candles[1].Timestamp)
	assert.Equal(t, uint64(100), candles[1].High)
	assert.Equal(t, uint64(100), candles[1].Low)
	assert.Equal(t, uint64(100), candles[1].Open)
	assert.Equal(t, uint64(100), candles[1].Close)
	assert.Equal(t, uint64(200), candles[1].Volume)

	assert.Equal(t, uint64(1542106980000000000), candles[2].Timestamp)
	assert.Equal(t, uint64(100), candles[2].High)
	assert.Equal(t, uint64(100), candles[2].Low)
	assert.Equal(t, uint64(100), candles[2].Open)
	assert.Equal(t, uint64(100), candles[2].Close)
	assert.Equal(t, uint64(0), candles[2].Volume)


	candles = candleStore.GetCandles(testMarket, t0, msg.Interval_I5M)
	fmt.Printf("Candles fetched for t0 and 5m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	candles = candleStore.GetCandles(testMarket, t0 + uint64(2 * time.Minute), msg.Interval_I15M)
	fmt.Printf("Candles fetched for t0 and 15m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)


	candles = candleStore.GetCandles(testMarket, t0 + uint64(17 * time.Minute), msg.Interval_I15M)
	fmt.Printf("Candles fetched for t0 and 15m: %+v\n", candles)

	assert.Equal(t, 0, len(candles))

	currentVegaTime = uint64(t0) + uint64(17 * time.Minute)
	candleStore.StartNewBuffer(testMarket, currentVegaTime)
	candleStore.GenerateCandlesFromBuffer(testMarket)

	candles = candleStore.GetCandles(testMarket, t0 + uint64(17 * time.Minute), msg.Interval_I15M)
	fmt.Printf("Candles fetched for t0 and 15m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542107700000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(0), candles[0].Volume)
}

func TestGetMapOfIntervalsToTimestamps(t *testing.T) {
	timestamp, _ := time.Parse(time.RFC3339, "2018-11-13T11:01:14Z")
	t0 := uint64(timestamp.UnixNano())
	fmt.Printf("%d", timestamp.UnixNano())

	timestamps := getMapOfIntervalsToRoundedTimestamps(uint64(timestamp.UnixNano()))
	assert.Equal(t, t0 - uint64(14 * time.Second), timestamps[msg.Interval_I1M])
	assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), timestamps[msg.Interval_I5M])
	assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), timestamps[msg.Interval_I15M])
	assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), timestamps[msg.Interval_I1H])
	assert.Equal(t, t0 - uint64(5 * time.Hour + time.Minute + 14 * time.Second), timestamps[msg.Interval_I6H])
	assert.Equal(t, t0 - uint64(11 * time.Hour + time.Minute + 14 * time.Second), timestamps[msg.Interval_I1D])
}

func TestCandleStore_SubscribeUnsubscribe(t *testing.T) {
	FlushCandleStore()
	candleStore := NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	internalTransport := make(map[msg.Interval]chan msg.Candle, 0)
	ref := candleStore.Subscribe(internalTransport)
	assert.Equal(t, uint64(1), ref)

	ref = candleStore.Subscribe(internalTransport)
	assert.Equal(t, uint64(2), ref)

	fmt.Printf("Unsubscribing\n")
	err := candleStore.Unsubscribe(1)
	assert.Nil(t, err)

	err = candleStore.Unsubscribe(1)
	assert.Equal(t, "CandleStore subscriber does not exist with id: 1", err.Error())

	err = candleStore.Unsubscribe(2)
	assert.Nil(t, err)

	fmt.Printf("Totally empty\n")

	err = candleStore.Unsubscribe(2)
	assert.Nil(t, err)
}

func TestCandleStore_QueueNotify(t *testing.T) {
	FlushCandleStore()
	candleStore := NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	internalTransport := make(map[msg.Interval]chan msg.Candle, 0)
	_ = candleStore.Subscribe(internalTransport)

	timestamp, _ := time.Parse(time.RFC3339, "2018-11-13T11:01:14Z")
	t0 := timestamp.UnixNano()

	candle1m := NewCandle(uint64(t0), 100, 100, msg.Interval_I1M)
	candle5m := NewCandle(uint64(t0), 100, 100, msg.Interval_I5M)
	candle15m := NewCandle(uint64(t0), 100, 100, msg.Interval_I15M)
	candle1h := NewCandle(uint64(t0), 100, 100, msg.Interval_I1H)
	candle6h := NewCandle(uint64(t0), 100, 100, msg.Interval_I6H)
	candle1d := NewCandle(uint64(t0), 100, 100, msg.Interval_I1D)

	candleStore.QueueEvent(*candle1m, msg.Interval_I1M)
	candleStore.QueueEvent(*candle5m, msg.Interval_I5M)
	candleStore.QueueEvent(*candle15m, msg.Interval_I15M)
	candleStore.QueueEvent(*candle1h, msg.Interval_I1H)
	candleStore.QueueEvent(*candle6h, msg.Interval_I6H)
	candleStore.QueueEvent(*candle1d, msg.Interval_I1D)

	assert.Equal(t, true, isTransportEmpty(internalTransport[msg.Interval_I1M]))
	assert.Equal(t, true, isTransportEmpty(internalTransport[msg.Interval_I5M]))
	assert.Equal(t, true, isTransportEmpty(internalTransport[msg.Interval_I15M]))
	assert.Equal(t, true, isTransportEmpty(internalTransport[msg.Interval_I1H]))
	assert.Equal(t, true, isTransportEmpty(internalTransport[msg.Interval_I6H]))
	assert.Equal(t, true, isTransportEmpty(internalTransport[msg.Interval_I1D]))

	candleStore.Notify()

	candle := <- internalTransport[msg.Interval_I1M]
	assert.Equal(t, candle.Interval, msg.Interval_I1M)

	candle = <- internalTransport[msg.Interval_I5M]
	assert.Equal(t, candle.Interval, msg.Interval_I5M)

	candle = <- internalTransport[msg.Interval_I15M]
	assert.Equal(t, candle.Interval, msg.Interval_I15M)

	candle = <- internalTransport[msg.Interval_I1H]
	assert.Equal(t, candle.Interval, msg.Interval_I1H)

	candle = <- internalTransport[msg.Interval_I6H]
	assert.Equal(t, candle.Interval, msg.Interval_I6H)

	candle = <- internalTransport[msg.Interval_I1D]
	assert.Equal(t, candle.Interval, msg.Interval_I1D)

}

func isTransportEmpty(transport chan msg.Candle) bool {
	select {
	case  <- transport:
		return false
	default:
		return true
	}
}