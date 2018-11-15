package datastore

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"fmt"
	"os"
	"time"
)

func TestCandleGenerator_AddTrade(t *testing.T) {
	trade := &msg.Trade{Id: "1", Price: uint64(100), Size: uint64(100), Timestamp: uint64(1542106874000000000)}
	testMarket := "testMarket"

	candleStore := NewCandleStore(testMarket)
	defer candleStore.Close()

	candleStore.AddTrade(trade)

	assert.Equal(t, 1, len(candleStore.tradesBuffer))
}

func flushCandleStore() {
	err := os.RemoveAll("./candleStore")
	if err != nil {
		fmt.Printf("UNABLE TO FLUSH DB: %s\n", err.Error())
	}
}

func TestCandleGenerator_Generate(t *testing.T) {
	testMarket := "testMarket"
	flushCandleStore()
	candleStore := NewCandleStore(testMarket)
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
		candleStore.AddTrade(trades[idx])
	}

	assert.Equal(t, 4, len(candleStore.tradesBuffer))

	vegaTimeAccessor := vegaTimeAccessor{}
	candleStore.Generate(vegaTimeAccessor)

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

	vegaTimeAccessor.currentVegaTime = int64(t0) + int64(2 * time.Minute)
	candleStore.Generate(vegaTimeAccessor)
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

	vegaTimeAccessor.currentVegaTime = int64(t0) + int64(17 * time.Minute)
	candleStore.Generate(vegaTimeAccessor)

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