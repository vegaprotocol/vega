package target_test

import (
	"math"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/liquidity/target"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/require"
)

var now time.Time = time.Date(2020, 10, 30, 9, 0, 0, 0, time.UTC)

func TestConstructor(t *testing.T) {
	engine := target.NewEngine(time.Hour, 10)

	require.NotNil(t, engine)
}

func TestRecordOpenInterest(t *testing.T) {
	engine := target.NewEngine(time.Hour, 10)
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
	engine := target.NewEngine(time.Hour, 10)
	rf := types.RiskFactor{
		Long:  0.3,
		Short: 0.1,
	}

	targetStake := engine.GetTargetStake(rf, now)

	require.Equal(t, 0.0, targetStake)
}

func TestGetTargetStake_VerifyFormula(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := 11.3
	rfLong := 0.3
	rfShort := 0.1
	var oi uint64 = 23
	expectedTargetStake := float64(oi) * math.Max(rfLong, rfShort) * scalingFactor

	engine := target.NewEngine(tWindow, scalingFactor)
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}
	err := engine.RecordOpenInterest(oi, now)
	require.NoError(t, err)

	targetStakeNow := engine.GetTargetStake(rf, now)
	targetStakeLaterInWindow := engine.GetTargetStake(rf, now.Add(time.Minute))
	targetStakeAtEndOfWindow := engine.GetTargetStake(rf, now.Add(tWindow))
	targetStakeAfterWindow := engine.GetTargetStake(rf, now.Add(tWindow).Add(time.Nanosecond))

	require.Equal(t, expectedTargetStake, targetStakeNow)
	require.Equal(t, expectedTargetStake, targetStakeLaterInWindow)
	require.Equal(t, expectedTargetStake, targetStakeAtEndOfWindow)
	require.Equal(t, expectedTargetStake, targetStakeAfterWindow)
}

func TestGetTargetStake_VerifyMaxOI(t *testing.T) {
	tWindow := time.Hour
	scalingFactor := 11.3
	rfLong := 0.3
	rfShort := 0.1
	expectedTargetStake := func(oi uint64) float64 {
		return float64(oi) * math.Max(rfLong, rfShort) * scalingFactor
	}

	engine := target.NewEngine(tWindow, scalingFactor)
	rf := types.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}

	// Max in current time
	var maxOI uint64 = 23
	err := engine.RecordOpenInterest(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1 := engine.GetTargetStake(rf, now)
	actualTargetStake2 := engine.GetTargetStake(rf, now.Add(time.Minute))
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)
	// Max in past
	now = now.Add(time.Nanosecond)
	err = engine.RecordOpenInterest(maxOI-1, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(rf, now)
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute))
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)

	// Max in current time
	now = now.Add(time.Second)
	maxOI = 10 * maxOI
	err = engine.RecordOpenInterest(maxOI, now)
	require.NoError(t, err)
	actualTargetStake1 = engine.GetTargetStake(rf, now)
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute))
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake1)
	require.Equal(t, expectedTargetStake(maxOI), actualTargetStake2)

	// Max in past, move time beyond window, don't update OI, max OI should be the last recorded value
	now = now.Add(time.Minute)
	var lastRecordedValue uint64 = 1
	err = engine.RecordOpenInterest(lastRecordedValue, now)
	require.NoError(t, err)
	now = now.Add(3 * tWindow)
	actualTargetStake1 = engine.GetTargetStake(rf, now)
	actualTargetStake2 = engine.GetTargetStake(rf, now.Add(time.Minute))
	require.Equal(t, expectedTargetStake(lastRecordedValue), actualTargetStake1)
	require.Equal(t, expectedTargetStake(lastRecordedValue), actualTargetStake2)

}
