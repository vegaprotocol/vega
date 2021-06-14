package target_test

import (
	"math"
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
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: 10}
	engine := target.NewEngine(params, nil)

	require.NotNil(t, engine)
}

func TestRecordOpenInterest(t *testing.T) {
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: 10}
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
	params := types.TargetStakeParameters{TimeWindow: 3600, ScalingFactor: 10}
	engine := target.NewEngine(params, nil)
	rf := types.RiskFactor{
		Long:  0.3,
		Short: 0.1,
	}

	targetStake := engine.GetTargetStake(rf, now, num.NewUint(123))

	require.Equal(t, num.DecimalFromFloat(0.0), targetStake)
}

func TestGetTargetStake_VerifyFormula(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := 11.3
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rfLong := 0.3
	rfShort := 0.1
	var oi uint64 = 23
	var markPrice *num.Uint = num.NewUint(123)

	// float64(markPrice.Uint64()*oi) * math.Max(rfLong, rfShort) * scalingFactor
	expectedTargetStake := num.DecimalFromUint(markPrice)
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromUint(num.NewUint(oi)))
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromFloat(math.Max(rfLong, rfShort) * scalingFactor))

	engine := target.NewEngine(params, nil)
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}
	err := engine.RecordOpenInterest(oi, now)
	require.NoError(t, err)

	targetStakeNow := engine.GetTargetStake(rf, now, markPrice)
	targetStakeLaterInWindow := engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)
	targetStakeAtEndOfWindow := engine.GetTargetStake(rf, now.Add(tWindow), markPrice)
	targetStakeAfterWindow := engine.GetTargetStake(rf, now.Add(tWindow).Add(time.Nanosecond), markPrice)

	require.Equal(t, expectedTargetStake, targetStakeNow)
	require.Equal(t, expectedTargetStake, targetStakeLaterInWindow)
	require.Equal(t, expectedTargetStake, targetStakeAtEndOfWindow)
	require.Equal(t, expectedTargetStake, targetStakeAfterWindow)
}

func TestGetTargetStake_VerifyMaxOI(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := 11.3
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rfLong := 0.3
	rfShort := 0.1
	var markPrice *num.Uint = num.NewUint(123)
	expectedTargetStake := func(oi uint64) num.Decimal {
		// float64(markPrice.Uint64()*oi) * math.Max(rfLong, rfShort) * scalingFactor
		mp := num.DecimalFromUint(markPrice)
		mp = mp.Mul(num.DecimalFromUint(num.NewUint(oi)))
		mp = mp.Mul(num.DecimalFromFloat(math.Max(rfLong, rfShort) * scalingFactor))
		return mp
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
	actualTargetStake1 := engine.GetTargetStake(rf, now, markPrice)
	actualTargetStake2 := engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)
	// Max in past
	now = now.Add(time.Nanosecond)
	markPrice = num.NewUint(456)
	err = engine.RecordOpenInterest(maxOI-1, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(rf, now, markPrice)
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)

	// Max in current time
	now = now.Add(time.Second)
	maxOI = 10 * maxOI
	markPrice = num.NewUint(23)
	err = engine.RecordOpenInterest(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(rf, now, markPrice)
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute), markPrice)
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
	scalingFactor := 11.3
	params := types.TargetStakeParameters{TimeWindow: int64(tWindow.Seconds()), ScalingFactor: scalingFactor}
	rfLong := 0.3
	rfShort := 0.1
	var oi uint64 = 23
	var markPrice *num.Uint = num.NewUint(123)

	// float64(markPrice.Uint64()*oi) * math.Max(rfLong, rfShort) * scalingFactor
	expectedTargetStake := num.DecimalFromUint(markPrice)
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromUint(num.NewUint(oi)))
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromFloat(math.Max(rfLong, rfShort) * scalingFactor))

	ctrl := gomock.NewController(t)
	oiCalc := mocks.NewMockOpenInterestCalculator(ctrl)
	engine := target.NewEngine(params, oiCalc)
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}
	err := engine.RecordOpenInterest(oi, now)
	require.NoError(t, err)

	targetStakeNow := engine.GetTargetStake(rf, now, markPrice)
	require.Equal(t, expectedTargetStake, targetStakeNow)

	var trades []*types.Trade

	// No change in OI
	theoreticalOI := oi
	oiCalc.EXPECT().GetOpenInterestGivenTrades(trades).Return(theoreticalOI).MaxTimes(1)
	expectedTheoreticalTargetStake, _ := num.UintFromDecimal(expectedTargetStake)
	theoreticalTargetStake := engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)

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
	expectedTargetStake = expectedTargetStake.Mul(num.DecimalFromFloat(math.Max(rfLong, rfShort) * scalingFactor))
	expectedTheoreticalTargetStake, _ = num.UintFromDecimal(expectedTargetStake)

	theoreticalTargetStake = engine.GetTheoreticalTargetStake(rf, now, markPrice, trades)

	require.Equal(t, expectedTheoreticalTargetStake, theoreticalTargetStake)

}
