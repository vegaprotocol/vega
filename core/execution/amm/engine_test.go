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
}

func TestAMMTrading(t *testing.T) {
	t.Run("test basic submit order", testBasicSubmitOrder)
	t.Run("test submit market order", testSubmitMarketOrder)
	t.Run("test submit order pro rata", testSubmitOrderProRata)
	t.Run("test best prices and volume", testBestPricesAndVolume)

	t.Run("test submit buy order across AMM boundary", testSubmitOrderAcrossAMMBoundary)
	t.Run("test submit sell order across AMM boundary", testSubmitOrderAcrossAMMBoundarySell)
}

func TestAmendAMM(t *testing.T) {
	t.Run("test amend AMM which doesn't exist", testAmendAMMWhichDoesntExist)
	t.Run("test amend AMM with sparse amend", TestAmendAMMSparse)
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

func TestAmendAMMSparse(t *testing.T) {
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

	updated, _, err := tst.engine.Amend(ctx, amend, riskFactors, scalingFactors, slippage)
	require.NoError(t, err)

	tst.engine.Confirm(ctx, updated)
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

	orders = tst.engine.SubmitOrder(agg, num.NewUint(2020), num.NewUint(1990))
	require.Len(t, orders, 1)
	assert.Equal(t, "2035", orders[0].Price.String())
	// note that this volume being bigger than 242367 above means we've moved back to position, then flipped
	// sign, and took volume from the other curve.
	assert.Equal(t, 362325, int(orders[0].Size))
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
	orders := tst.engine.SubmitOrder(agg, num.NewUint(1980), num.NewUint(1990))
	require.Len(t, orders, 1)
	assert.Equal(t, "2005", orders[0].Price.String())
	assert.Equal(t, 126420, int(orders[0].Size))
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
	assert.Equal(t, "2125", orders[4].Price.String())

	// third round, 1 orders moving the last pool to its boundary
	assert.Equal(t, "2175", orders[5].Price.String())
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
	assert.Equal(t, "2053", orders[0].Price.String())
	assert.Equal(t, "2053", orders[1].Price.String())
	assert.Equal(t, "2053", orders[2].Price.String())

	// second round, 2 orders moving all pool's to the upper boundary of the second shortest
	assert.Equal(t, "1925", orders[3].Price.String())
	assert.Equal(t, "1925", orders[4].Price.String())

	// third round, 1 orders moving the last pool to its boundary
	assert.Equal(t, "1875", orders[5].Price.String())
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

	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(9).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.NewUint(0)}},
	)

	bid, bvolume, ask, avolume := tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "1999", bid.String())
	assert.Equal(t, "2001", ask.String())
	assert.Equal(t, 37512, int(bvolume))
	assert.Equal(t, 35781, int(avolume))

	// test GetVolumeAtPrice returns the same volume given best bid/ask
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(6 * 2).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.NewUint(0)}},
	)
	bvAt := tst.engine.GetVolumeAtPrice(bid, types.SideSell)
	assert.Equal(t, bvolume, bvAt)
	avAt := tst.engine.GetVolumeAtPrice(ask, types.SideBuy)
	assert.Equal(t, avolume, avAt)
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
	for _, p := range tst.engine.poolsCpy {
		assert.Equal(t, types.AMMPoolStatusStopped, p.status)
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

	ctx := context.Background()
	pool, err := tst.engine.Create(ctx, submission, vgcrypto.RandomHash(), riskFactors, scalingFactors, slippage)
	require.NoError(t, err)
	require.NoError(t, tst.engine.Confirm(ctx, pool))
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
	engine *Engine
	broker *bmocks.MockBroker
	col    *mocks.MockCollateral
	pos    *mocks.MockPosition
	ctrl   *gomock.Controller

	marketID string
	assetID  string
}

func getTestEngine(t *testing.T) *tstEngine {
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

	mat := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker)

	eng := New(logging.NewTestLogger(), broker, col, marketID, assetID, pos, num.DecimalOne(), num.DecimalOne(), mat)

	// do an ontick to initialise the idgen
	ctx := vgcontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	eng.OnTick(ctx, time.Now())

	return &tstEngine{
		engine:   eng,
		broker:   broker,
		col:      col,
		pos:      pos,
		ctrl:     ctrl,
		marketID: marketID,
		assetID:  assetID,
	}
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
