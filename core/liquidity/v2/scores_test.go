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

package liquidity_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestLiquidityScoresMechanics(t *testing.T) {
	var (
		party1     = "party-1"
		party2     = "party-2"
		party3     = "party-3"
		party4     = "party-4"
		ctx        = context.Background()
		now        = time.Now()
		tng        = newTestEngine(t, now)
		bestBid    = num.NewDecimalFromFloat(95)
		bestAsk    = num.NewDecimalFromFloat(105)
		minLpPrice = num.NewUint(90)
		maxLpPrice = num.NewUint(110)
		minPmPrice = num.NewWrappedDecimal(num.NewUint(85), num.DecimalFromFloat(85))
		maxPmPrice = num.NewWrappedDecimal(num.NewUint(115), num.DecimalFromFloat(115))
		commitment = 1000000
		offset     = num.NewUint(2)
	)
	defer tng.ctrl.Finish()
	tng.priceMonitor.EXPECT().GetValidPriceRange().AnyTimes().Return(minPmPrice, maxPmPrice).AnyTimes()

	gomock.InOrder(
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.5)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.4)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.3)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.2)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.1)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestBid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.0)),
	)
	gomock.InOrder(
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.5)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.4)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.3)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.2)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.1)),
		tng.riskModel.EXPECT().ProbabilityOfTrading(bestAsk, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(num.DecimalFromFloat(0.0)),
	)

	// We don't care about the following calls
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	// initialise PoT
	tng.engine.SetGetStaticPricesFunc(func() (num.Decimal, num.Decimal, error) { return bestBid, bestAsk, nil })
	tng.stateVar.OnTick(ctx, now)
	require.True(t, tng.engine.IsPoTInitialised())

	// party1 submission
	tng.sortOutLpSubAndOrders(t, ctx, party1, commitment, offset, minLpPrice, maxLpPrice, bestBid, bestAsk, 9)

	cLiq1, t1 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq1, 1)
	require.True(t, t1.GreaterThan(num.DecimalZero()))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores1 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores1, 1)
	lScoresSumTo1(t, lScores1)

	// party2 submission with 3*commitment
	tng.sortOutLpSubAndOrders(t, ctx, party2, 3*commitment, offset, minLpPrice, maxLpPrice, bestBid, bestAsk, 100)

	cLiq2, t2 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq2, 2)
	require.True(t, t2.GreaterThan(num.DecimalZero()))

	p1cLiq := cLiq2[party1].Copy()
	p2cLiqExp := p1cLiq.Mul(num.DecimalFromFloat(3))
	// there's some ceiling going on when creating order volumes from commitment so check results within delta
	expFP, _ := p2cLiqExp.Float64()
	actFP, _ := cLiq2[party2].Float64()
	require.InDelta(t, expFP, actFP, 1e-4*float64(commitment))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores2 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores2, 2)
	lScoresSumTo1(t, lScores2)

	// party3 submission with 3*offset
	offsetTimes3 := num.UintZero().Mul(offset, num.NewUint(3))
	tng.sortOutLpSubAndOrders(t, ctx, party3, commitment, offsetTimes3, minLpPrice, maxLpPrice, bestBid, bestAsk, 100)

	cLiq3, t3 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq3, 3)
	require.True(t, t3.GreaterThan(num.DecimalZero()))
	require.True(t, cLiq3[party1].GreaterThan(cLiq3[party3]))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores3 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores3, 3)
	lScoresSumTo1(t, lScores3)

	// now add 1 LP, remove 1 LP and change
	//    remove party3
	require.NoError(t, tng.engine.CancelLiquidityProvision(ctx, party3))
	//    add same submission as party3, but by party4
	tng.sortOutLpSubAndOrders(t, ctx, party4, commitment, offsetTimes3, minLpPrice, maxLpPrice, bestBid, bestAsk, 100)

	cLiq4, t4 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq4, 3)
	require.True(t, t4.GreaterThan(num.DecimalZero()))
	// should get same value for party4 as for party3 in previous round
	require.True(t, cLiq4[party4].Equal(cLiq3[party3]))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores4 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores4, 3)
	lScoresSumTo1(t, lScores4)

	keys := make([]string, 0, len(lScores4))
	for k := range lScores4 {
		keys = append(keys, k)
	}
	activeParties := []string{party1, party2, party4}
	require.ElementsMatch(t, activeParties, keys)

	tng.sortOutLpAmendmentAndOrders(t, ctx, party1, 3*commitment, offset, minLpPrice, maxLpPrice, bestBid, bestAsk)

	cLiq5, t5 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq5, 3)
	require.True(t, t5.GreaterThan(num.DecimalZero()))
	// commitment size should have almost no impact on score (only via relative order size differences due to ceiling)
	expFP, _ = cLiq4[party1].Float64()
	actFP, _ = cLiq5[party1].Float64()
	require.InDelta(t, expFP, actFP, 1e-4)

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores5 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores5, 3)
	lScoresSumTo1(t, lScores5)

	// check running average
	n := num.DecimalFromInt64(5)
	nMinus1 := n.Sub(num.DecimalOne())
	nMinus1overN := nMinus1.Div(n)
	expectedScore := (lScores4[party1].Mul(nMinus1overN).Add(cLiq5[party1].Div(t5).Div(n))).Round(10)
	require.True(t, expectedScore.Equal(lScores5[party1]))

	// now reset scores and do another round
	tng.engine.ResetAverageLiquidityScores()
	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)

	lScores6 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores6, 3)
	lScoresSumTo1(t, lScores6)
	for _, p := range activeParties {
		// we've just reset so running average should be same as previous observation normalised
		require.True(t, lScores6[p].Equal((cLiq5[p].Div(t5)).Round(10)))
	}
}

func (tng *testEngine) sortOutLpSubAndOrders(
	t *testing.T,
	ctx context.Context,
	party string,
	commitment int,
	offset *num.Uint,
	minLpPrice, maxLpPrice *num.Uint,
	bestBid, bestAsk num.Decimal,
	maxTimes int,
) {
	t.Helper()

	lps := &types.LiquidityProvisionSubmission{
		MarketID:         tng.marketID,
		CommitmentAmount: num.NewUint(uint64(commitment)),
		Fee:              num.DecimalFromFloat(0.5),
	}

	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgeneration.New(crypto.RandomHash())),
	)
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Return([]*types.Order{}).Times(1)

	// TODO karel - generate orders
	partyOrders := []*types.Order{}

	require.Len(t, partyOrders, len(lps.Buys)+len(lps.Sells))
	require.Equal(t, types.LiquidityProvisionStatusActive, tng.engine.LiquidityProvisionByPartyID(party).Status)
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Return(partyOrders).MaxTimes(maxTimes)
}

func (tng *testEngine) sortOutLpAmendmentAndOrders(
	t *testing.T,
	ctx context.Context,
	party string,
	commitment int,
	offset *num.Uint,
	minLpPrice, maxLpPrice *num.Uint,
	bestBid, bestAsk num.Decimal,
) {
	t.Helper()

	lpa := &types.LiquidityProvisionAmendment{
		MarketID:         tng.marketID,
		CommitmentAmount: num.NewUint(uint64(commitment)),
		Fee:              num.DecimalFromFloat(0.5),
	}

	err := tng.engine.AmendLiquidityProvision(ctx, lpa, party)
	require.NoError(t, err)

	// TODO karel - generate orders
	partyOrders := []*types.Order{}

	require.Len(t, partyOrders, len(lpa.Buys)+len(lpa.Sells))
	require.Equal(t, types.LiquidityProvisionStatusActive, tng.engine.LiquidityProvisionByPartyID(party).Status)
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Return(partyOrders).AnyTimes()
}

func lScoresSumTo1(t *testing.T, lScores map[string]num.Decimal) {
	t.Helper()

	goTo0 := num.DecimalOne()
	for _, v := range lScores {
		goTo0 = goTo0.Sub(v)
	}

	zeroFp, _ := goTo0.Float64()

	require.InDelta(t, 0, zeroFp, 1e-8)
}
