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

package spot_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/liquidity/target/spot"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

var (
	now      = time.Date(2020, 10, 30, 9, 0, 0, 0, time.UTC)
	marketID = "market-1"
)

func TestConstructor(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := spot.NewEngine(params, marketID, num.DecimalFromFloat(1))

	require.NotNil(t, engine)
}

func TestTotalStake(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := spot.NewEngine(params, marketID, num.DecimalFromFloat(1))
	err := engine.RecordTotalStake(9, now)
	require.NoError(t, err)
	err = engine.RecordTotalStake(0, now)
	require.NoError(t, err)
	err = engine.RecordTotalStake(11, now.Add(time.Nanosecond))
	require.NoError(t, err)
	err = engine.RecordTotalStake(12, now.Add(time.Nanosecond))
	require.NoError(t, err)
	err = engine.RecordTotalStake(13, now.Add(-2*time.Nanosecond))
	require.Error(t, err)
}

func TestGetTargetStake_NoRecordedTotalStake(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := spot.NewEngine(params, marketID, num.DecimalFromFloat(1))
	targetStake := engine.GetTargetStake(now)
	require.Equal(t, num.UintZero(), targetStake)
}

func TestGetTargetStake_VerifyMaxOI(t *testing.T) {
	tWindow := 60 * time.Minute
	scalingFactor := num.DecimalFromFloat(0.25)
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	expectedTargetStake := func(oi uint64) *num.Uint {
		mp := num.NewUint(oi).ToDecimal().Mul(scalingFactor)
		ump, _ := num.UintFromDecimal(mp)
		return ump
	}

	engine := spot.NewEngine(params, marketID, num.DecimalFromFloat(1))

	// Max in current time
	var maxOI uint64 = 23
	err := engine.RecordTotalStake(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1 := engine.GetTargetStake(now)
	actualTargetStake2 := engine.GetTargetStake(now.Add(time.Minute))

	exp := expectedTargetStake(maxOI)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)
	// Max in past
	now = now.Add(time.Nanosecond)
	err = engine.RecordTotalStake(maxOI-1, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(now)
	actualTargetStake2 = engine.GetTargetStake(now.Add(time.Minute))

	exp = expectedTargetStake(maxOI)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in current time
	now = now.Add(time.Second)
	maxOI = 10 * maxOI
	err = engine.RecordTotalStake(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(now)
	actualTargetStake2 = engine.GetTargetStake(now.Add(time.Minute))

	exp = expectedTargetStake(maxOI)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in past, move time beyond window, don't update OI, max OI should be the last recorded value
	now = now.Add(time.Minute)
	var lastRecordedValue uint64 = 1
	err = engine.RecordTotalStake(lastRecordedValue, now)
	require.NoError(t, err)
	now = now.Add(3 * tWindow)
	actualTargetStake1 = engine.GetTargetStake(now)
	actualTargetStake2 = engine.GetTargetStake(now.Add(time.Minute))

	exp = expectedTargetStake(lastRecordedValue)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in past with smaller value after it, move time beyond window so that the current max gets dropped, now target stake should be based on next value
	now = now.Add(time.Minute)
	var penultimateValue uint64 = 1000
	err = engine.RecordTotalStake(penultimateValue, now)
	require.NoError(t, err)
	// Half a time window
	now = now.Add(30 * time.Minute)
	lastRecordedValue = 5
	err = engine.RecordTotalStake(lastRecordedValue, now)
	require.NoError(t, err)
	// Move entire time window and a bit
	now = now.Add(61 * time.Minute)
	actualTargetStake1 = engine.GetTargetStake(now)
	actualTargetStake2 = engine.GetTargetStake(now.Add(time.Minute))

	exp = expectedTargetStake(lastRecordedValue)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in past with OI of 0 value after it, move time beyond window so that the current max gets dropped, now target stake should be 0
	now = now.Add(time.Minute)
	penultimateValue = 1000
	err = engine.RecordTotalStake(penultimateValue, now)
	require.NoError(t, err)
	// Half a time window
	now = now.Add(30 * time.Minute)
	lastRecordedValueIsZero := uint64(0)
	err = engine.RecordTotalStake(lastRecordedValueIsZero, now)
	require.NoError(t, err)
	// Move entire time window and a bit
	now = now.Add(61 * time.Minute)
	actualTargetStake1 = engine.GetTargetStake(now)
	actualTargetStake2 = engine.GetTargetStake(now.Add(time.Minute))

	exp = expectedTargetStake(lastRecordedValueIsZero)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)
}
