package supplied

import (
	"strconv"
	"testing"

	"code.vegaprotocol.io/vega/risk/models"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

var (
	min = num.DecimalZero()
	max = num.MaxUint().ToDecimal()
)

func TestBidRange(t *testing.T) {
	offsets, prob := calculateBidRange(num.NewUint(100), num.NewUint(0), num.NewUint(1), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 101, len(offsets))
	require.Equal(t, 101, len(prob))
	for i, o := range offsets {
		require.Equal(t, strconv.Itoa(i), o.String())
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob[i].String())
	}

	offsets2, prob2 := calculateBidRange(num.NewUint(200), num.NewUint(0), num.NewUint(2), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 101, len(offsets2))
	require.Equal(t, 101, len(prob2))
	for i, o := range offsets2 {
		require.Equal(t, strconv.Itoa(2*i), o.String())
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob2[i].String())
	}
}

func TestBidRangeAtTheEdge(t *testing.T) {
	params := &types.LogNormalRiskModel{
		RiskAversionParameter: num.DecimalFromFloat(0.001),
		Tau:                   num.DecimalFromFloat(0o0011407711613050422),
		Params: &types.LogNormalModelParams{
			R:     num.DecimalFromFloat(0.016),
			Sigma: num.DecimalFromFloat(1.5),
		},
	}
	risk, _ := models.NewBuiltinFutures(params, "asset")

	o, p := calculateBidRange(num.NewUint(900), num.Zero(), num.NewUint(10000), num.DecimalFromFloat(0.00012345), risk.ProbabilityOfTrading)

	require.Equal(t, 2, len(o))
	require.Equal(t, 2, len(p))

	pot := &probabilityOfTrading{
		bidOffset:      o,
		bidProbability: p,
	}
	minProb := num.DecimalFromFloat(0.021)

	// order price 899, best bid 900
	// we have offset 0 and offset 10000
	// so we're interpolating between the probability 1 at offset 0 (with weight 1-1e-4)
	// and probability of 0 at offset 10000 with weight 1e-4
	// so expecting scaled probability to be
	require.Equal(t, "0.49995", getProbabilityOfTrading(num.DecimalFromFloat(900), num.DecimalFromFloat(900), num.DecimalZero(), num.MaxDecimal(), pot, num.DecimalFromFloat(899), true, minProb, num.DecimalZero()).String())
}

func TestAskRange(t *testing.T) {
	offsets, prob := calculateAskRange(num.NewUint(100), num.NewUint(200), num.NewUint(1), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 101, len(offsets))
	require.Equal(t, 101, len(prob))
	for i, o := range offsets {
		require.Equal(t, strconv.Itoa(i), o.String())
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob[i].String())
	}

	offsets2, prob2 := calculateAskRange(num.NewUint(100), num.NewUint(300), num.NewUint(2), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 101, len(offsets2))
	require.Equal(t, 101, len(prob2))
	for i, o := range offsets2 {
		require.Equal(t, strconv.Itoa(2*i), o.String())
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob2[i].String())
	}
}

func pOfT(best, p *num.Uint, min, max, tauScaled num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
	return num.DecimalFromFloat(1).Sub(best.ToDecimal().Sub(p.ToDecimal()).Abs().Div(max.Sub(min)))
}

func TestGetProbability(t *testing.T) {
	minProb := num.DecimalFromFloat(0.021)

	// no consensus yet within 100 ticks (bid)
	require.Equal(t, defaultProbability.String(), getProbabilityOfTrading(num.DecimalFromFloat(200000), num.DecimalFromFloat(600), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(150000), true, minProb, num.DecimalFromFloat(1000)).String())
	require.Equal(t, defaultProbability.String(), getProbabilityOfTrading(num.DecimalFromFloat(200000), num.DecimalFromFloat(600), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(100000), true, minProb, num.DecimalFromFloat(1000)).String())
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(200000), num.DecimalFromFloat(600), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(99999), true, minProb, num.DecimalFromFloat(1000)).String())

	// no consensus yet within 100 ticks (ask)
	require.Equal(t, defaultProbability.String(), getProbabilityOfTrading(num.DecimalFromFloat(100000), num.DecimalFromFloat(200000), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(250000), false, minProb, num.DecimalFromFloat(1000)).String())
	require.Equal(t, defaultProbability.String(), getProbabilityOfTrading(num.DecimalFromFloat(100000), num.DecimalFromFloat(200000), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(300000), false, minProb, num.DecimalFromFloat(1000)).String())
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(100000), num.DecimalFromFloat(200000), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(300001), false, minProb, num.DecimalFromFloat(1000)).String())

	// price is GTE best bid and LTE best ask - use defaultInRangeProbabilityOfTrading
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(120), true, minProb, num.DecimalZero()))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(121), true, minProb, num.DecimalZero()))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(220), false, minProb, num.DecimalZero()))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(219), false, minProb, num.DecimalZero()))

	bOffsets, bProb := calculateBidRange(num.NewUint(400), num.NewUint(0), num.NewUint(1), num.DecimalFromInt64(1), pOfT)
	aOffsets, aProb := calculateAskRange(num.NewUint(600), num.NewUint(1000), num.NewUint(1), num.DecimalFromInt64(1), pOfT)

	pot := &probabilityOfTrading{
		bidOffset:      bOffsets,
		bidProbability: bProb,
		askOffset:      aOffsets,
		askProbability: aProb,
	}

	// find exact match bid side - offset of 200 is the middle point of the probabilities so should have probability of 0.5 scaled by 0.5
	require.Equal(t, "0.25", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(200), true, minProb, num.DecimalZero()).String())
	// linterp bid => (0.75*0.5+0.25*0.51)/2
	require.Equal(t, "0.25125", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(201), true, minProb, num.DecimalZero()).String())
	// linterp bid => (0.5*0.5+0.5*0.51)/2
	require.Equal(t, "0.2525", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(202), true, minProb, num.DecimalZero()).String())
	// linterp bid => (0.25*0.5+0.75*0.51)/2
	require.Equal(t, "0.25375", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(203), true, minProb, num.DecimalZero()).String())
	// linterp bid => (0*0.5+1*0.51)/2
	require.Equal(t, "0.255", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(204), true, minProb, num.DecimalZero()).String())

	// linterp bid => max(0, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(100), true, minProb, num.DecimalZero()).String())
	// linterp bid => max(0.75*0 + 0.5*0.01, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(101), true, minProb, num.DecimalZero()).String())
	// linterp bid => max(0.5*0 + 0.5*0.01, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(102), true, minProb, num.DecimalZero()).String())
	// linterp bid => max(0.25*0 + 0.5*0.01, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(103), true, minProb, num.DecimalZero()).String())

	// find exact match bid side - offset of 200 is the middle point of the probabilities so should have probability of 0.5 scaled by 0.5
	require.Equal(t, "0.25", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(800), false, minProb, num.DecimalZero()).String())
	// linterp ask => (0.75*0.5+0.25*0.51)/2
	require.Equal(t, "0.25125", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(799), false, minProb, num.DecimalZero()).String())
	// linterp ask => (0.5*0.5+0.5*0.51)/2
	require.Equal(t, "0.2525", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(798), false, minProb, num.DecimalZero()).String())
	// linterp ask => (0.25*0.5+0.75*0.51)/2
	require.Equal(t, "0.25375", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(797), false, minProb, num.DecimalZero()).String())
	// linterp ask => (0*0.5+1*0.51)/2
	require.Equal(t, "0.255", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(796), false, minProb, num.DecimalZero()).String())

	// linterp ask => max(0, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(1000), false, minProb, num.DecimalZero()).String())
	// linterp ask => max(0.75*0 + 0.5*0.01, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(999), false, minProb, num.DecimalZero()).String())
	// linterp ask => max(0.5*0 + 0.5*0.01, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(998), false, minProb, num.DecimalZero()).String())
	// linterp ask => max(0.25*0 + 0.5*0.01, minProb) = 0.021
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(997), false, minProb, num.DecimalZero()).String())

	// not within the min,max
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), num.DecimalFromFloat(200), max, pot, num.DecimalFromInt64(150), true, minProb, num.DecimalZero()).String())
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, num.DecimalFromFloat(800), pot, num.DecimalFromInt64(1000), false, minProb, num.DecimalZero()).String())

	// extrapolating
	potForExtrapolation := &probabilityOfTrading{
		bidOffset:      []num.Decimal{num.DecimalFromFloat(0), num.DecimalFromFloat(10)},
		askOffset:      []num.Decimal{num.DecimalFromFloat(0), num.DecimalFromFloat(10)},
		bidProbability: []num.Decimal{num.DecimalFromFloat(0.9), num.DecimalFromFloat(0.5)},
		askProbability: []num.Decimal{num.DecimalFromFloat(0.9), num.DecimalFromFloat(0.5)},
	}
	// bid
	require.Equal(t, "0.05", getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, potForExtrapolation, num.DecimalFromInt64(480), true, minProb, num.DecimalZero()).String())
	// too far, floored at min prob
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, potForExtrapolation, num.DecimalFromInt64(100), true, minProb, num.DecimalZero()).String())

	// ask
	require.Equal(t, "0.05", getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, potForExtrapolation, num.DecimalFromInt64(620), false, minProb, num.DecimalZero()).String())
	// too far, floored at min prob
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, potForExtrapolation, num.DecimalFromInt64(1000), false, minProb, num.DecimalZero()).String())
}
