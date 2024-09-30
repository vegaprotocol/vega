// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package amm

import (
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/amm/mocks"
	"code.vegaprotocol.io/vega/core/execution/common"
	cmocks "code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	riskFactors    = &types.RiskFactor{Market: "", Short: num.DecimalOne(), Long: num.DecimalOne()}
	scalingFactors = &types.ScalingFactors{InitialMargin: num.DecimalOne()}
	slippage       = num.DecimalOne()
)

func TestSubmitAMM(t *testing.T) {
	t.Run("test one pool per party", testOnePoolPerParty)
	t.Run("test creation of sparse AMM", testSparseAMMEngine)
	t.Run("test AMM snapshot", testAMMSnapshot)
}

func TestAMMTrading(t *testing.T) {
	t.Run("test basic submit order", testBasicSubmitOrder)
	t.Run("test submit order at best price", testSubmitOrderAtBestPrice)
	t.Run("test submit market order", testSubmitMarketOrder)
	t.Run("test submit market order unbounded", testSubmitMarketOrderUnbounded)
	t.Run("test submit order pro rata", testSubmitOrderProRata)
	t.Run("test best prices and volume", testBestPricesAndVolume)

	t.Run("test submit buy order across AMM boundary", testSubmitOrderAcrossAMMBoundary)
	t.Run("test submit sell order across AMM boundary", testSubmitOrderAcrossAMMBoundarySell)
}

func TestAmendAMM(t *testing.T) {
	t.Run("test amend AMM which doesn't exist", testAmendAMMWhichDoesntExist)
	t.Run("test amend AMM with sparse amend", testAmendAMMSparse)
	t.Run("test amend AMM insufficient commitment", testAmendInsufficientCommitment)
	t.Run("test amend AMM when position to large", testAmendWhenPositionLarge)
}

func TestClosingAMM(t *testing.T) {
	t.Run("test closing a pool as reduce only when its position is 0", testClosingReduceOnlyPool)
	t.Run("test amending closing pool makes it actives", testAmendMakesClosingPoolActive)
	t.Run("test closing pool removed when position hits zero", testClosingPoolRemovedWhenPositionZero)
	t.Run("test closing pool immediately", testClosingPoolImmediate)
}

func TestStoppingAMM(t *testing.T) {
	t.Run("test stopping distressed AMM", testStoppingDistressedAMM)
	t.Run("test AMM with no balance is stopped", testAMMWithNoBalanceStopped)
	t.Run("test market closure", testMarketClosure)
}

func testOnePoolPerParty(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// when the party submits another, it is rejected
	_, err := tst.engine.Create(ctx, submit, vgcrypto.RandomHash(), riskFactors, scalingFactors, slippage)
	require.ErrorContains(t, err, "party already own a pool for market")
}

func testAmendAMMWhichDoesntExist(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	// make an amend when the party doesn't have a pool
	party, _ := getParty(t, tst)
	amend := getPoolAmendment(t, party, tst.marketID)

	_, _, err := tst.engine.Amend(ctx, amend, riskFactors, scalingFactors, slippage)
	require.ErrorIs(t, err, ErrNoPoolMatchingParty)
}

func testAmendAMMSparse(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	amend := getPoolAmendment(t, party, tst.marketID)
	// no amend to the commitment amount
	amend.CommitmentAmount = nil
	// no amend to the margin factors either
	amend.Parameters.LeverageAtLowerBound = nil
	amend.Parameters.LeverageAtUpperBound = nil
	// to change something at least, inc the base + bounds by 1
	amend.Parameters.Base.AddSum(num.UintOne())
	amend.Parameters.UpperBound.AddSum(num.UintOne())
	amend.Parameters.LowerBound.AddSum(num.UintOne())

	ensurePosition(t, tst.pos, 0, nil)
	updated, _, err := tst.engine.Amend(ctx, amend, riskFactors, scalingFactors, slippage)
	require.NoError(t, err)

	tst.engine.Confirm(ctx, updated)
}

func testAmendInsufficientCommitment(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	poolID := tst.engine.poolsCpy[0].ID

	amend := getPoolAmendment(t, party, tst.marketID)
	// no amend to the commitment amount
	amend.CommitmentAmount = nil

	// amend to super wide bounds so that the commitment is too thin to support the AMM
	amend.Parameters.Base.AddSum(num.UintOne())
	amend.Parameters.UpperBound.AddSum(num.NewUint(1000000))
	amend.Parameters.LowerBound.AddSum(num.UintOne())

	_, _, err := tst.engine.Amend(ctx, amend, riskFactors, scalingFactors, slippage)
	require.ErrorContains(t, err, "commitment amount too low")

	// check that the original pool still exists
	assert.Equal(t, poolID, tst.engine.poolsCpy[0].ID)
}

func testAmendWhenPositionLarge(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	poolID := tst.engine.poolsCpy[0].ID

	amend := getPoolAmendment(t, party, tst.marketID)

	// lower commitment so that the AMM's position at the same price bounds will be less
	amend.CommitmentAmount = num.NewUint(50000000000)

	expectBalanceChecks(t, tst, party, subAccount, 100000000000)
	ensurePosition(t, tst.pos, 20000000, nil)
	_, _, err := tst.engine.Amend(ctx, amend, riskFactors, scalingFactors, slippage)
	require.ErrorContains(t, err, "current position is outside of amended bounds")

	// check that the original pool still exists
	assert.Equal(t, poolID, tst.engine.poolsCpy[0].ID)

	expectBalanceChecks(t, tst, party, subAccount, 100000000000)
	ensurePosition(t, tst.pos, -20000000, nil)
	_, _, err = tst.engine.Amend(ctx, amend, riskFactors, scalingFactors, slippage)
	require.ErrorContains(t, err, "current position is outside of amended bounds")

	// check that the original pool still exists
	assert.Equal(t, poolID, tst.engine.poolsCpy[0].ID)
}

func testBasicSubmitOrder(t *testing.T) {
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// now submit an order against it
	agg := &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideBuy,
		Price:     num.NewUint(2100),
		Type:      types.OrderTypeLimit,
	}

	ensurePosition(t, tst.pos, 0, num.NewUint(0))
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2000), num.NewUint(2020))
	require.Len(t, orders, 1)
	assert.Equal(t, "2009", orders[0].Price.String())
	assert.Equal(t, 236855, int(orders[0].Size))

	// AMM is now short, but another order comes in that will flip its position to long
	agg = &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideSell,
		Price:     num.NewUint(1900),
	}

	// fair-price is now 2020
	bb, _, ba, _ := tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "2019", bb.String())
	assert.Equal(t, "2021", ba.String())

	orders = tst.engine.SubmitOrder(agg, num.NewUint(2020), num.NewUint(1990))
	require.Len(t, orders, 1)
	assert.Equal(t, "2004", orders[0].Price.String())
	// note that this volume being bigger than 242367 above means we've moved back to position, then flipped
	// sign, and took volume from the other curve.
	assert.Equal(t, 362325, int(orders[0].Size))
}

func testSubmitOrderAtBestPrice(t *testing.T) {
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// AMM has fair-price of 2000 so is willing to sell at 2001, send an incoming buy order at 2001
	agg := &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideBuy,
		Price:     num.NewUint(2001),
		Type:      types.OrderTypeLimit,
	}

	ensurePosition(t, tst.pos, 0, num.NewUint(0))
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2000), num.NewUint(2001))
	require.Len(t, orders, 1)
	assert.Equal(t, "2000", orders[0].Price.String())
	assert.Equal(t, 11927, int(orders[0].Size))

	bb, _, ba, _ := tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "2002", ba.String())
	assert.Equal(t, "2000", bb.String())

	// now trade back with a price of 2000
	agg = &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideSell,
		Price:     num.NewUint(2000),
		Type:      types.OrderTypeLimit,
	}
	orders = tst.engine.SubmitOrder(agg, num.NewUint(2001), num.NewUint(2000))
	require.Len(t, orders, 1)
	assert.Equal(t, "2000", orders[0].Price.String())
	assert.Equal(t, 11927, int(orders[0].Size))
}

func testSubmitMarketOrder(t *testing.T) {
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// now submit an order against it
	agg := &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideSell,
		Price:     num.NewUint(0),
		Type:      types.OrderTypeMarket,
	}

	ensurePosition(t, tst.pos, 0, num.NewUint(0))
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2000), num.NewUint(1980))
	require.Len(t, orders, 1)
	assert.Equal(t, "1989", orders[0].Price.String())
	assert.Equal(t, 251890, int(orders[0].Size))
}

func testSubmitMarketOrderUnbounded(t *testing.T) {
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// now submit an order against it
	agg := &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideSell,
		Price:     num.NewUint(0),
		Type:      types.OrderTypeMarket,
	}

	ensurePosition(t, tst.pos, 0, num.NewUint(0))
	orders := tst.engine.SubmitOrder(agg, num.NewUint(1980), nil)
	require.Len(t, orders, 1)
	assert.Equal(t, "1960", orders[0].Price.String())
	assert.Equal(t, 1000000, int(orders[0].Size))
}

func testSubmitOrderProRata(t *testing.T) {
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		expectSubaccountCreation(t, tst, party, subAccount)
		whenAMMIsSubmitted(t, tst, submit)
	}

	ensurePositionN(t, tst.pos, 0, num.NewUint(0), 3)

	// now submit an order against it
	agg := &types.Order{
		Size:      666,
		Remaining: 666,
		Side:      types.SideBuy,
		Price:     num.NewUint(2100),
	}
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2010), num.NewUint(2020))
	require.Len(t, orders, 3)
	for _, o := range orders {
		assert.Equal(t, "2000", o.Price.String())
		assert.Equal(t, uint64(222), o.Size)
	}
}

func testSubmitOrderAcrossAMMBoundary(t *testing.T) {
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		// going to shrink the boundaries
		submit.Parameters.LowerBound.Add(submit.Parameters.LowerBound, num.NewUint(uint64(i*50)))
		submit.Parameters.UpperBound.Sub(submit.Parameters.UpperBound, num.NewUint(uint64(i*50)))

		expectSubaccountCreation(t, tst, party, subAccount)
		whenAMMIsSubmitted(t, tst, submit)
	}

	ensureBalancesN(t, tst.col, 10000000000, -1)
	ensurePositionN(t, tst.pos, 0, num.NewUint(0), -1)

	// now submit an order against it
	agg := &types.Order{
		Size:      1000000000000,
		Remaining: 1000000000000,
		Side:      types.SideBuy,
		Price:     num.NewUint(2200),
	}

	// pools upper boundaries are 2100, 2150, 2200, and we submit a big order
	// we expect to trade with each pool in these three chunks
	// - first 3 orders with all pools from [2000, 2100]
	// - then 2 orders with the longer two pools from [2100, 2150]
	// - then 1 order just the last pool from [2150, 2200]
	// so 6 orders in total
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2000), num.NewUint(2200))
	require.Len(t, orders, 6)

	// first round, three orders moving all pool's to the upper boundary of the shortest
	assert.Equal(t, "2049", orders[0].Price.String())
	assert.Equal(t, "2049", orders[1].Price.String())
	assert.Equal(t, "2049", orders[2].Price.String())

	// second round, 2 orders moving all pool's to the upper boundary of the second shortest
	assert.Equal(t, "2124", orders[3].Price.String())
	assert.Equal(t, "2124", orders[4].Price.String())

	// third round, 1 orders moving the last pool to its boundary
	assert.Equal(t, "2174", orders[5].Price.String())
}

func testSubmitOrderAcrossAMMBoundarySell(t *testing.T) {
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		// going to shrink the boundaries
		submit.Parameters.LowerBound.Add(submit.Parameters.LowerBound, num.NewUint(uint64(i*50)))
		submit.Parameters.UpperBound.Sub(submit.Parameters.UpperBound, num.NewUint(uint64(i*50)))

		expectSubaccountCreation(t, tst, party, subAccount)
		whenAMMIsSubmitted(t, tst, submit)
	}

	ensureBalancesN(t, tst.col, 10000000000, -1)
	ensurePositionN(t, tst.pos, 0, num.NewUint(0), -1)

	// now submit an order against it
	agg := &types.Order{
		Size:      1000000000000,
		Remaining: 1000000000000,
		Side:      types.SideSell,
		Price:     num.NewUint(1800),
	}

	// pools lower boundaries are 1800, 1850, 1900, and we submit a big order
	// we expect to trade with each pool in these three chunks
	// - first 3 orders with all pools from [2000, 1900]
	// - then 2 orders with the longer two pools from [1900, 1850]
	// - then 1 order just the last pool from [1850, 1800]
	// so 6 orders in total
	// orders := tst.engine.SubmitOrder(agg, num.NewUint(2000), num.NewUint(1800))
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2000), num.NewUint(1800))
	require.Len(t, orders, 6)

	// first round, three orders moving all pool's to the upper boundary of the shortest
	assert.Equal(t, "1949", orders[0].Price.String())
	assert.Equal(t, "1949", orders[1].Price.String())
	assert.Equal(t, "1949", orders[2].Price.String())

	// second round, 2 orders moving all pool's to the upper boundary of the second shortest
	assert.Equal(t, "1874", orders[3].Price.String())
	assert.Equal(t, "1874", orders[4].Price.String())

	// third round, 1 orders moving the last pool to its boundary
	assert.Equal(t, "1824", orders[5].Price.String())
}

func testBestPricesAndVolume(t *testing.T) {
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		expectSubaccountCreation(t, tst, party, subAccount)
		whenAMMIsSubmitted(t, tst, submit)
	}

	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).AnyTimes().Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.NewUint(0)}},
	)

	bid, bvolume, ask, avolume := tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "1999", bid.String())
	assert.Equal(t, "2001", ask.String())
	assert.Equal(t, 37512, int(bvolume))
	assert.Equal(t, 35781, int(avolume))

	// test GetVolumeAtPrice returns the same volume given best bid/ask
	bvAt := tst.engine.GetVolumeAtPrice(bid, types.SideSell)
	assert.Equal(t, bvolume, bvAt)
	avAt := tst.engine.GetVolumeAtPrice(ask, types.SideBuy)
	assert.Equal(t, avolume, avAt)
}

func TestBestPricesAndVolumeNearBound(t *testing.T) {
	tst := getTestEngineWithFactors(t, num.DecimalFromInt64(100), num.DecimalFromFloat(10), 0)

	// create three pools
	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(10).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.NewUint(0)}},
	)

	bid, bvolume, ask, avolume := tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "199900", bid.String())
	assert.Equal(t, "200100", ask.String())
	assert.Equal(t, 1250, int(bvolume))
	assert.Equal(t, 1192, int(avolume))

	// lets move its position so that the fair price is within one tick of the AMMs upper boundary
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(10).Return(
		[]events.MarketPosition{&marketPosition{size: -222000, averageEntry: num.NewUint(0)}},
	)

	bid, bvolume, ask, avolume = tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "219890", bid.String())
	assert.Equal(t, "220000", ask.String()) // make sure we are capped to the boundary and not 220090
	assert.Equal(t, 1034, int(bvolume))
	assert.Equal(t, 104, int(avolume))

	// lets move its position so that the fair price is within one tick of the AMMs upper boundary
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(10).Return(
		[]events.MarketPosition{&marketPosition{size: 270400, averageEntry: num.NewUint(0)}},
	)

	bid, bvolume, ask, avolume = tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "180000", bid.String()) // make sure we are capped to the boundary and not 179904
	assert.Equal(t, "180104", ask.String())
	assert.Equal(t, 62, int(bvolume))
	assert.Equal(t, 1460, int(avolume))
}

func testClosingReduceOnlyPool(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// pool position is zero it should get removed right away with no fuss
	ensurePosition(t, tst.pos, 0, num.UintZero())
	ensurePosition(t, tst.pos, 0, num.UintZero())
	expectSubAccountRelease(t, tst, party, subAccount)
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMCancellationMethodReduceOnly)
	mevt, err := tst.engine.CancelAMM(ctx, cancel)
	require.NoError(t, err)
	assert.Nil(t, mevt) // no closeout necessary so not event
	tst.engine.OnMTM(ctx)
	assert.Len(t, tst.engine.pools, 0)
}

func testClosingPoolImmediate(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// pool has a position but gets closed anyway
	ensurePosition(t, tst.pos, 12, num.UintZero())
	expectSubAccountRelease(t, tst, party, subAccount)
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMCancellationMethodImmediate)
	mevt, err := tst.engine.CancelAMM(ctx, cancel)
	require.NoError(t, err)
	assert.Nil(t, mevt) // no closeout necessary so not event
	assert.Len(t, tst.engine.pools, 0)
}

func testAmendMakesClosingPoolActive(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// pool position is non-zero so it''l hang around
	ensurePosition(t, tst.pos, 12, num.UintZero())
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMCancellationMethodReduceOnly)
	closeout, err := tst.engine.CancelAMM(ctx, cancel)
	require.NoError(t, err)
	assert.Nil(t, closeout)
	tst.engine.OnMTM(ctx)
	assert.Len(t, tst.engine.pools, 1)
	assert.True(t, tst.engine.poolsCpy[0].closing())

	amend := getPoolAmendment(t, party, tst.marketID)
	expectBalanceChecks(t, tst, party, subAccount, amend.CommitmentAmount.Uint64())
	ensurePosition(t, tst.pos, 0, num.UintZero())
	updated, _, err := tst.engine.Amend(ctx, amend, riskFactors, scalingFactors, slippage)
	require.NoError(t, err)
	tst.engine.Confirm(ctx, updated)

	// pool is active again
	assert.False(t, tst.engine.poolsCpy[0].closing())
}

func testClosingPoolRemovedWhenPositionZero(t *testing.T) {
	ctx := vgcontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// pool position is non-zero so it''l hang around
	ensurePosition(t, tst.pos, 12, num.UintZero())
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMCancellationMethodReduceOnly)
	closeout, err := tst.engine.CancelAMM(ctx, cancel)
	require.NoError(t, err)
	assert.Nil(t, closeout)
	tst.engine.OnMTM(ctx)
	assert.True(t, tst.engine.poolsCpy[0].closing())

	// position is lower but non-zero
	ensurePosition(t, tst.pos, 1, num.UintZero())
	tst.engine.OnMTM(ctx)
	assert.True(t, tst.engine.poolsCpy[0].closing())

	// position is zero, it will get removed
	ensurePositionN(t, tst.pos, 0, num.UintZero(), 2)
	expectSubAccountRelease(t, tst, party, subAccount)
	tst.engine.OnMTM(ctx)
	assert.Len(t, tst.engine.poolsCpy, 0)
}

func testStoppingDistressedAMM(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	// call remove distressed with a AMM's owner will not remove the pool
	closed := []events.MarketPosition{
		mpos{party},
	}
	tst.engine.RemoveDistressed(ctx, closed)
	assert.Len(t, tst.engine.pools, 1)

	// call remove distressed with a AMM's subacouunt will remove the pool
	closed = []events.MarketPosition{
		mpos{subAccount},
	}
	tst.engine.RemoveDistressed(ctx, closed)
	assert.Len(t, tst.engine.pools, 0)
}

func testAMMWithNoBalanceStopped(t *testing.T) {
	ctx := vgcontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)
	ensureBalances(t, tst.col, 10000)
	tst.engine.OnTick(ctx, time.Now())
	assert.Len(t, tst.engine.pools, 1)

	ensureBalances(t, tst.col, 0)
	tst.engine.OnTick(ctx, time.Now())
	assert.Len(t, tst.engine.pools, 0)
}

func testMarketClosure(t *testing.T) {
	ctx := vgcontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tst := getTestEngine(t)

	for i := 0; i < 10; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		expectSubaccountCreation(t, tst, party, subAccount)
		whenAMMIsSubmitted(t, tst, submit)
		expectSubAccountClose(t, tst, party, subAccount)
	}

	require.NoError(t, tst.engine.MarketClosing(ctx))
	require.Equal(t, 0, len(tst.engine.pools))
	require.Equal(t, 0, len(tst.engine.poolsCpy))
	require.Equal(t, 0, len(tst.engine.ammParties))
}

func testSparseAMMEngine(t *testing.T) {
	tst := getTestEngineWithFactors(t, num.DecimalOne(), num.DecimalOne(), 10)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	submit.CommitmentAmount = num.NewUint(100000)

	expectSubaccountCreation(t, tst, party, subAccount)
	whenAMMIsSubmitted(t, tst, submit)

	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).AnyTimes().Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: nil}},
	)
	bb, bv, ba, av := tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "1992", bb.String())
	assert.Equal(t, 1, int(bv))
	assert.Equal(t, "2009", ba.String())
	assert.Equal(t, 1, int(av))
}

func testAMMSnapshot(t *testing.T) {
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		expectSubaccountCreation(t, tst, party, subAccount)
		whenAMMIsSubmitted(t, tst, submit)
	}

	ensurePositionN(t, tst.pos, 0, num.NewUint(0), 3)

	// now submit an order against it
	agg := &types.Order{
		Size:      666,
		Remaining: 666,
		Side:      types.SideBuy,
		Price:     num.NewUint(2100),
	}
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2010), num.NewUint(2020))
	require.Len(t, orders, 3)
	for _, o := range orders {
		assert.Equal(t, "2000", o.Price.String())
		assert.Equal(t, uint64(222), o.Size)
	}

	bb1, bv1, ba1, av1 := tst.engine.BestPricesAndVolumes()

	// now snapshot
	state := tst.engine.IntoProto()
	tst2 := getTestEngineWithProto(t, state)

	// now do some stuff with it
	ensurePositionN(t, tst2.pos, -222, num.NewUint(0), -1)
	bb2, bv2, ba2, av2 := tst2.engine.BestPricesAndVolumes()
	assert.Equal(t, bb1, bb2)
	assert.Equal(t, bv1, bv2)
	assert.Equal(t, ba1, ba2)
	assert.Equal(t, av1, av2)

	// now submit an order against it
	agg = &types.Order{
		Size:      666,
		Remaining: 666,
		Side:      types.SideSell,
		Price:     num.NewUint(1000),
	}
	orders = tst2.engine.SubmitOrder(agg, nil, nil)
	require.Len(t, orders, 3)
	for _, o := range orders {
		assert.Equal(t, "2000", o.Price.String())
		assert.Equal(t, uint64(222), o.Size)
	}
}

func expectSubaccountCreation(t *testing.T, tst *tstEngine, party, subAccount string) {
	t.Helper()

	// accounts are created
	tst.col.EXPECT().CreatePartyAMMsSubAccounts(gomock.Any(), party, subAccount, tst.assetID, tst.marketID).Times(1)
}

func expectSubAccountRelease(t *testing.T, tst *tstEngine, party, subAccount string) {
	t.Helper()
	// account is update from party's main accounts
	tst.col.EXPECT().SubAccountRelease(
		gomock.Any(),
		party,
		subAccount,
		tst.assetID,
		tst.marketID,
		gomock.Any(),
	).Times(1).Return([]*types.LedgerMovement{}, nil, nil)
}

func expectSubAccountClose(t *testing.T, tst *tstEngine, party, subAccount string) {
	t.Helper()
	tst.col.EXPECT().SubAccountClosed(
		gomock.Any(),
		party,
		subAccount,
		tst.assetID,
		tst.marketID).Times(1).Return([]*types.LedgerMovement{}, nil)
}

func expectBalanceChecks(t *testing.T, tst *tstEngine, party, subAccount string, total uint64) {
	t.Helper()
	tst.col.EXPECT().GetPartyMarginAccount(tst.marketID, subAccount, tst.assetID).Times(1).Return(getAccount(0), nil)
	tst.col.EXPECT().GetPartyGeneralAccount(subAccount, tst.assetID).Times(1).Return(getAccount(0), nil)
	tst.col.EXPECT().GetPartyGeneralAccount(party, tst.assetID).Times(1).Return(getAccount(total), nil)
}

func whenAMMIsSubmitted(t *testing.T, tst *tstEngine, submission *types.SubmitAMM) {
	t.Helper()

	party := submission.Party
	subAccount := DeriveAMMParty(party, tst.marketID, "AMMv1", 0)
	expectBalanceChecks(t, tst, party, subAccount, submission.CommitmentAmount.Uint64())

	ensurePosition(t, tst.pos, 0, nil)

	ctx := context.Background()
	pool, err := tst.engine.Create(ctx, submission, vgcrypto.RandomHash(), riskFactors, scalingFactors, slippage)
	require.NoError(t, err)
	tst.engine.Confirm(ctx, pool)
}

func getParty(t *testing.T, tst *tstEngine) (string, string) {
	t.Helper()

	party := vgcrypto.RandomHash()
	subAccount := DeriveAMMParty(party, tst.marketID, "AMMv1", 0)
	return party, subAccount
}

func getPoolSubmission(t *testing.T, party, market string) *types.SubmitAMM {
	t.Helper()
	return &types.SubmitAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			Party:             party,
			MarketID:          market,
			SlippageTolerance: num.DecimalFromFloat(0.1),
		},
		CommitmentAmount: num.NewUint(10000000000),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 num.NewUint(2000),
			LowerBound:           num.NewUint(1800),
			UpperBound:           num.NewUint(2200),
			LeverageAtLowerBound: ptr.From(num.DecimalOne()),
			LeverageAtUpperBound: ptr.From(num.DecimalOne()),
		},
	}
}

func getPoolAmendment(t *testing.T, party, market string) *types.AmendAMM {
	t.Helper()
	return &types.AmendAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			Party:             party,
			MarketID:          market,
			SlippageTolerance: num.DecimalFromFloat(0.1),
		},
		CommitmentAmount: num.NewUint(10000000000),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 num.NewUint(2100),
			LowerBound:           num.NewUint(1900),
			UpperBound:           num.NewUint(2300),
			LeverageAtLowerBound: ptr.From(num.DecimalOne()),
			LeverageAtUpperBound: ptr.From(num.DecimalOne()),
		},
	}
}

func getCancelSubmission(t *testing.T, party, market string, method types.AMMCancellationMethod) *types.CancelAMM {
	t.Helper()
	return &types.CancelAMM{
		MarketID: market,
		Party:    party,
		Method:   method,
	}
}

type tstEngine struct {
	engine  *Engine
	broker  *bmocks.MockBroker
	col     *mocks.MockCollateral
	pos     *mocks.MockPosition
	parties *cmocks.MockParties
	ctrl    *gomock.Controller

	marketID string
	assetID  string
}

func getTestEngineWithFactors(t *testing.T, priceFactor, positionFactor num.Decimal, allowedEmptyLevels uint64) *tstEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	pos := mocks.NewMockPosition(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	marketID := vgcrypto.RandomHash()
	assetID := vgcrypto.RandomHash()

	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	col.EXPECT().GetAssetQuantum(assetID).AnyTimes().Return(num.DecimalOne(), nil)

	teams := cmocks.NewMockTeams(ctrl)
	balanceChecker := cmocks.NewMockAccountBalanceChecker(ctrl)

	mat := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, col)

	parties := cmocks.NewMockParties(ctrl)
	parties.EXPECT().AssignDeriveKey(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	eng := New(logging.NewTestLogger(), broker, col, marketID, assetID, pos, priceFactor, positionFactor, mat, parties, allowedEmptyLevels)

	// do an ontick to initialise the idgen
	ctx := vgcontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	eng.OnTick(ctx, time.Now())

	return &tstEngine{
		engine:   eng,
		broker:   broker,
		col:      col,
		pos:      pos,
		ctrl:     ctrl,
		parties:  parties,
		marketID: marketID,
		assetID:  assetID,
	}
}

func getTestEngineWithProto(t *testing.T, state *v1.AmmState) *tstEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := mocks.NewMockCollateral(ctrl)
	pos := mocks.NewMockPosition(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	marketID := vgcrypto.RandomHash()
	assetID := vgcrypto.RandomHash()

	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	col.EXPECT().GetAssetQuantum(assetID).AnyTimes().Return(num.DecimalOne(), nil)

	teams := cmocks.NewMockTeams(ctrl)
	balanceChecker := cmocks.NewMockAccountBalanceChecker(ctrl)

	mat := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, col)

	parties := cmocks.NewMockParties(ctrl)
	parties.EXPECT().AssignDeriveKey(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	priceFactor := num.DecimalOne()
	positionFactor := num.DecimalOne()

	eng, err := NewFromProto(logging.NewTestLogger(), broker, col, marketID, assetID, pos, state, priceFactor, positionFactor, mat, parties, 0)
	require.NoError(t, err)

	return &tstEngine{
		engine:   eng,
		broker:   broker,
		col:      col,
		pos:      pos,
		ctrl:     ctrl,
		parties:  parties,
		marketID: marketID,
		assetID:  assetID,
	}
}

func getTestEngine(t *testing.T) *tstEngine {
	t.Helper()
	return getTestEngineWithFactors(t, num.DecimalOne(), num.DecimalOne(), 0)
}

func getAccount(balance uint64) *types.Account {
	return &types.Account{
		Balance: num.NewUint(balance),
	}
}

type mpos struct {
	party string
}

func (m mpos) AverageEntryPrice() *num.Uint { return num.UintZero() }
func (m mpos) Party() string                { return m.party }
func (m mpos) Size() int64                  { return 0 }
func (m mpos) Buy() int64                   { return 0 }
func (m mpos) Sell() int64                  { return 0 }
func (m mpos) Price() *num.Uint             { return num.UintZero() }
func (m mpos) BuySumProduct() *num.Uint     { return num.UintZero() }
func (m mpos) SellSumProduct() *num.Uint    { return num.UintZero() }
func (m mpos) ClearPotentials()             {}
func (m mpos) VWBuy() *num.Uint             { return num.UintZero() }
func (m mpos) VWSell() *num.Uint            { return num.UintZero() }
