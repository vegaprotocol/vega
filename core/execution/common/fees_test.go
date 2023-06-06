// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
