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

package common_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

func TestFeeSplitter(t *testing.T) {
	var (
		totalStake              = num.NewUint(100)
		timeWindowStart         = time.Now()
		marketValueWindowLength = 1 * time.Minute
	)

	tests := []struct {
		currentTime        time.Time
		tradedValue        *num.Uint
		expectedValueProxy num.Decimal
	}{
		{
			tradedValue:        num.UintZero(),
			currentTime:        timeWindowStart,
			expectedValueProxy: num.DecimalFromFloat(100),
		},
		{
			tradedValue:        num.NewUint(10),
			currentTime:        timeWindowStart.Add(10 * time.Second),
			expectedValueProxy: num.DecimalFromFloat(100),
		},
		{
			tradedValue:        num.NewUint(100),
			currentTime:        timeWindowStart.Add(30 * time.Second),
			expectedValueProxy: num.DecimalFromFloat(200),
		},
		{
			tradedValue:        num.NewUint(300),
			currentTime:        timeWindowStart.Add(3 * marketValueWindowLength),
			expectedValueProxy: num.DecimalFromFloat(300),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			fs := common.NewFeeSplitter()
			fs.TimeWindowStart(timeWindowStart)
			require.NoError(t,
				fs.SetCurrentTime(test.currentTime),
			)
			fs.AddTradeValue(test.tradedValue)

			got := fs.MarketValueProxy(marketValueWindowLength, totalStake)
			require.True(t, test.expectedValueProxy.Equal(got))
		})
	}
}

func TestFeeSplitterSnapshot(t *testing.T) {
	fs := common.NewFeeSplitter()
	require.True(t, fs.Changed())

	// reset changed flags
	fs.GetState()
	require.False(t, fs.Changed())

	// add a trade value to cause a change
	fs.AddTradeValue(num.NewUint(12))
	require.True(t, fs.Changed())

	// reset changed flag
	fs.GetState()
	require.False(t, fs.Changed())

	// set time window to cause a change
	fs.TimeWindowStart(time.Now())
	require.True(t, fs.Changed())

	currentTime := time.Now().Add(time.Minute)
	err := fs.SetCurrentTime(currentTime)
	require.NoError(t, err)

	// load state and check its the same
	snap := common.NewFeeSplitterFromSnapshot(fs.GetState(), currentTime)
	require.Equal(t, fs.Elapsed(), snap.Elapsed())
	require.Equal(t, fs.MarketValueProxy(3*time.Second, num.NewUint(5)), snap.MarketValueProxy(3*time.Second, num.NewUint(5)))
}
