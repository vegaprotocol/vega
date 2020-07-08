package storage_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/stretchr/testify/assert"
)

func testStorage_GenerateCandles(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)
	candleStore, err := storage.NewCandles(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)
	defer candleStore.Close()

	// t0 = 2018-11-13T11:01:14Z
	t0 := vegatime.UnixNano(1542106874000000000)
	t.Log(fmt.Sprintf("t0 = %s", vegatime.Format(t0)))

	var trades = []*types.Trade{
		{Type: types.Trade_TYPE_DEFAULT, Id: "1", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "2", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(20 * time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "3", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(1 * time.Minute).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "4", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 20*time.Second).UnixNano()},
	}

	sub := subscribers.NewCandleSub(ctx, candleStore, true)
	sub.Push(events.NewMarketEvent(ctx, types.Market{Id: testMarket}))
	tEvt := events.NewTime(ctx, t0)
	sub.Push(tEvt)

	for idx := range trades {
		sub.Push(events.NewTradeEvent(ctx, *trades[idx]))
	}
	sub.Push(tEvt)
	sub.Push(tEvt)

	currentVegaTime := t0.Add(2 * time.Minute)
	tEvt = events.NewTime(ctx, currentVegaTime)
	sub.Push(tEvt)
	sub.Push(tEvt)

	candles, err := candleStore.GetCandles(ctx, testMarket, t0, types.Interval_INTERVAL_I1M)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))
	assert.Nil(t, err)

	if assert.Equal(t, 2, len(candles)) {
		t.Log(vegatime.Format(time.Unix(1542106860, 000000000)))
		assert.Equal(t, int64(1542106860000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(200), candles[0].Volume)

		assert.Equal(t, int64(1542106920000000000), candles[1].Timestamp)
		assert.Equal(t, uint64(100), candles[1].High)
		assert.Equal(t, uint64(100), candles[1].Low)
		assert.Equal(t, uint64(100), candles[1].Open)
		assert.Equal(t, uint64(100), candles[1].Close)
		assert.Equal(t, uint64(200), candles[1].Volume)
	}

	candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(1*time.Minute), types.Interval_INTERVAL_I1M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	if assert.Equal(t, 1, len(candles)) {
		assert.Equal(t, int64(1542106920000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(200), candles[0].Volume)
	}

	candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(1*time.Minute), types.Interval_INTERVAL_I5M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

	assert.Equal(t, 1, len(candles))
	assert.Equal(t, int64(1542106800000000000), candles[0].Timestamp)
	assert.Equal(t, uint64(100), candles[0].High)
	assert.Equal(t, uint64(100), candles[0].Low)
	assert.Equal(t, uint64(100), candles[0].Open)
	assert.Equal(t, uint64(100), candles[0].Close)
	assert.Equal(t, uint64(400), candles[0].Volume)

	//------------------- generate empty candles-------------------------//
	// currentVegaTime := t0.Add(2 * time.Minute)
	assert.Nil(t, err)
	currentVegaTime = t0.Add(17 * time.Minute)
	// send it twice, the internal buffer means we need to sync up
	tEvt = events.NewTime(ctx, currentVegaTime)
	sub.Push(tEvt)
	sub.Push(tEvt)

	candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_INTERVAL_I1M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	if assert.Equal(t, 3, len(candles)) {
		assert.Equal(t, int64(1542106860000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(200), candles[0].Volume)

		assert.Equal(t, int64(1542106920000000000), candles[1].Timestamp)
		assert.Equal(t, uint64(100), candles[1].High)
		assert.Equal(t, uint64(100), candles[1].Low)
		assert.Equal(t, uint64(100), candles[1].Open)
		assert.Equal(t, uint64(100), candles[1].Close)
		assert.Equal(t, uint64(200), candles[1].Volume)

		assert.Equal(t, int64(1542106980000000000), candles[2].Timestamp)
		assert.Equal(t, uint64(100), candles[2].High)
		assert.Equal(t, uint64(100), candles[2].Low)
		assert.Equal(t, uint64(100), candles[2].Open)
		assert.Equal(t, uint64(100), candles[2].Close)
		assert.Equal(t, uint64(0), candles[2].Volume)
	}

	candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_INTERVAL_I5M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

	if assert.Equal(t, 1, len(candles)) {
		assert.Equal(t, int64(1542106800000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)
	}

	candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(2*time.Minute), types.Interval_INTERVAL_I15M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

	if assert.Equal(t, 1, len(candles)) {
		assert.Equal(t, int64(1542106800000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)
	}

	candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(17*time.Minute), types.Interval_INTERVAL_I15M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

	assert.Equal(t, 0, len(candles))

	currentVegaTime = t0.Add(20 * time.Minute)
	// send it twice, the internal buffer means we need to sync up
	tEvt = events.NewTime(ctx, currentVegaTime)
	sub.Push(tEvt)
	sub.Push(tEvt)

	candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(17*time.Minute), types.Interval_INTERVAL_I15M)
	assert.Nil(t, err)
	t.Log(fmt.Sprintf("Candles fetched for t0 and 15m: %+v", candles))

	if assert.Equal(t, 1, len(candles)) {
		assert.Equal(t, int64(1542107700000000000), candles[0].Timestamp)
		assert.Equal(t, uint64(100), candles[0].High)
		assert.Equal(t, uint64(100), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(100), candles[0].Close)
		assert.Equal(t, uint64(0), candles[0].Volume)
	}

}

func TestStorage_GetMapOfIntervalsToTimestamps(t *testing.T) {
	timestamp, _ := vegatime.Parse("2018-11-13T11:01:14Z")
	t0 := timestamp
	timestamps := subscribers.GetMapOfIntervalsToRoundedTimestamps(timestamp)
	assert.Equal(t, t0.Add(-14*time.Second), timestamps[types.Interval_INTERVAL_I1M])
	assert.Equal(t, t0.Add(-(time.Minute + 14*time.Second)), timestamps[types.Interval_INTERVAL_I5M])
	assert.Equal(t, t0.Add(-(time.Minute + 14*time.Second)), timestamps[types.Interval_INTERVAL_I15M])
	assert.Equal(t, t0.Add(-(time.Minute + 14*time.Second)), timestamps[types.Interval_INTERVAL_I1H])
	assert.Equal(t, t0.Add(-(5*time.Hour + time.Minute + 14*time.Second)), timestamps[types.Interval_INTERVAL_I6H])
	assert.Equal(t, t0.Add(-(11*time.Hour + time.Minute + 14*time.Second)), timestamps[types.Interval_INTERVAL_I1D])
}

func TestStorage_SubscribeUnsubscribeCandles(t *testing.T) {
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)
	candleStore, err := storage.NewCandles(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)
	defer candleStore.Close()

	internalTransport1 := &storage.InternalTransport{
		Market:    testMarket,
		Interval:  types.Interval_INTERVAL_I1M,
		Transport: make(chan *types.Candle)}
	ref := candleStore.Subscribe(internalTransport1)
	assert.Equal(t, uint64(1), ref)

	internalTransport2 := &storage.InternalTransport{
		Market:    testMarket,
		Interval:  types.Interval_INTERVAL_I1M,
		Transport: make(chan *types.Candle)}
	ref = candleStore.Subscribe(internalTransport2)
	assert.Equal(t, uint64(2), ref)

	err = candleStore.Unsubscribe(1)
	assert.Nil(t, err)

	err = candleStore.Unsubscribe(1)
	assert.Equal(t, "subscriber to Candle store does not exist with id: 1", err.Error())

	err = candleStore.Unsubscribe(2)
	assert.Nil(t, err)

	err = candleStore.Unsubscribe(2)
	assert.Nil(t, err)
}

func testStorage_PreviousCandleDerivedValues(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	config, err := storage.NewTestConfig()
	if err != nil {
		t.Fatalf("unable to setup badger dirs: %v", err)
	}

	storage.FlushStores(logging.NewTestLogger(), config)
	candleStore, err := storage.NewCandles(logging.NewTestLogger(), config, func() {})
	assert.Nil(t, err)
	defer candleStore.Close()

	// t0 = 2018-11-13T11:00:00Z
	t0 := vegatime.UnixNano(1542106800000000000)

	var trades1 = []*types.Trade{
		{Type: types.Trade_TYPE_DEFAULT, Id: "1", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "2", MarketID: testMarket, Price: uint64(99), Size: uint64(100), Timestamp: t0.Add(10 * time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "3", MarketID: testMarket, Price: uint64(108), Size: uint64(100), Timestamp: t0.Add(20 * time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "4", MarketID: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0.Add(30 * time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "5", MarketID: testMarket, Price: uint64(110), Size: uint64(100), Timestamp: t0.Add(1 * time.Minute).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "6", MarketID: testMarket, Price: uint64(112), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 10*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "7", MarketID: testMarket, Price: uint64(113), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 20*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "8", MarketID: testMarket, Price: uint64(109), Size: uint64(100), Timestamp: t0.Add(1*time.Minute + 30*time.Second).UnixNano()},
	}

	sub := subscribers.NewCandleSub(ctx, candleStore, true)
	sub.Push(events.NewMarketEvent(ctx, types.Market{Id: testMarket}))
	tEvt := events.NewTime(ctx, t0)
	sub.Push(tEvt)
	for idx := range trades1 {
		sub.Push(events.NewTradeEvent(ctx, *trades1[idx]))
	}
	tEvt = events.NewTime(ctx, t0.Add(2*time.Minute))
	sub.Push(tEvt)
	sub.Push(tEvt)

	candles, err := candleStore.GetCandles(ctx, testMarket, t0, types.Interval_INTERVAL_I1M)
	assert.Nil(t, err)

	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	t.Log(vegatime.Format(time.Unix(1542106860, 000000000)))

	if assert.Equal(t, 2, len(candles)) {
		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(108), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(105), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[1].Timestamp)
		assert.Equal(t, uint64(113), candles[1].High)
		assert.Equal(t, uint64(109), candles[1].Low)
		assert.Equal(t, uint64(110), candles[1].Open)
		assert.Equal(t, uint64(109), candles[1].Close)
		assert.Equal(t, uint64(400), candles[1].Volume)
	}

	candles, err = candleStore.GetCandles(ctx, testMarket, t0.Add(1*time.Minute), types.Interval_INTERVAL_I1M)
	assert.Nil(t, err)

	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	if assert.Equal(t, 1, len(candles)) {
		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(113), candles[0].High)
		assert.Equal(t, uint64(109), candles[0].Low)
		assert.Equal(t, uint64(110), candles[0].Open)
		assert.Equal(t, uint64(109), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)
	}

	candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_INTERVAL_I5M)
	assert.Nil(t, err)

	t.Log(fmt.Sprintf("Candles fetched for t0 and 5m: %+v", candles))

	if assert.Equal(t, 1, len(candles)) {
		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(113), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(109), candles[0].Close)
		assert.Equal(t, uint64(800), candles[0].Volume)
	}

	var trades2 = []*types.Trade{
		{Type: types.Trade_TYPE_DEFAULT, Id: "9", MarketID: testMarket, Price: uint64(100), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 10*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "10", MarketID: testMarket, Price: uint64(99), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 20*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "11", MarketID: testMarket, Price: uint64(108), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 30*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "12", MarketID: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0.Add(2*time.Minute + 40*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "13", MarketID: testMarket, Price: uint64(110), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 10*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "14", MarketID: testMarket, Price: uint64(112), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 20*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "15", MarketID: testMarket, Price: uint64(113), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 30*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "16", MarketID: testMarket, Price: uint64(109), Size: uint64(100), Timestamp: t0.Add(3*time.Minute + 40*time.Second).UnixNano()},
	}

	assert.Nil(t, err)
	for idx := range trades2 {
		sub.Push(events.NewTradeEvent(ctx, *trades2[idx]))
	}
	tEvt = events.NewTime(ctx, t0.Add(4*time.Minute))
	sub.Push(tEvt)
	sub.Push(tEvt)

	candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_INTERVAL_I1M)
	assert.Nil(t, err)

	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	if assert.Equal(t, 4, len(candles)) {
		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(108), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(105), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[1].Timestamp)
		assert.Equal(t, uint64(113), candles[1].High)
		assert.Equal(t, uint64(109), candles[1].Low)
		assert.Equal(t, uint64(110), candles[1].Open)
		assert.Equal(t, uint64(109), candles[1].Close)
		assert.Equal(t, uint64(400), candles[1].Volume)

		assert.Equal(t, t0.Add(2*time.Minute).UnixNano(), candles[2].Timestamp)
		assert.Equal(t, uint64(108), candles[2].High)
		assert.Equal(t, uint64(99), candles[2].Low)
		assert.Equal(t, uint64(100), candles[2].Open)
		assert.Equal(t, uint64(105), candles[2].Close)
		assert.Equal(t, uint64(400), candles[2].Volume)

		assert.Equal(t, t0.Add(3*time.Minute).UnixNano(), candles[3].Timestamp)
		assert.Equal(t, uint64(113), candles[3].High)
		assert.Equal(t, uint64(109), candles[3].Low)
		assert.Equal(t, uint64(110), candles[3].Open)
		assert.Equal(t, uint64(109), candles[3].Close)
		assert.Equal(t, uint64(400), candles[3].Volume)
	}

	var trades3 = []*types.Trade{
		{Type: types.Trade_TYPE_DEFAULT, Id: "17", MarketID: testMarket, Price: uint64(95), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 10*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "18", MarketID: testMarket, Price: uint64(80), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 20*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "19", MarketID: testMarket, Price: uint64(120), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 30*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "20", MarketID: testMarket, Price: uint64(105), Size: uint64(100), Timestamp: t0.Add(4*time.Minute + 40*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "21", MarketID: testMarket, Price: uint64(103), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 10*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "22", MarketID: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 20*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "23", MarketID: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 30*time.Second).UnixNano()},
		{Type: types.Trade_TYPE_DEFAULT, Id: "24", MarketID: testMarket, Price: uint64(101), Size: uint64(100), Timestamp: t0.Add(5*time.Minute + 40*time.Second).UnixNano()},
	}

	assert.Nil(t, err)
	for idx := range trades3 {
		sub.Push(events.NewTradeEvent(ctx, *trades3[idx]))
	}
	tEvt = events.NewTime(ctx, t0.Add(10*time.Minute))
	sub.Push(tEvt)
	sub.Push(tEvt)

	candles, err = candleStore.GetCandles(ctx, testMarket, t0, types.Interval_INTERVAL_I1M)
	assert.Nil(t, err)

	t.Log(fmt.Sprintf("Candles fetched for t0 and 1m: %+v", candles))

	if assert.Equal(t, 6, len(candles)) {
		assert.Equal(t, t0.UnixNano(), candles[0].Timestamp)
		assert.Equal(t, uint64(108), candles[0].High)
		assert.Equal(t, uint64(99), candles[0].Low)
		assert.Equal(t, uint64(100), candles[0].Open)
		assert.Equal(t, uint64(105), candles[0].Close)
		assert.Equal(t, uint64(400), candles[0].Volume)

		assert.Equal(t, t0.Add(1*time.Minute).UnixNano(), candles[1].Timestamp)
		assert.Equal(t, uint64(113), candles[1].High)
		assert.Equal(t, uint64(109), candles[1].Low)
		assert.Equal(t, uint64(110), candles[1].Open)
		assert.Equal(t, uint64(109), candles[1].Close)
		assert.Equal(t, uint64(400), candles[1].Volume)

		assert.Equal(t, t0.Add(2*time.Minute).UnixNano(), candles[2].Timestamp)
		assert.Equal(t, uint64(108), candles[2].High)
		assert.Equal(t, uint64(99), candles[2].Low)
		assert.Equal(t, uint64(100), candles[2].Open)
		assert.Equal(t, uint64(105), candles[2].Close)
		assert.Equal(t, uint64(400), candles[2].Volume)

		assert.Equal(t, t0.Add(3*time.Minute).UnixNano(), candles[3].Timestamp)
		assert.Equal(t, uint64(113), candles[3].High)
		assert.Equal(t, uint64(109), candles[3].Low)
		assert.Equal(t, uint64(110), candles[3].Open)
		assert.Equal(t, uint64(109), candles[3].Close)
		assert.Equal(t, uint64(400), candles[3].Volume)

		assert.Equal(t, t0.Add(4*time.Minute).UnixNano(), candles[4].Timestamp)
		assert.Equal(t, uint64(120), candles[4].High)
		assert.Equal(t, uint64(80), candles[4].Low)
		assert.Equal(t, uint64(95), candles[4].Open)
		assert.Equal(t, uint64(105), candles[4].Close)
		assert.Equal(t, uint64(400), candles[4].Volume)

		assert.Equal(t, t0.Add(5*time.Minute).UnixNano(), candles[5].Timestamp)
		assert.Equal(t, uint64(103), candles[5].High)
		assert.Equal(t, uint64(101), candles[5].Low)
		assert.Equal(t, uint64(103), candles[5].Open)
		assert.Equal(t, uint64(101), candles[5].Close)
		assert.Equal(t, uint64(400), candles[5].Volume)
	}
}
