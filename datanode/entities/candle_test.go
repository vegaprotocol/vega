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

package entities_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestProtoFromCandle(t *testing.T) {
	periodStart := time.Now()
	lastUpdate := periodStart.Add(5 * time.Microsecond)
	candle := entities.Candle{
		PeriodStart:        periodStart,
		LastUpdateInPeriod: lastUpdate,
		Open:               decimal.NewFromInt(4),
		Close:              decimal.NewFromInt(5),
		High:               decimal.NewFromInt(6),
		Low:                decimal.NewFromInt(7),
		Volume:             30,
	}

	p, err := candle.ToV1CandleProto(vega.Interval_INTERVAL_I6H)
	if err != nil {
		t.Fatalf("failed to conver proto to candle:%s", err)
	}

	assert.Equal(t, periodStart.UnixNano(), p.Timestamp)
	assert.Equal(t, lastUpdate.Format(time.RFC3339Nano), p.Datetime)
	assert.Equal(t, "4", p.Open)
	assert.Equal(t, "5", p.Close)
	assert.Equal(t, "6", p.High)
	assert.Equal(t, "7", p.Low)
	assert.Equal(t, uint64(30), p.Volume)
	assert.Equal(t, vega.Interval_INTERVAL_I6H, p.Interval)
}
