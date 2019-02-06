package storage

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"fmt"
	"time"
)

func TestStorage_GenerateCandles(t *testing.T) {
	config := NewTestConfig()
	FlushStores(config)
	candleStore, err := NewCandleStore(config)
	assert.Nil(t, err)
	defer candleStore.Close()

	// t0 = 2018-11-13T11:01:14Z
	t0 := uint64(1542106874000000000)
	t0Seconds := int64(1542106874)
	t0NanoSeconds := int64(000000000)

	t.Log(fmt.Sprintf("t0 = %s", time.Unix(t0Seconds, t0NanoSeconds).Format(time.RFC3339)))

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

	candles, err := candleStore.GetCandles(testMarket, t0, msg.Interval_I1M)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))
	assert.Nil(t, err)

	assert.Equal(t, 2, len(candles))
	t.Log(fmt.Sprintf("%s", time.Unix(1542106860,000000000).Format(time.RFC3339)))
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

	candles, err = candleStore.GetCandles(testMarket, t0 + uint64(1 * time.Minute), msg.Interval_I1M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106920000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(200), candles[0].Volume)

	candles, err = candleStore.GetCandles(testMarket, t0 + uint64(1 * time.Minute), msg.Interval_I5M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)
	
	//------------------- generate empty candles-------------------------//

	currentVegaTime := uint64(t0) + uint64(2 * time.Minute)
	candleStore.StartNewBuffer(testMarket, currentVegaTime)
	candleStore.GenerateCandlesFromBuffer(testMarket)

	candles, err = candleStore.GetCandles(testMarket, t0, msg.Interval_I1M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

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


	candles, err = candleStore.GetCandles(testMarket, t0, msg.Interval_I5M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	candles, err = candleStore.GetCandles(testMarket, t0 + uint64(2 * time.Minute), msg.Interval_I15M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)


	candles, err = candleStore.GetCandles(testMarket, t0 + uint64(17 * time.Minute), msg.Interval_I15M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

	assert.Equal(t, 0, len(candles))

	currentVegaTime = uint64(t0) + uint64(17 * time.Minute)
	candleStore.StartNewBuffer(testMarket, currentVegaTime)
	candleStore.GenerateCandlesFromBuffer(testMarket)

	candles, err = candleStore.GetCandles(testMarket, t0 + uint64(17 * time.Minute), msg.Interval_I15M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, uint64(1542107700000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(0), candles[0].Volume)
}

func TestStorage_GetMapOfIntervalsToTimestamps(t *testing.T) {
	timestamp, _ := time.Parse(time.RFC3339, "2018-11-13T11:01:14Z")
	t0 := uint64(timestamp.UnixNano())
	timestamps := getMapOfIntervalsToRoundedTimestamps(uint64(timestamp.UnixNano()))
	assert.Equal(t, t0 - uint64(14 * time.Second), timestamps[msg.Interval_I1M])
	assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), timestamps[msg.Interval_I5M])
	assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), timestamps[msg.Interval_I15M])
	assert.Equal(t, t0 - uint64(time.Minute + 14 * time.Second), timestamps[msg.Interval_I1H])
	assert.Equal(t, t0 - uint64(5 * time.Hour + time.Minute + 14 * time.Second), timestamps[msg.Interval_I6H])
	assert.Equal(t, t0 - uint64(11 * time.Hour + time.Minute + 14 * time.Second), timestamps[msg.Interval_I1D])
}

func TestStorage_SubscribeUnsubscribeCandles(t *testing.T) {
	config := NewTestConfig()
	FlushStores(config)
	candleStore, err := NewCandleStore(config)
	assert.Nil(t, err)
	defer candleStore.Close()

	internalTransport1 := &InternalTransport{testMarket, msg.Interval_I1M, make(chan msg.Candle)}
	ref := candleStore.Subscribe(internalTransport1)
	assert.Equal(t, uint64(1), ref)

	internalTransport2 := &InternalTransport{testMarket, msg.Interval_I1M, make(chan msg.Candle)}
	ref = candleStore.Subscribe(internalTransport2)
	assert.Equal(t, uint64(2), ref)

	err = candleStore.Unsubscribe(1)
	assert.Nil(t, err)

	err = candleStore.Unsubscribe(1)
	assert.Equal(t, "CandleStore subscriber does not exist with id: 1", err.Error())

	err = candleStore.Unsubscribe(2)
	assert.Nil(t, err)

	err = candleStore.Unsubscribe(2)
	assert.Nil(t, err)
}


func TestStorage_PreviousCandleDerivedValues(t *testing.T) {
	config := NewTestConfig()
	FlushStores(config)
	candleStore, err := NewCandleStore(config)
	assert.Nil(t, err)
	defer candleStore.Close()

	// t0 = 2018-11-13T11:00:00Z
	t0 := uint64(1542106800000000000)

	var trades1 = []*msg.Trade{
		{Id: "1", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0},
		{Id: "2", Market: testMarket, Price: uint64(99), Size: uint64(100), Timestamp: t0 + uint64(10 * time.Second)},
		{Id: "3", Market: testMarket, Price: uint64(108), Size: uint64(100), Timestamp: t0 + uint64(20 * time.Second)},
		{Id: "4", Market: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0 + uint64(30 * time.Second)},
		{Id: "5", Market: testMarket, Price: uint64(110), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute)},
		{Id: "6", Market: testMarket, Price: uint64(112), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 10 * time.Second)},
		{Id: "7", Market: testMarket, Price: uint64(113), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 20 * time.Second)},
		{Id: "8", Market: testMarket, Price: uint64(109), Size: uint64(100), Timestamp: t0 + uint64(1 * time.Minute + 30 * time.Second)},
	}

	candleStore.StartNewBuffer(testMarket, t0)
	for idx := range trades1 {
		candleStore.AddTradeToBuffer(trades1[idx].Market, *trades1[idx])
	}
	candleStore.GenerateCandlesFromBuffer(testMarket)

	candles, err := candleStore.GetCandles(testMarket, t0, msg.Interval_I1M)
	assert.Nil(t, err)
	
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	assert.Equal(t, 2, len(candles))
	
	t.Log(fmt.Sprintf("%s", time.Unix(1542106860,000000000).Format(time.RFC3339)))

	assert.Equal(t, t0, candles[0].Timestamp)
	assert.Equal(t, uint64(108), candles[0].High)
	assert.Equal(t, uint64(99), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(105), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	assert.Equal(t, t0 + uint64(1 * time.Minute), candles[1].Timestamp)
	assert.Equal(t, uint64(113), candles[1].High)
	assert.Equal(t, uint64(109), candles[1].Low)
	assert.Equal(t, uint64(110), candles[1].Open)
	assert.Equal(t, uint64(109), candles[1].Close)
	assert.Equal(t, uint64(400), candles[1].Volume)

	candles, err = candleStore.GetCandles(testMarket, t0 + uint64(1 * time.Minute), msg.Interval_I1M)
	assert.Nil(t, err)
	
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, t0 + uint64(1 * time.Minute), candles[0].Timestamp)
	assert.Equal(t, uint64(113), candles[0].High)
	assert.Equal(t, uint64(109), candles[0].Low)
	assert.Equal(t, uint64(110), candles[0].Open)
	assert.Equal(t, uint64(109), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	candles, err = candleStore.GetCandles(testMarket, t0, msg.Interval_I5M)
	assert.Nil(t, err)

	t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, t0, candles[0].Timestamp)
	assert.Equal(t, uint64(113), candles[0].High)
	assert.Equal(t, uint64(99), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(109), candles[0].Close)
	assert.Equal(t, uint64(800), candles[0].Volume)

	var trades2 = []*msg.Trade{
		{Id: "9", Market: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute + 10 * time.Second)},
		{Id: "10", Market: testMarket, Price: uint64(99), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute + 20 * time.Second)},
		{Id: "11", Market: testMarket, Price: uint64(108), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute + 30 * time.Second)},
		{Id: "12", Market: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0 + uint64(2 * time.Minute + 40 * time.Second)},
		{Id: "13", Market: testMarket, Price: uint64(110), Size: uint64(100), Timestamp: t0 + uint64(3 * time.Minute + 10 * time.Second)},
		{Id: "14", Market: testMarket, Price: uint64(112), Size: uint64(100), Timestamp: t0 + uint64(3 * time.Minute + 20 * time.Second)},
		{Id: "15", Market: testMarket, Price: uint64(113), Size: uint64(100), Timestamp: t0 + uint64(3 * time.Minute + 30 * time.Second)},
		{Id: "16", Market: testMarket, Price: uint64(109), Size: uint64(100), Timestamp: t0 + uint64(3 * time.Minute + 40 * time.Second)},
	}

	candleStore.StartNewBuffer(testMarket, t0 + uint64(2 * time.Minute))
	for idx := range trades2 {
		candleStore.AddTradeToBuffer(trades2[idx].Market, *trades2[idx])
	}
	candleStore.GenerateCandlesFromBuffer(testMarket)

	candles, err = candleStore.GetCandles(testMarket, t0, msg.Interval_I1M)
	assert.Nil(t, err)

	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	assert.Equal(t, t0, candles[0].Timestamp)
	assert.Equal(t, uint64(108), candles[0].High)
	assert.Equal(t, uint64(99), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(105), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	assert.Equal(t, t0 + uint64(1 * time.Minute), candles[1].Timestamp)
	assert.Equal(t, uint64(113), candles[1].High)
	assert.Equal(t, uint64(109), candles[1].Low)
	assert.Equal(t, uint64(110), candles[1].Open)
	assert.Equal(t, uint64(109), candles[1].Close)
	assert.Equal(t, uint64(400), candles[1].Volume)

	assert.Equal(t, t0 + uint64(2 * time.Minute), candles[2].Timestamp)
	assert.Equal(t, uint64(108), candles[2].High)
	assert.Equal(t, uint64(99), candles[2].Low)
	assert.Equal(t, uint64(100), candles[2].Open)
	assert.Equal(t, uint64(105), candles[2].Close)
	assert.Equal(t, uint64(400), candles[2].Volume)

	assert.Equal(t, t0 + uint64(3 * time.Minute), candles[3].Timestamp)
	assert.Equal(t, uint64(113), candles[3].High)
	assert.Equal(t, uint64(109), candles[3].Low)
	assert.Equal(t, uint64(110), candles[3].Open)
	assert.Equal(t, uint64(109), candles[3].Close)
	assert.Equal(t, uint64(400), candles[3].Volume)

	var trades3 = []*msg.Trade{
		{Id: "17", Market: testMarket, Price: uint64(95), Size: uint64(100), Timestamp: t0 + uint64(4 * time.Minute + 10 * time.Second)},
		{Id: "18", Market: testMarket, Price: uint64(80), Size: uint64(100), Timestamp: t0 + uint64(4 * time.Minute + 20 * time.Second)},
		{Id: "19", Market: testMarket, Price: uint64(120), Size: uint64(100), Timestamp: t0 + uint64(4 * time.Minute + 30 * time.Second)},
		{Id: "20", Market: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0 + uint64(4 * time.Minute + 40 * time.Second)},
		{Id: "21", Market: testMarket, Price: uint64(103), Size: uint64(100), Timestamp: t0 + uint64(5 * time.Minute + 10 * time.Second)},
		{Id: "22", Market: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0 + uint64(5 * time.Minute + 20 * time.Second)},
		{Id: "23", Market: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0 + uint64(5 * time.Minute + 30 * time.Second)},
		{Id: "24", Market: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0 + uint64(5 * time.Minute + 40 * time.Second)},
	}

	candleStore.StartNewBuffer(testMarket, t0 + uint64(4 * time.Minute))
	for idx := range trades3 {
		candleStore.AddTradeToBuffer(trades3[idx].Market, *trades3[idx])
	}
	candleStore.GenerateCandlesFromBuffer(testMarket)
	
	candles, err = candleStore.GetCandles(testMarket, t0, msg.Interval_I1M)
	assert.Nil(t, err)
	
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	assert.Equal(t, t0, candles[0].Timestamp)
	assert.Equal(t, uint64(108), candles[0].High)
	assert.Equal(t, uint64(99), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(105), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	assert.Equal(t, t0 + uint64(1 * time.Minute), candles[1].Timestamp)
	assert.Equal(t, uint64(113), candles[1].High)
	assert.Equal(t, uint64(109), candles[1].Low)
	assert.Equal(t, uint64(110), candles[1].Open)
	assert.Equal(t, uint64(109), candles[1].Close)
	assert.Equal(t, uint64(400), candles[1].Volume)

	assert.Equal(t, t0 + uint64(2 * time.Minute), candles[2].Timestamp)
	assert.Equal(t, uint64(108), candles[2].High)
	assert.Equal(t, uint64(99), candles[2].Low)
	assert.Equal(t, uint64(100), candles[2].Open)
	assert.Equal(t, uint64(105), candles[2].Close)
	assert.Equal(t, uint64(400), candles[2].Volume)

	assert.Equal(t, t0 + uint64(3 * time.Minute), candles[3].Timestamp)
	assert.Equal(t, uint64(113), candles[3].High)
	assert.Equal(t, uint64(109), candles[3].Low)
	assert.Equal(t, uint64(110), candles[3].Open)
	assert.Equal(t, uint64(109), candles[3].Close)
	assert.Equal(t, uint64(400), candles[3].Volume)

	assert.Equal(t, t0 + uint64(4 * time.Minute), candles[4].Timestamp)
	assert.Equal(t, uint64(120), candles[4].High)
	assert.Equal(t, uint64(80), candles[4].Low)
	assert.Equal(t, uint64(95), candles[4].Open)
	assert.Equal(t, uint64(105), candles[4].Close)
	assert.Equal(t, uint64(400), candles[4].Volume)

	assert.Equal(t, t0 + uint64(5 * time.Minute), candles[5].Timestamp)
	assert.Equal(t, uint64(103), candles[5].High)
	assert.Equal(t, uint64(101), candles[5].Low)
	assert.Equal(t, uint64(103), candles[5].Open)
	assert.Equal(t, uint64(101), candles[5].Close)
	assert.Equal(t, uint64(400), candles[5].Volume)
}