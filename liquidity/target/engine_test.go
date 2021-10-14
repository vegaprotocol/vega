package target_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/liquidity/target"
	"code.vegaprotocol.io/vega/liquidity/target/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var now = time.Date(2020, 10, 30, 9, 0, 0, 0, time.UTC)

func TestConstructor(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := target.NewEngine(params, nil)

	require.NotNil(t, engine)
}

func TestRecordOpenInterest(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: num.DecimalFromFloat(10)}
	engine := target.NewEngine(params, nil)
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
	engine := target.NewEngine(params, nil)
	rf := types.RiskFactor{
		Long:  num.DecimalFromFloat(0.3),
		Short: num.DecimalFromFloat(0.1),
	}

	targetStake := engine.GetTargetStake(rf, now, num.NewUint(123))

	require.Equal(t, num.Zero(), targetStake)
}

func TestGetTargetStake_VerifyFormula(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := num.DecimalFromFloat(11.3)
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rfLong := num.DecimalFromFloat(0.3)
	rfShort := num.DecimalFromFloat(0.1)
	var oi uint64 = 23
	markPrice := num.NewUint(123)

	// float64(markPrice.Uint64()*oi) * math.Max(rfLong, rfShort) * scalingFactor
	expectedTargetStake := num.DecimalFromUint(markPrice)
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromUint(num.NewUint(oi)))
	factor := rfLong
	if factor.LessThan(rfShort) {
		factor = rfShort
	}
	expectedTargetStake = expectedTargetStake.Mul(factor.Mul(scalingFactor))

	engine := target.NewEngine(params, nil)
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}
	err := engine.RecordOpenInterest(oi, now)
	require.NoError(t, err)

	targetStakeNow := engine.GetTargetStake(rf, now, markPrice.Clone())
	targetStakeLaterInWindow := engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())
	targetStakeAtEndOfWindow := engine.GetTargetStake(rf, now.Add(tWindow), markPrice.Clone())
	targetStakeAfterWindow := engine.GetTargetStake(rf, now.Add(tWindow).Add(time.Nanosecond), markPrice.Clone())

	expectedUint, _ := num.UintFromDecimal(expectedTargetStake)
	require.Equal(t, expectedUint, targetStakeNow)
	require.Equal(t, expectedUint, targetStakeLaterInWindow)
	require.Equal(t, expectedUint, targetStakeAtEndOfWindow)
	require.Equal(t, expectedUint, targetStakeAfterWindow)
}

func TestGetTargetStake_VerifyMaxOI(t *testing.T) {
	tWindow := time.Hour
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

	engine := target.NewEngine(params, nil)
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}

	// Max in current time
	var maxOI uint64 = 23
	err := engine.RecordOpenInterest(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1 := engine.GetTargetStake(rf, now, markPrice.Clone())
	actualTargetStake2 := engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)
	// Max in past
	now = now.Add(time.Nanosecond)
	markPrice = num.NewUint(456)
	err = engine.RecordOpenInterest(maxOI-1, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(rf, now, markPrice.Clone())
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)

	// Max in current time
	now = now.Add(time.Second)
	maxOI = 10 * maxOI
	markPrice = num.NewUint(23)
	err = engine.RecordOpenInterest(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(rf, now, markPrice.Clone())
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice.Clone())
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)

	// Max in past, move time beyond window, don't update OI, max OI should be the last recorded value
	now = now.Add(time.Minute)
	var lastRecordedValue uint64 = 1
	err = engine.RecordOpenInterest(lastRecordedValue, now)
	require.NoError(t, err)
	now = now.Add(3 * tWindow)
	markPrice = num.NewUint(7777777)
	actualTargetStake1 = engine.GetTargetStake(rf, now, markPrice)
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)
	require.Equal(t, expectedTargetStake(lastRecordedValue), actualTargetStake1)
	require.Equal(t, expectedTargetStake(lastRecordedValue), actualTargetStake2)
}

func TestGetTheoreticalTargetStake(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := num.DecimalFromFloat(11.3)
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rfLong := num.DecimalFromFloat(0.3)
	rfShort := num.DecimalFromFloat(0.1)
	var oi uint64 = 23
	markPrice := num.NewUint(123)

	// float64(markPrice.Uint64()*oi) * math.Max(rfLong, rfShort) * scalingFactor
	expectedTargetStake := num.DecimalFromUint(markPrice)
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromUint(num.NewUint(oi)))
	factor := rfLong
	if factor.LessThan(rfShort) {
		factor = rfShort
	}
	expectedTargetStake = expectedTargetStake.Mul(factor.Mul(scalingFactor))

	ctrl := gomock.NewController(t)
	oiCalc := mocks.NewMockOpenInterestCalculator(ctrl)
	engine := target.NewEngine(params, oiCalc)
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}
	err := engine.RecordOpenInterest(oi, now)
	require.NoError(t, err)

	expectedUint, _ := num.UintFromDecimal(expectedTargetStake)
	targetStakeNow := engine.GetTargetStake(rf, now, markPrice.Clone())
	require.Equal(t, expectedUint, targetStakeNow)

	var trades []*types.Trade

	// No change in OI
	theoreticalOI := oi
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(1)
	expectedTheoreticalTargetStake, _ := num.UintFromDecimal(expectedTargetStake)
	theoreticalTargetStake := engine.GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades)

	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)

	// OI decreases
	theoreticalOI = oi - 2
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(1)
	theoreticalTargetStake = engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)

	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)

	// OI increases
	theoreticalOI = oi + 2
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(1)

	// float64(markPrice.Uint64()*theoreticalOI) * math.Max(rfLong, rfShort) * scalingFactor
	expectedTargetStake = num.DecimalFromUint(markPrice)
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromUint(num.NewUint(theoreticalOI)))
	expectedTargetStake = expectedTargetStake.Mul(factor.Mul(scalingFactor))
	expectedTheoreticalTargetStake, _ = num.UintFromDecimal(expectedTargetStake)

	theoreticalTargetStake = engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)

	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)
}
