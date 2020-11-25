package storage_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/stretchr/testify/assert"
)

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
