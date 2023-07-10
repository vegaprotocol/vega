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
		tng        = newTestEngine(t)
		bestBid    = num.NewDecimalFromFloat(95)
		bestAsk    = num.NewDecimalFromFloat(105)
		minLpPrice = num.NewUint(90)
		maxLpPrice = num.NewUint(110)
		minPmPrice = num.NewWrappedDecimal(num.NewUint(85), num.DecimalFromFloat(85))
		maxPmPrice = num.NewWrappedDecimal(num.NewUint(115), num.DecimalFromFloat(115))
		commitment = 1000000
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

	tng.auctionState.EXPECT().InAuction().Return(false).AnyTimes()

	zero := num.UintZero()

	tng.orderbook.EXPECT().GetBestStaticBidPrice().Return(zero, nil).AnyTimes()
	tng.orderbook.EXPECT().GetBestStaticAskPrice().Return(zero, nil).AnyTimes()

	// initialise PoT
	tng.engine.SetGetStaticPricesFunc(func() (num.Decimal, num.Decimal, error) { return bestBid, bestAsk, nil })
	tng.stateVar.OnTick(ctx, now)
	require.True(t, tng.engine.IsPoTInitialised())

	idgen := idgeneration.New(crypto.RandomHash())

	partyOneOrders := []*types.Order{
		{Side: types.SideBuy, Price: num.NewUint(98), Size: 5103},
		{Side: types.SideBuy, Price: num.NewUint(93), Size: 5377},
		{Side: types.SideSell, Price: num.NewUint(102), Size: 4902},
		{Side: types.SideSell, Price: num.NewUint(107), Size: 4673},
	}

	// party1 submission
	tng.submitLiquidityProvisionAndCreateOrders(t, ctx, party1, commitment, idgen, partyOneOrders)

	cLiq1, t1 := tng.engine.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	require.Len(t, cLiq1, 1)
	require.True(t, t1.GreaterThan(num.DecimalZero()))

	tng.engine.UpdateAverageLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	lScores1 := tng.engine.GetAverageLiquidityScores()
	require.Len(t, lScores1, 1)
	lScoresSumTo1(t, lScores1)

	// party2 submission with 3*commitment
	partyTwoOrders := []*types.Order{
		{Side: types.SideBuy, Price: num.NewUint(98), Size: 15307},
		{Side: types.SideBuy, Price: num.NewUint(93), Size: 16130},
		{Side: types.SideSell, Price: num.NewUint(102), Size: 14706},
		{Side: types.SideSell, Price: num.NewUint(107), Size: 14019},
	}

	tng.submitLiquidityProvisionAndCreateOrders(t, ctx, party2, 3*commitment, idgen, partyTwoOrders)

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
	partyThreeOrders := []*types.Order{
		{Side: types.SideBuy, Price: num.NewUint(94), Size: 5320},
		{Side: types.SideBuy, Price: num.NewUint(89), Size: 5618},
		{Side: types.SideSell, Price: num.NewUint(106), Size: 4717},
		{Side: types.SideSell, Price: num.NewUint(111), Size: 4505},
	}

	tng.submitLiquidityProvisionAndCreateOrders(t, ctx, party3, commitment, idgen, partyThreeOrders)

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
	tng.submitLiquidityProvisionAndCreateOrders(t, ctx, party4, commitment, idgen, partyThreeOrders)

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
}

func (tng *testEngine) submitLiquidityProvisionAndCreateOrders(
	t *testing.T,
	ctx context.Context,
	party string,
	commitment int,
	idgen *idgeneration.IDGenerator,
	orders []*types.Order,
) {
	t.Helper()

	lps := &types.LiquidityProvisionSubmission{
		MarketID:         tng.marketID,
		CommitmentAmount: num.NewUint(uint64(commitment)),
		Fee:              num.DecimalFromFloat(0.5),
	}

	_, err := tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgeneration.New(crypto.RandomHash()))
	require.NoError(t, err)

	zero := num.UintOne()
	now := tng.tsvc.GetTimeNow()
	tng.engine.ResetSLAEpoch(now, zero, zero, num.DecimalZero())
	tng.engine.ApplyPendingProvisions(ctx, now)

	for _, o := range orders {
		o.ID = idgen.NextID()
		o.MarketID = tng.marketID
		o.TimeInForce = types.OrderTimeInForceGTC
		o.Type = types.OrderTypeLimit
		o.Status = types.OrderStatusActive
		o.Remaining = o.Size
	}

	require.Equal(t, types.LiquidityProvisionStatusActive, tng.engine.LiquidityProvisionByPartyID(party).Status)
	tng.orderbook.EXPECT().GetOrdersPerParty(party).Return(orders).AnyTimes()
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
