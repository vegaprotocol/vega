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

package target_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/liquidity/target"
	"code.vegaprotocol.io/vega/core/liquidity/target/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	now      = time.Date(2020, 10, 30, 9, 0, 0, 0, time.UTC)
	marketID = "market-1"
)

func TestConstructor(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := target.NewEngine(params, nil, marketID, num.DecimalFromFloat(1))

	require.NotNil(t, engine)
}

func TestRecordOpenInterest(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := target.NewEngine(params, nil, marketID, num.DecimalFromFloat(1))
	err := engine.RecordOpenInterest(9, now)
	require.NoError(t, err)
	err = engine.RecordOpenInterest(0, now)
	require.NoError(t, err)
	err = engine.RecordOpenInterest(11, now.Add(time.Nanosecond))
	require.NoError(t, err)
	err = engine.RecordOpenInterest(12, now.Add(time.Nanosecond))
	require.NoError(t, err)
	err = engine.RecordOpenInterest(13, now.Add(-2*time.Nanosecond))
	require.Error(t, err)
}

func TestGetTargetStake_NoRecordedOpenInterest(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := target.NewEngine(params, nil, marketID, num.DecimalFromFloat(1))
	rf := types.RiskFactor{
		Long:  num.DecimalFromFloat(0.3),
		Short: num.DecimalFromFloat(0.1),
	}

	targetStake, _ := engine.GetTargetStake(rf, now, num.NewUint(123))

	require.Equal(t, num.UintZero(), targetStake)
}

func TestGetTargetStake_VerifyFormula(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := num.DecimalFromFloat(11.3)
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rf := types.RiskFactor{
		Long:  num.DecimalFromFloat(0.3),
		Short: num.DecimalFromFloat(0.1),
	}
	oi := uint64(23)
	markPrice := num.NewUint(123)

	// float64(markPrice.Uint64()*oi) * math.Max(rfLong, rfShort) * scalingFactor
	expectedTargetStake := num.DecimalFromUint(markPrice)
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromUint(num.NewUint(oi)))
	expectedTargetStake = expectedTargetStake.Mul(rf.Long.Mul(scalingFactor))

	engine := target.NewEngine(params, nil, marketID, num.DecimalFromFloat(1))

	err := engine.RecordOpenInterest(oi, now)
	require.NoError(t, err)

	targetStakeNow, _ := engine.GetTargetStake(rf, now, markPrice.Clone())
	targetStakeLaterInWindow, _ := engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())
	targetStakeAtEndOfWindow, _ := engine.GetTargetStake(rf, now.Add(tWindow), markPrice.Clone())
	targetStakeAfterWindow, _ := engine.GetTargetStake(rf, now.Add(tWindow).Add(time.Nanosecond), markPrice.Clone())

	expectedUint, _ := num.UintFromDecimal(expectedTargetStake)
	require.Equal(t, expectedUint, targetStakeNow)
	require.Equal(t, expectedUint, targetStakeLaterInWindow)
	require.Equal(t, expectedUint, targetStakeAtEndOfWindow)
	require.Equal(t, expectedUint, targetStakeAfterWindow)
}

func TestGetTargetStake_VerifyFormulaAfterParametersUpdate(t *testing.T) {
	// given
	tWindow := time.Hour
	scalingFactor := num.DecimalFromFloat(11.3)
	params := types.TargetStakeParameters{
		TimeWindow:    int64(tWindow.Seconds()),
		ScalingFactor: scalingFactor,
	}
	openInterest := uint64(23)

	// setup
	engine := target.NewEngine(params, nil, marketID, num.DecimalFromFloat(1))

	// when
	err := engine.RecordOpenInterest(openInterest, now)

	// then
	require.NoError(t, err)

	// given
	markPrice := num.NewUint(123)
	rf := types.RiskFactor{
		Long:  num.DecimalFromFloat(0.3),
		Short: num.DecimalFromFloat(0.1),
	}

	// when
	targetStakeNow, _ := engine.GetTargetStake(rf, now, markPrice.Clone())
	targetStakeLaterInWindow, _ := engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())
	targetStakeAtEndOfWindow, _ := engine.GetTargetStake(rf, now.Add(tWindow), markPrice.Clone())
	targetStakeAfterWindow, _ := engine.GetTargetStake(rf, now.Add(tWindow).Add(time.Nanosecond), markPrice.Clone())

	// then
	// float64(markPrice.Uint64()*openInterest) * math.Max(rf.Long, rf.Short) * scalingFactor
	expectedTargetStake := num.DecimalFromUint(markPrice)
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromUint(num.NewUint(openInterest)))
	expectedTargetStake = expectedTargetStake.Mul(rf.Long.Mul(scalingFactor))
	expectedTargetStakeUint, _ := num.UintFromDecimal(expectedTargetStake)
	assert.Equal(t, expectedTargetStakeUint, targetStakeNow)
	assert.Equal(t, expectedTargetStakeUint, targetStakeLaterInWindow)
	assert.Equal(t, expectedTargetStakeUint, targetStakeAtEndOfWindow)
	assert.Equal(t, expectedTargetStakeUint, targetStakeAfterWindow)

	// given
	updatedTWindow := tWindow - (10 * time.Minute)
	updatedParams := types.TargetStakeParameters{
		TimeWindow:    int64(updatedTWindow.Seconds()),
		ScalingFactor: num.DecimalFromFloat(10.5),
	}

	// when
	engine.UpdateParameters(updatedParams)

	// given

	newOpenInterest := uint64(14)

	// when
	err = engine.RecordOpenInterest(newOpenInterest, now.Add(time.Second))

	// when
	require.NoError(t, err)

	// The new open interest should be selected as a new max open interest,
	// even though it's smaller than the previously registered open interest,
	// because we are recording the new open interest a second after new
	// maximum time an open interest is kept in memory.
	later := now.Add(updatedTWindow).Add(2 * time.Second)

	// when
	updatedTargetStakeNow, _ := engine.GetTargetStake(rf, later, markPrice.Clone())
	updatedTargetStakeLaterInWindow, _ := engine.GetTargetStake(rf, later.Add(time.Minute), markPrice.Clone())
	updatedTargetStakeAtEndOfWindow, _ := engine.GetTargetStake(rf, later.Add(updatedTWindow), markPrice.Clone())
	updatedTargetStakeAfterWindow, _ := engine.GetTargetStake(rf, later.Add(updatedTWindow).Add(time.Nanosecond), markPrice.Clone())

	// then
	// float64(markPrice.Uint64()*newOpenInterest) * math.Max(rfLong, rfShort) * updatedScalingFactor
	expectedUpdatedTargetStake := num.DecimalFromUint(markPrice)
	expectedUpdatedTargetStake = expectedUpdatedTargetStake.Mul(num.DecimalFromUint(num.NewUint(newOpenInterest)))
	expectedUpdatedTargetStake = expectedUpdatedTargetStake.Mul(rf.Long.Mul(updatedParams.ScalingFactor))
	expectedUpdatedTargetStakeUint, _ := num.UintFromDecimal(expectedUpdatedTargetStake)
	assert.Equal(t, expectedUpdatedTargetStakeUint, updatedTargetStakeNow)
	assert.Equal(t, expectedUpdatedTargetStakeUint, updatedTargetStakeLaterInWindow)
	assert.Equal(t, expectedUpdatedTargetStakeUint, updatedTargetStakeAtEndOfWindow)
	assert.Equal(t, expectedUpdatedTargetStakeUint, updatedTargetStakeAfterWindow)
}

func TestGetTargetStake_VerifyMaxOI(t *testing.T) {
	tWindow := 60 * time.Minute
	scalingFactor := num.DecimalFromFloat(11.3)
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rfLong := num.DecimalFromFloat(0.3)
	rfShort := num.DecimalFromFloat(0.1)
	markPrice := num.NewUint(123)
	expectedTargetStake := func(oi uint64) *num.Uint {
		// float64(markPrice.Uint64()*oi) * math.Max(rfLong, rfShort) * scalingFactor
		mp := num.DecimalFromUint(markPrice)
		mp = mp.Mul(num.DecimalFromUint(num.NewUint(oi)))
		factor := rfLong
		if factor.LessThan(rfShort) {
			factor = rfShort
		}
		mp = mp.Mul(factor.Mul(scalingFactor))
		ump, _ := num.UintFromDecimal(mp)
		return ump
	}

	engine := target.NewEngine(params, nil, marketID, num.DecimalFromFloat(1))
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}

	// Max in current time
	var maxOI uint64 = 23
	err := engine.RecordOpenInterest(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1, _ := engine.GetTargetStake(rf, now, markPrice.Clone())
	actualTargetStake2, _ := engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())

	exp := expectedTargetStake(maxOI)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)
	// Max in past
	now = now.Add(time.Nanosecond)
	markPrice = num.NewUint(456)
	err = engine.RecordOpenInterest(maxOI-1, now)
	require.NoError(t, err)
	actualTargetStake1, _ = engine.GetTargetStake(rf, now, markPrice.Clone())
	actualTargetStake2, _ = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())

	exp = expectedTargetStake(maxOI)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in current time
	now = now.Add(time.Second)
	maxOI = 10 * maxOI
	markPrice = num.NewUint(23)
	err = engine.RecordOpenInterest(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1, _ = engine.GetTargetStake(rf, now, markPrice.Clone())
	actualTargetStake2, _ = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())

	exp = expectedTargetStake(maxOI)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in past, move time beyond window, don't update OI, max OI should be the last recorded value
	now = now.Add(time.Minute)
	var lastRecordedValue uint64 = 1
	err = engine.RecordOpenInterest(lastRecordedValue, now)
	require.NoError(t, err)
	now = now.Add(3 * tWindow)
	markPrice = num.NewUint(7777777)
	actualTargetStake1, _ = engine.GetTargetStake(rf, now, markPrice)
	actualTargetStake2, _ = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)

	exp = expectedTargetStake(lastRecordedValue)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in past with smaller value after it, move time beyond window so that the current max gets dropped, now target stake should be based on next value
	now = now.Add(time.Minute)
	var penultimateValue uint64 = 1000
	err = engine.RecordOpenInterest(penultimateValue, now)
	require.NoError(t, err)
	// Half a time window
	now = now.Add(30 * time.Minute)
	lastRecordedValue = 5
	err = engine.RecordOpenInterest(lastRecordedValue, now)
	require.NoError(t, err)
	// Move entire time window and a bit
	now = now.Add(61 * time.Minute)
	markPrice = num.NewUint(7777777)
	actualTargetStake1, _ = engine.GetTargetStake(rf, now, markPrice)
	actualTargetStake2, _ = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)

	exp = expectedTargetStake(lastRecordedValue)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)

	// Max in past with OI of 0 value after it, move time beyond window so that the current max gets dropped, now target stake should be 0
	now = now.Add(time.Minute)
	penultimateValue = 1000
	err = engine.RecordOpenInterest(penultimateValue, now)
	require.NoError(t, err)
	// Half a time window
	now = now.Add(30 * time.Minute)
	lastRecordedValueIsZero := uint64(0)
	err = engine.RecordOpenInterest(lastRecordedValueIsZero, now)
	require.NoError(t, err)
	// Move entire time window and a bit
	now = now.Add(61 * time.Minute)
	markPrice = num.NewUint(7777777)
	actualTargetStake1, _ = engine.GetTargetStake(rf, now, markPrice)
	actualTargetStake2, _ = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)

	exp = expectedTargetStake(lastRecordedValueIsZero)
	require.Equal(t, exp, actualTargetStake1)
	require.Equal(t, exp, actualTargetStake2)
}

func TestGetTheoreticalTargetStake(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := num.DecimalFromFloat(11.3)
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rfLong := num.DecimalFromFloat(0.3)
	rfShort := num.DecimalFromFloat(0.1)
	var oi uint64 = 23
	markPrice := num.NewUint(123)

	factor := rfLong
	if factor.LessThan(rfShort) {
		factor = rfShort
	}
	expectedTargetStake, _ := num.UintFromDecimal(num.DecimalFromUint(markPrice).Mul(num.DecimalFromUint(num.NewUint(oi))).Mul(factor.Mul(scalingFactor)))

	ctrl := gomock.NewController(t)
	oiCalc := mocks.NewMockOpenInterestCalculator(ctrl)
	engine := target.NewEngine(params, oiCalc, marketID, num.DecimalFromFloat(1))
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}
	err := engine.RecordOpenInterest(oi, now)
	require.NoError(t, err)

	targetStakeNow, _ := engine.GetTargetStake(rf, now, markPrice.Clone())
	require.Equal(t, expectedTargetStake, targetStakeNow)

	var trades []*types.Trade

	// No change in OI
	theoreticalOI := oi
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(1)
	expectedTheoreticalTargetStake := expectedTargetStake.Clone()
	theoreticalTargetStake, _ := engine.GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades)

	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)

	// OI decreases
	theoreticalOI = oi - 2
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(1)
	theoreticalTargetStake, _ = engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)

	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)

	// OI increases
	theoreticalOI = oi + 2
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(1)

	expectedTheoreticalTargetStake, _ = num.UintFromDecimal(num.DecimalFromUint(markPrice).Mul(num.DecimalFromUint(num.NewUint(theoreticalOI))).Mul(factor.Mul(scalingFactor)))

	theoreticalTargetStake, _ = engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)

	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)

	// OI decreases
	theoreticalOI = oi - 5
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(2)

	now = now.Add(30 * time.Minute)
	// last observation still within the time window so expecting theoretical target stake stay unchanged
	theoreticalTargetStake, _ = engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)
	require.Equal(t, expectedTargetStake, theoreticalTargetStake)

	// last observation out of the time window now so expecting theoretical target stake to drop
	expectedTheoreticalTargetStake, _ = num.UintFromDecimal(num.DecimalFromUint(markPrice).Mul(num.DecimalFromUint(num.NewUint(theoreticalOI))).Mul(factor.Mul(scalingFactor)))
	now = now.Add(31 * time.Minute)
	theoreticalTargetStake, _ = engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)
	require.NotEqual(t, expectedTargetStake, theoreticalTargetStake)
	require.True(t, expectedTheoreticalTargetStake.LT(expectedTargetStake))
	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)
}
