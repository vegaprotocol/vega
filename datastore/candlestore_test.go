package datastore

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"fmt"
	"os"
	"time"
)

const (
	candleStoreDir = "./candleStore"
)


func flushCandleStore() {
	err := os.RemoveAll(candleStoreDir)
	if err != nil {
		fmt.Printf("UNABLE TO FLUSH DB: %s\n", err.Error())
	}
}

func TestCandleGenerator_Generate(t *testing.T) {

	flushCandleStore()
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

	for idx := range trades {
		candleStore.GenerateCandles(trades[idx])
	}

	// test for 1 minute intervals

	candles := candleStore.GetCandles(testMarket, t0, "1m")
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

	candles = candleStore.GetCandles(testMarket, t0 + uint64(1 * time.Minute), "1m")
	fmt.Printf("Candles fetched for t0 and 1m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106920000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(200), candles[0].Volume)

	candles = candleStore.GetCandles(testMarket, t0 + uint64(1 * time.Minute), "5m")
	fmt.Printf("Candles fetched for t0 and 5m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	//------------------- generate empty candles-------------------------//

	currentVegaTime := uint64(t0) + uint64(2 * time.Minute)
	candleStore.GenerateEmptyCandles(testMarket, currentVegaTime)
	candles = candleStore.GetCandles(testMarket, t0, "1m")
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


	candles = candleStore.GetCandles(testMarket, t0, "5m")
	fmt.Printf("Candles fetched for t0 and 5m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	candles = candleStore.GetCandles(testMarket, t0 + uint64(2 * time.Minute), "15m")
	fmt.Printf("Candles fetched for t0 and 15m: %+v\n", candles)

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)


	candles = candleStore.GetCandles(testMarket, t0 + uint64(17 * time.Minute), "15m")
	fmt.Printf("Candles fetched for t0 and 15m: %+v\n", candles)

	assert.Equal(t, 0, len(candles))

	currentVegaTime = uint64(t0) + uint64(17 * time.Minute)
	candleStore.GenerateEmptyCandles(testMarket, currentVegaTime)

	candles = candleStore.GetCandles(testMarket, t0 + uint64(17 * time.Minute), "15m")
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
	t0 := timestamp.UnixNano()
	fmt.Printf("%d", timestamp.UnixNano())

	timestamps := getMapOfIntervalsToTimestamps(timestamp.UnixNano())
	assert.Equal(t, t0 - int64(14 * time.Second), timestamps["1m"])
	assert.Equal(t, t0 - int64(time.Minute + 14 * time.Second), timestamps["5m"])
	assert.Equal(t, t0 - int64(time.Minute + 14 * time.Second), timestamps["15m"])
	assert.Equal(t, t0 - int64(time.Minute + 14 * time.Second), timestamps["1h"])
	assert.Equal(t, t0 - int64(5 * time.Hour + time.Minute + 14 * time.Second), timestamps["6h"])
	assert.Equal(t, t0 - int64(11 * time.Hour + time.Minute + 14 * time.Second), timestamps["1d"])
}

func TestCandleStore_SubscribeUnsubscribe(t *testing.T) {
	flushCandleStore()
	candleStore := NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	internalTransport := make(map[string]chan msg.Candle, 0)
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
	flushCandleStore()
	candleStore := NewCandleStore(candleStoreDir)
	defer candleStore.Close()

	internalTransport := make(map[string]chan msg.Candle, 0)
	_ = candleStore.Subscribe(internalTransport)

	timestamp, _ := time.Parse(time.RFC3339, "2018-11-13T11:01:14Z")
	t0 := timestamp.UnixNano()

	candle1m := NewCandle(uint64(t0), 100, 100, "1m")
	candle5m := NewCandle(uint64(t0), 100, 100, "5m")
	candle15m := NewCandle(uint64(t0), 100, 100, "15m")
	candle1h := NewCandle(uint64(t0), 100, 100, "1h")
	candle6h := NewCandle(uint64(t0), 100, 100, "6h")
	candle1d := NewCandle(uint64(t0), 100, 100, "1d")

	candleStore.QueueEvent(*candle1m, "1m")
	candleStore.QueueEvent(*candle5m, "5m")
	candleStore.QueueEvent(*candle15m, "15m")
	candleStore.QueueEvent(*candle1h, "1h")
	candleStore.QueueEvent(*candle6h, "6h")
	candleStore.QueueEvent(*candle1d, "1d")

	assert.Equal(t, true, isTransportEmpty(internalTransport["1m"]))
	assert.Equal(t, true, isTransportEmpty(internalTransport["5m"]))
	assert.Equal(t, true, isTransportEmpty(internalTransport["15m"]))
	assert.Equal(t, true, isTransportEmpty(internalTransport["1h"]))
	assert.Equal(t, true, isTransportEmpty(internalTransport["6h"]))
	assert.Equal(t, true, isTransportEmpty(internalTransport["1d"]))

	candleStore.Notify()

	candle := <- internalTransport["1m"]
	assert.Equal(t, candle.Interval, "1m")

	candle = <- internalTransport["5m"]
	assert.Equal(t, candle.Interval, "5m")

	candle = <- internalTransport["15m"]
	assert.Equal(t, candle.Interval, "15m")

	candle = <- internalTransport["1h"]
	assert.Equal(t, candle.Interval, "1h")

	candle = <- internalTransport["6h"]
	assert.Equal(t, candle.Interval, "6h")

	candle = <- internalTransport["1d"]
	assert.Equal(t, candle.Interval, "1d")

}

func isTransportEmpty(transport chan msg.Candle) bool {
	select {
	case  <- transport:
		return false
	default:
		return true
	}
}