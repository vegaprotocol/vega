// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package candlesv2_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/entities"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

type testCandleSource struct {
	candles chan []entities.Candle
}

func (t *testCandleSource) GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time,
	p entities.CursorPagination,
) ([]entities.Candle, entities.PageInfo, error) {
	pageInfo := entities.PageInfo{}
	select {
	case c := <-t.candles:
		return c, pageInfo, nil
	default:
		return nil, pageInfo, nil
	}
}

func TestSubscribe(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(1).CandleUpdates)
	startTime := time.Now()

	_, out1 := updates.Subscribe()
	_, out2 := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	candle2 := <-out2
	assert.Equal(t, expectedCandle, candle2)

	expectedCandle = createCandle(startTime.Add(1*time.Minute), startTime.Add(1*time.Minute), 2, 2, 2, 2, 20)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 = <-out1
	assert.Equal(t, expectedCandle, candle1)

	candle2 = <-out2
	assert.Equal(t, expectedCandle, candle2)
}

func TestUnsubscribe(t *testing.T) {
	testCandleSource := &testCandleSource{candles: make(chan []entities.Candle)}

	updates := candlesv2.NewCandleUpdates(context.Background(), logging.NewTestLogger(), "testCandles",
		testCandleSource, newTestCandleConfig(1).CandleUpdates)
	startTime := time.Now()

	id, out1 := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10)
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
		testCandleSource, newTestCandleConfig(1).CandleUpdates)
	startTime := time.Now()

	_, out1 := updates.Subscribe()

	expectedCandle := createCandle(startTime, startTime, 1, 1, 1, 1, 10)
	testCandleSource.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	_, out2 := updates.Subscribe()
	candle2 := <-out2
	assert.Equal(t, expectedCandle, candle2)
}

func newTestCandleConfig(bufferSize int) candlesv2.Config {
	conf := candlesv2.NewDefaultConfig()
	conf.CandleUpdates = candlesv2.CandleUpdatesConfig{
		CandleUpdatesStreamBufferSize: bufferSize,
		CandleUpdatesStreamInterval:   encoding.Duration{Duration: 1 * time.Microsecond},
		CandlesFetchTimeout:           encoding.Duration{Duration: 2 * time.Minute},
	}

	return conf
}

func createCandle(periodStart time.Time, lastUpdate time.Time, open int, close int, high int, low int, volume int) entities.Candle {
	return entities.Candle{
		PeriodStart:        periodStart,
		LastUpdateInPeriod: lastUpdate,
		Open:               decimal.NewFromInt(int64(open)),
		Close:              decimal.NewFromInt(int64(close)),
		High:               decimal.NewFromInt(int64(high)),
		Low:                decimal.NewFromInt(int64(low)),
		Volume:             uint64(volume),
	}
}
