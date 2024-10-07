// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package candlesv2_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nonReturningCandleSource struct{}

func (t *nonReturningCandleSource) GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
	p entities.CursorPagination,
) ([]entities.Candle, entities.PageInfo, error) {
	for {
		time.Sleep(1 * time.Second)
	}
}

type errorsAlwaysCandleSource struct{}

func (t *errorsAlwaysCandleSource) GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
	p entities.CursorPagination,
) ([]entities.Candle, entities.PageInfo, error) {
	return nil, entities.PageInfo{}, fmt.Errorf("always errors")
}

type testCandleSource struct {
	candles chan []entities.Candle
	errorCh chan error
}

func (t *testCandleSource) GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
	p entities.CursorPagination,
) ([]entities.Candle, entities.PageInfo, error) {
	pageInfo := entities.PageInfo{}
	select {
	case c := <-t.candles:
		return c, pageInfo, nil
	case err := <-t.errorCh:
		return nil, entities.PageInfo{}, err
	default:
		return nil, pageInfo, nil
	}
}

func TestSubscribeAndUnsubscribeCloseChannelPanic(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle, 3), errorCh: make(chan error)}
	// ensure the sub channels are buffered
	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(5).CandleUpdates)
	startTime := time.Now()

	updated := startTime
	firstCandle := createCandle(startTime, updated, 1, 1, 1, 1, 10, 200)
	lastCandle := firstCandle // just for the sake of types
	fCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		testCandleSource.candles <- []entities.Candle{firstCandle}
		close(fCh)
		// keep updating the most recent candle
		for i := 0; i < 3; i++ {
			updated = updated.Add(time.Second * time.Duration(i))
			lastCandle = createCandle(startTime, updated, 1, 1, 1, 1, 10, 200)
			testCandleSource.candles <- []entities.Candle{lastCandle}
		}
	}()
	<-fCh
	// ensure the first candle is sent
	sub1Id, out1, _ := updates.Subscribe()
	sub2Id, out2, _ := updates.Subscribe()

	candle1 := <-out1
	assert.Equal(t, firstCandle, candle1)

	candle2 := <-out2
	assert.Equal(t, firstCandle, candle2)

	// unsubscribe the first subscriber
	updates.Unsubscribe(sub1Id)
	// now wait for the updates:
	wg.Wait()
	sub3Id, out3, _ := updates.Subscribe()
	candle3 := <-out3
	require.Equal(t, lastCandle, candle3)
	// this should unsubscribe sub2 already
	testCandleSource.errorCh <- fmt.Errorf("transient error")

	// this sub should get instantly unsubscribed.
	errSub, eOut, _ := updates.Subscribe()
	require.NotNil(t, eOut)
	// reading from the channel should indicate it was closed already due to the error
	// once the channel is closed, the subscriber effectively has to have been removed.
	_, closed := <-eOut
	require.True(t, closed)
	// we can still safely call unsubscribe, though:
	updates.Unsubscribe(errSub)
	updates.Unsubscribe(sub2Id)
	updates.Unsubscribe(sub3Id)
}

func TestSubscribeAndUnsubscribeWhenCandleSourceErrorsAlways(t *testing.T) {
	errorsAlwaysCandleSource := &errorsAlwaysCandleSource{}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		errorsAlwaysCandleSource, newTestCandleConfig(0).CandleUpdates)

	sub1Id, _, err1 := updates.Subscribe()
	sub2Id, _, err2 := updates.Subscribe()
	require.NoError(t, err1)
	require.NoError(t, err2)

	updates.Unsubscribe(sub1Id)
	updates.Unsubscribe(sub2Id)
}

func TestUnsubscribeAfterTransientFailure(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle), errorCh: make(chan error)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(0).CandleUpdates)
	startTime := time.Now()

	sub1Id, out1, _ := updates.Subscribe()
	sub2Id, out2, _ := updates.Subscribe()

	firstCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10, 200)
	testCandleSource.candles <- []entities.Candle{firstCandle}

	candle1 := <-out1
	assert.Equal(t, firstCandle, candle1)

	candle2 := <-out2
	assert.Equal(t, firstCandle, candle2)

	testCandleSource.errorCh <- fmt.Errorf("transient error")

	updates.Unsubscribe(sub1Id)
	updates.Unsubscribe(sub2Id)
}

func TestSubscribeAfterTransientFailure(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle), errorCh: make(chan error)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(0).CandleUpdates)
	startTime := time.Now()

	_, out1, _ := updates.Subscribe()
	_, out2, _ := updates.Subscribe()

	firstCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10, 100)
	testCandleSource.candles <- []entities.Candle{firstCandle}

	candle1 := <-out1
	assert.Equal(t, firstCandle, candle1)

	candle2 := <-out2
	assert.Equal(t, firstCandle, candle2)

	testCandleSource.errorCh <- fmt.Errorf("transient error")

	_, out3, _ := updates.Subscribe()

	candle3 := <-out3
	assert.Equal(t, firstCandle, candle3)

	secondCandle := createCandle(startTime.Add(1*time.Minute), startTime.Add(1*time.Minute), 2, 2, 2, 2, 20, 100)
	testCandleSource.candles <- []entities.Candle{secondCandle}

	candle1 = <-out1
	assert.Equal(t, secondCandle, candle1)

	candle2 = <-out2
	assert.Equal(t, secondCandle, candle2)

	candle3 = <-out3
	assert.Equal(t, secondCandle, candle3)
}

func TestSubscribe(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(0).CandleUpdates)
	startTime := time.Now()

	_, out1, _ := updates.Subscribe()
	_, out2, _ := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10, 100)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	candle2 := <-out2
	assert.Equal(t, expectedCandle, candle2)

	expectedCandle = createCandle(startTime.Add(1*time.Minute), startTime.Add(1*time.Minute), 2, 2, 2, 2, 20, 100)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 = <-out1
	assert.Equal(t, expectedCandle, candle1)

	candle2 = <-out2
	assert.Equal(t, expectedCandle, candle2)
}

func TestUnsubscribe(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(0).CandleUpdates)
	startTime := time.Now()

	id, out1, _ := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10, 100)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	updates.Unsubscribe(id)

	_, ok := <-out1
	assert.False(t, ok, "candle should be closed")
}

func TestNewSubscriberAlwaysGetsLastCandle(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(0).CandleUpdates)
	startTime := time.Now()

	_, out1, _ := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10, 100)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	_, out2, _ := updates.Subscribe()
	candle2 := <-out2
	assert.Equal(t, expectedCandle, candle2)
}

func TestSubscribeWithNonZeroSubscribeBuffer(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(100).CandleUpdates)
	startTime := time.Now()

	_, out1, _ := updates.Subscribe()
	_, out2, _ := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10, 100)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	candle2 := <-out2
	assert.Equal(t, expectedCandle, candle2)

	expectedCandle = createCandle(startTime.Add(1*time.Minute), startTime.Add(1*time.Minute), 2, 2, 2, 2, 20, 100)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 = <-out1
	assert.Equal(t, expectedCandle, candle1)

	candle2 = <-out2
	assert.Equal(t, expectedCandle, candle2)
}

func TestUnsubscribeWithNonZeroSubscribeBuffer(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(100).CandleUpdates)
	startTime := time.Now()

	id, out1, _ := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10, 100)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	updates.Unsubscribe(id)

	_, ok := <-out1
	assert.False(t, ok, "candle should be closed")
}

func TestSubscribeAndUnSubscribeWithNonReturningSource(t *testing.T) {
	testCandleSource := &nonReturningCandleSource{}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(100).CandleUpdates)

	subID1, _, _ := updates.Subscribe()
	subID2, _, _ := updates.Subscribe()

	updates.Unsubscribe(subID1)
	updates.Unsubscribe(subID2)
}

func TestMultipleSlowConsumers(t *testing.T) {
	nSends := 100
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle, nSends), errorCh: make(chan error)}
	// ensure the sub channels are buffered
	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(5).CandleUpdates)
	startTime := time.Now()

	updated := startTime
	firstCandle := createCandle(startTime, updated, 1, 1, 1, 1, 10, 200)
	lastCandle := firstCandle // just for the sake of types

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		testCandleSource.candles <- []entities.Candle{firstCandle}

		// keep updating the most recent candle
		for i := 0; i < nSends; i++ {
			updated = updated.Add(time.Second * time.Duration(i))
			lastCandle = createCandle(startTime, updated, 1, 1, 1, 1, 10, 200)
			testCandleSource.candles <- []entities.Candle{lastCandle, lastCandle, lastCandle}
		}
	}()
	// ensure the first candle is sent
	_, _, _ = updates.Subscribe()
	_, _, _ = updates.Subscribe()
	wg.Wait()
}

func newTestCandleConfig(subscribeBufferSize int) candlesv2.Config {
	conf := candlesv2.NewDefaultConfig()
	conf.CandleUpdates = candlesv2.CandleUpdatesConfig{
		CandleUpdatesStreamBufferSize:                1,
		CandleUpdatesStreamInterval:                  encoding.Duration{Duration: 1 * time.Microsecond},
		CandlesFetchTimeout:                          encoding.Duration{Duration: 2 * time.Minute},
		CandleUpdatesStreamSubscriptionMsgBufferSize: subscribeBufferSize,
	}

	return conf
}

func createCandle(periodStart time.Time, lastUpdate time.Time, open int, close int, high int, low int, volume, notional int) entities.Candle {
	return entities.Candle{
		PeriodStart:        periodStart,
		LastUpdateInPeriod: lastUpdate,
		Open:               decimal.NewFromInt(int64(open)),
		Close:              decimal.NewFromInt(int64(close)),
		High:               decimal.NewFromInt(int64(high)),
		Low:                decimal.NewFromInt(int64(low)),
		Volume:             uint64(volume),
		Notional:           uint64(notional),
	}
}
