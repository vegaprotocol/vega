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

package supplied

import (
	"testing"

	"code.vegaprotocol.io/vega/core/risk/models"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/require"
)

var (
	min = num.DecimalZero()
	max = num.MaxUint().ToDecimal()
)

func TestBidRange(t *testing.T) {
	prob := calculateBidRange(num.DecimalFromFloat(100), num.DecimalFromFloat(0.01), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 100, len(prob))
	for i := range prob {
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob[i].String())
	}

	prob2 := calculateBidRange(num.DecimalFromFloat(200), num.DecimalFromFloat(0.01), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 100, len(prob2))
	for i := range prob2 {
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob2[i].String())
	}
}

func TestAskRange(t *testing.T) {
	prob := calculateAskRange(num.DecimalFromFloat(100), num.DecimalFromFloat(0.01), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 100, len(prob))
	for i := range prob {
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob[i].String())
	}

	prob2 := calculateAskRange(num.DecimalFromFloat(200), num.DecimalFromFloat(0.01), num.DecimalFromInt64(1), pOfT)
	require.Equal(t, 100, len(prob2))
	for i := range prob2 {
		require.Equal(t, num.DecimalFromFloat(1.0).Sub(num.DecimalFromInt64(int64(i)).Div(num.DecimalFromFloat(100))).String(), prob[i].String())
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

	p := calculateBidRange(num.DecimalFromFloat(900), num.DecimalFromFloat(0.001), num.DecimalFromFloat(0.00012345), risk.ProbabilityOfTrading)

	require.Equal(t, 92, len(p))

	pot := &probabilityOfTrading{
		bidProbability: p,
		bidOffset:      []uint32{0, 40},
	}
	minProb := num.DecimalFromFloat(0.021)

	// order price 899, best bid 900
	// offset is therefore int(1600*(900-899)/900) = 1
	// so we're interpolating between the probability 1 at offset 0 and probability of 0.9524363412970467 at offset 40
	// so expecting scaled probability to be (0.975*1 + 0.025*0.9524363412970467)/2 = 0.4999762182
	require.Equal(t, "0.4994054543", getProbabilityOfTrading(num.DecimalFromFloat(900), num.DecimalFromFloat(900), num.DecimalZero(), num.MaxDecimal(), pot, num.DecimalFromFloat(899), true, minProb, num.DecimalFromInt64(1600)).StringFixed(10))
}

func pOfT(best, p, min, max, tauScaled num.Decimal, isBid bool, applyMinMax bool) num.Decimal {
	if p.LessThanOrEqual(num.DecimalZero()) {
		return defaultMinimumProbabilityOfTrading
	}
	if p.GreaterThanOrEqual(num.DecimalFromInt64(2).Mul(best)) {
		return defaultMinimumProbabilityOfTrading
	}
	return num.DecimalFromFloat(1).Sub(best.Sub(p).Abs().Div(best))
}

func TestGetProbability(t *testing.T) {
	offsetFactor := num.DecimalFromInt64(1000000)
	minProb := num.DecimalFromFloat(0.21)

	// no consensus yet within 20% ticks (bid)
	require.Equal(t, defaultInRangeProbabilityOfTrading.String(), getProbabilityOfTrading(num.DecimalFromFloat(200000), num.DecimalFromFloat(600), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(160000), true, minProb, offsetFactor).String())
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(200000), num.DecimalFromFloat(600), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(159999), true, minProb, offsetFactor).String())

	// no consensus yet within 20% ticks (ask)
	require.Equal(t, defaultInRangeProbabilityOfTrading.String(), getProbabilityOfTrading(num.DecimalFromFloat(100000), num.DecimalFromFloat(200000), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(240000), false, minProb, offsetFactor).String())
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(100000), num.DecimalFromFloat(200000), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(240001), false, minProb, offsetFactor).String())

	// price is GTE best bid and LTE best ask - use defaultInRangeProbabilityOfTrading
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(120), true, minProb, offsetFactor))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(121), true, minProb, offsetFactor))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(220), false, minProb, offsetFactor))
	require.Equal(t, defaultInRangeProbabilityOfTrading, getProbabilityOfTrading(num.DecimalFromFloat(120), num.DecimalFromFloat(220), min, max, &probabilityOfTrading{}, num.DecimalFromInt64(219), false, minProb, offsetFactor))

	bProb := calculateBidRange(num.DecimalFromFloat(400), num.DecimalFromFloat(0.001), num.DecimalFromFloat(1), pOfT)
	aProb := calculateAskRange(num.DecimalFromFloat(600), num.DecimalFromFloat(0.001), num.DecimalFromInt64(1), pOfT)

	bOffsets := make([]uint32, 0, len(bProb))
	aOffsets := make([]uint32, 0, len(aProb))

	for i := range bProb {
		bOffsets = append(bOffsets, uint32(i*1000))
	}
	for i := range aProb {
		aOffsets = append(aOffsets, uint32(i*1000))
	}

	pot := &probabilityOfTrading{
		bidOffset:      bOffsets,
		bidProbability: bProb,
		askOffset:      aOffsets,
		askProbability: aProb,
	}

	// find exact match bid side - offset of 200 is the middle point of the probabilities so should have probability of 0.5 scaled by 0.5
	require.Equal(t, "0.25", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(200), true, minProb, offsetFactor).String())
	// linterp bid => (0.75*0.5+0.25*0.51)/2
	require.Equal(t, "0.25125", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(201), true, minProb, offsetFactor).String())
	// linterp bid => (0.5*0.5+0.5*0.51)/2
	require.Equal(t, "0.2525", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(202), true, minProb, offsetFactor).String())
	// linterp bid => (0.25*0.5+0.75*0.51)/2
	require.Equal(t, "0.25375", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(203), true, minProb, offsetFactor).String())
	// linterp bid => (0*0.5+1*0.51)/2
	require.Equal(t, "0.255", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(204), true, minProb, offsetFactor).String())

	// linterp bid => max(0.2, 0.21) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(100), true, minProb, offsetFactor).String())
	// linterp bid => max(0.75*0 + 0.5*0.2, minProb) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(101), true, minProb, offsetFactor).String())
	// linterp bid => max(0.5*0 + 0.5*0.2, minProb) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(102), true, minProb, offsetFactor).String())
	// linterp bid => max(0.25*0 + 0.5*0.2, minProb) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(103), true, minProb, offsetFactor).String())

	// find exact match ask side - offset of 200 is the middle point of the probabilities so should have probability of 0.5 scaled by 0.5
	require.Equal(t, "0.25", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(900), false, minProb, offsetFactor).String())
	// linterp ask => (0.75*0.5+0.25*0.51)/2
	require.Equal(t, "0.25125", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromFloat(898.5), false, minProb, offsetFactor).String())
	// linterp ask => (0.5*0.5+0.5*0.51)/2
	require.Equal(t, "0.2525", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(897), false, minProb, offsetFactor).String())
	// linterp ask => (0.25*0.5+0.75*0.51)/2
	require.Equal(t, "0.25375", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromFloat(895.5), false, minProb, offsetFactor).String())
	// linterp ask => (0*0.5+1*0.51)/2
	require.Equal(t, "0.255", getProbabilityOfTrading(num.DecimalFromFloat(400), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(894), false, minProb, offsetFactor).String())

	// linterp ask => max(0, minProb) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(1000), false, minProb, offsetFactor).String())
	// linterp ask => max(0.75*0 + 0.5*0.01, minProb) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(999), false, minProb, offsetFactor).String())
	// linterp ask => max(0.5*0 + 0.5*0.01, minProb) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(998), false, minProb, offsetFactor).String())
	// linterp ask => max(0.25*0 + 0.5*0.01, minProb) = 0.21
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, pot, num.DecimalFromInt64(997), false, minProb, offsetFactor).String())

	// // extrapolating
	potForExtrapolation := &probabilityOfTrading{
		bidOffset:      []uint32{0, 1000 * 100},
		askOffset:      []uint32{0, 1000 * 100},
		bidProbability: []num.Decimal{num.DecimalFromFloat(0.9), num.DecimalFromFloat(0.5)},
		askProbability: []num.Decimal{num.DecimalFromFloat(0.9), num.DecimalFromFloat(0.5)},
	}

	// ask
	require.Equal(t, "0.05", getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, potForExtrapolation, num.DecimalFromInt64(720), false, num.DecimalZero(), offsetFactor).String())
	// too far, floored at min prob
	require.Equal(t, minProb.String(), getProbabilityOfTrading(num.DecimalFromFloat(500), num.DecimalFromFloat(600), min, max, potForExtrapolation, num.DecimalFromInt64(10000), false, minProb, offsetFactor).String())
}
