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

func TestSubmitAMM(t *testing.T) {
	t.Run("test one pool per party", testOnePoolPerParty)
	t.Run("test rebase on submit", testRebaseOnSubmit)
	t.Run("test rebase on submit order fails", testRebaseOnSubmitOrderFails)
	t.Run("test rebase on submit order did not trade", testRebaseOnSubmitOrderDidNotTrade)
	t.Run("test rebase on submit order target is base", testSubmitTargetIsBase)
	t.Run("test rebase on submit order target out of bounds", testSubmitTargetIsOutOfBounds)
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
	t.Run("test amend AMM with rebase", testAmendAMMWithRebase)
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
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	// when the party submits another, it is rejected
	err := tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil)
	require.ErrorContains(t, err, "party already own a pool for market")
}

func testRebaseOnSubmit(t *testing.T) {
	tst := getTestEngine(t)
	ctx := context.Background()

	// the party will make this AMM submission
	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)

	// where the mark-price is away from the base price
	target := num.NewUint(2100)
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(2).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.UintZero()}},
	)

	// so that an order will be submitting to rebase the pool
	expectOrderSubmission(t, tst, subAccount, types.OrderStatusFilled, nil)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), target))
}

func testSubmitTargetIsBase(t *testing.T) {
	tst := getTestEngine(t)
	ctx := context.Background()

	// the party will make this AMM submission
	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)

	// where the mark-price is the same as the base price
	target := num.NewUint(2000)
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(1).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.UintZero()}},
	)

	// so no rebasing order is necessary
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), target))
}

func testSubmitTargetIsOutOfBounds(t *testing.T) {
	tst := getTestEngine(t)
	ctx := context.Background()

	// the party will make this AMM submission
	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)

	// where the mark-price is the same as the base price
	target := num.NewUint(1)

	// the submission will fail so subaccount will be emptied
	expectSubAccountUpdate(t, tst, party, subAccount, 1000)
	err := tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), target)
	require.ErrorIs(t, ErrRebaseTargetOutsideBounds, err)
}

func testRebaseOnSubmitOrderFails(t *testing.T) {
	tst := getTestEngine(t)
	ctx := context.Background()

	// the party will make this AMM submission
	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)

	// where the mark-price is away from the base price
	target := num.NewUint(2100)
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(2).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.UintZero()}},
	)

	// so that an order will be submitting to rebase the pool
	expectOrderSubmission(t, tst, subAccount, types.OrderStatusStopped, assert.AnError)

	// the subaccount balances will be reverted
	expectSubAccountUpdate(t, tst, party, subAccount, 1000)

	err := tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), target)
	require.ErrorIs(t, err, assert.AnError)
}

func testRebaseOnSubmitOrderDidNotTrade(t *testing.T) {
	tst := getTestEngine(t)
	ctx := context.Background()

	// the party will make this AMM submission
	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)
	expectSubaccountCreation(t, tst, party, subAccount)

	// where the mark-price is away from the base price
	target := num.NewUint(2100)
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(2).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.UintZero()}},
	)

	// so that an order will be submitting to rebase the pool
	expectOrderSubmission(t, tst, subAccount, types.OrderStatusStopped, nil)

	// the subaccount balances will be reverted
	expectSubAccountUpdate(t, tst, party, subAccount, 1000)

	err := tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), target)
	require.ErrorIs(t, err, ErrRebaseOrderDidNotTrade)
}

func testAmendAMMWhichDoesntExist(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	// make an amend when the party doesn't have a pool
	party, _ := getParty(t, tst)
	amend := getPoolAmendment(t, party, tst.marketID)

	err := tst.engine.AmendAMM(ctx, amend, vgcrypto.RandomHash())
	require.ErrorIs(t, err, ErrNoPoolMatchingParty)
}

func testAmendAMMWithRebase(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	// now amend it so that we have to rebase the pool
	amend := getPoolAmendment(t, party, tst.marketID)

	expectSubAccountUpdate(t, tst, party, subAccount, 1000)
	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(3).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.UintZero()}},
	)

	// so that an order will be submitting to rebase the pool
	expectOrderSubmission(t, tst, subAccount, types.OrderStatusFilled, nil)

	err := tst.engine.AmendAMM(ctx, amend, vgcrypto.RandomHash())
	require.NoError(t, err)
}

func testBasicSubmitOrder(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	// now submit an order against it
	agg := &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideBuy,
		Price:     num.NewUint(2100),
		Type:      types.OrderTypeLimit,
	}

	ensureBalances(t, tst.col, 10000000000)
	ensurePosition(t, tst.pos, 0, num.NewUint(0))
	orders := tst.engine.SubmitOrder(agg, num.NewUint(2000), num.NewUint(2020))
	require.Len(t, orders, 1)
	assert.Equal(t, "2009", orders[0].Price.String())
	assert.Equal(t, uint64(242367), orders[0].Size)

	// AMM is now short, but another order comes in that will flip its position to long
	agg = &types.Order{
		Size:      1000000,
		Remaining: 1000000,
		Side:      types.SideSell,
		Price:     num.NewUint(1900),
	}

	ensureBalancesN(t, tst.col, 10000000000, 2)
	orders = tst.engine.SubmitOrder(agg, num.NewUint(2020), num.NewUint(1990))
	require.Len(t, orders, 1)
	assert.Equal(t, "2035", orders[0].Price.String())
	// note that this volume being bigger than 242367 above means we've moved back to position, then flipped
	// sign, and took volume from the other curve.
	assert.Equal(t, uint64(371231), orders[0].Size)
}

func testSubmitMarketOrder(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

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
	assert.Equal(t, uint64(129839), orders[0].Size)
}

func testSubmitOrderProRata(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		expectSubaccountCreation(t, tst, party, subAccount)
		require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))
	}

	ensureBalancesN(t, tst.col, 10000000000, 3)
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
	ctx := context.Background()
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		// going to shrink the boundaries
		submit.Parameters.LowerBound.Add(submit.Parameters.LowerBound, num.NewUint(uint64(i*50)))
		submit.Parameters.UpperBound.Sub(submit.Parameters.UpperBound, num.NewUint(uint64(i*50)))

		expectSubaccountCreation(t, tst, party, subAccount)
		require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))
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
	ctx := context.Background()
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		// going to shrink the boundaries
		submit.Parameters.LowerBound.Add(submit.Parameters.LowerBound, num.NewUint(uint64(i*50)))
		submit.Parameters.UpperBound.Sub(submit.Parameters.UpperBound, num.NewUint(uint64(i*50)))

		expectSubaccountCreation(t, tst, party, subAccount)
		require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))
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
	assert.Equal(t, "1923", orders[3].Price.String())
	assert.Equal(t, "1923", orders[4].Price.String())

	// third round, 1 orders moving the last pool to its boundary
	assert.Equal(t, "1872", orders[5].Price.String())
}

func testBestPricesAndVolume(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	// create three pools
	for i := 0; i < 3; i++ {
		party, subAccount := getParty(t, tst)
		submit := getPoolSubmission(t, party, tst.marketID)

		expectSubaccountCreation(t, tst, party, subAccount)
		require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))
	}

	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(9).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.NewUint(0)}},
	)

	bid, bvolume, ask, avolume := tst.engine.BestPricesAndVolumes()
	assert.Equal(t, "1999", bid.String())
	assert.Equal(t, "2001", ask.String())
	assert.Equal(t, uint64(38526), bvolume)
	assert.Equal(t, uint64(36615), avolume)
}

func testClosingReduceOnlyPool(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	// pool position is zero it should get removed right away with no fuss
	ensurePosition(t, tst.pos, 0, num.UintZero())
	ensurePosition(t, tst.pos, 0, num.UintZero())
	expectSubAccountRelease(t, tst, party, subAccount)
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMPoolCancellationMethodReduceOnly)
	mevt, err := tst.engine.CancelAMM(ctx, cancel)
	require.NoError(t, err)
	assert.Nil(t, mevt) // no closeout necessary so not event
	assert.Len(t, tst.engine.pools, 0)
}

func testClosingPoolImmediate(t *testing.T) {
	ctx := context.Background()
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	// pool has a position but gets closed anyway
	ensurePosition(t, tst.pos, 12, num.UintZero())
	expectSubAccountRelease(t, tst, party, subAccount)
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMPoolCancellationMethodImmediate)
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
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	// pool position is non-zero so it''l hang around
	ensurePosition(t, tst.pos, 12, num.UintZero())
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMPoolCancellationMethodReduceOnly)
	closeout, err := tst.engine.CancelAMM(ctx, cancel)
	require.NoError(t, err)
	assert.Nil(t, closeout)
	assert.Len(t, tst.engine.pools, 1)
	assert.True(t, tst.engine.poolsCpy[0].closing())

	tst.pos.EXPECT().GetPositionsByParty(gomock.Any()).Times(3).Return(
		[]events.MarketPosition{&marketPosition{size: 0, averageEntry: num.UintZero()}},
	)
	expectSubAccountUpdate(t, tst, party, subAccount, 1000)
	amend := getPoolAmendment(t, party, tst.marketID)
	require.NoError(t, tst.engine.AmendAMM(ctx, amend, vgcrypto.RandomHash()))

	// pool is active again
	assert.False(t, tst.engine.poolsCpy[0].closing())
}

func testClosingPoolRemovedWhenPositionZero(t *testing.T) {
	ctx := vgcontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tst := getTestEngine(t)

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	// pool position is non-zero so it''l hang around
	ensurePosition(t, tst.pos, 12, num.UintZero())
	cancel := getCancelSubmission(t, party, tst.marketID, types.AMMPoolCancellationMethodReduceOnly)
	closeout, err := tst.engine.CancelAMM(ctx, cancel)
	require.NoError(t, err)
	assert.Nil(t, closeout)
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
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

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
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))
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

	party, subAccount := getParty(t, tst)
	submit := getPoolSubmission(t, party, tst.marketID)

	expectSubaccountCreation(t, tst, party, subAccount)
	require.NoError(t, tst.engine.SubmitAMM(ctx, submit, vgcrypto.RandomHash(), nil))

	ensurePosition(t, tst.pos, 0, num.UintZero())
	expectSubAccountRelease(t, tst, party, subAccount)
	require.NoError(t, tst.engine.MarketClosing(ctx))
	assert.Len(t, tst.engine.poolsCpy, 0)
}

func expectSubaccountCreation(t *testing.T, tst *tstEngine, party, subAccount string) {
	t.Helper()

	// accounts are created
	tst.col.EXPECT().CreatePartyAMMsSubAccounts(gomock.Any(), party, subAccount, tst.assetID, tst.marketID).Times(1)

	expectSubAccountUpdate(t, tst, party, subAccount, 0)
}

func expectSubAccountUpdate(t *testing.T, tst *tstEngine, party, subAccount string, balance uint64) {
	t.Helper()

	// sub-account starts with zero balance
	tst.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(getAccount(balance), nil)
	tst.col.EXPECT().GetPartyMarginAccount(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(getAccount(balance), nil)

	// account is update from party's main accounts
	tst.col.EXPECT().SubAccountUpdate(
		gomock.Any(),
		party,
		subAccount,
		tst.assetID,
		tst.marketID,
		gomock.Any(),
		gomock.Any(),
	).Times(1).Return(&types.LedgerMovement{}, nil)
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

func expectOrderSubmission(t *testing.T, tst *tstEngine, subAccount string, status types.OrderStatus, err error) {
	t.Helper()

	conf := &types.OrderConfirmation{
		Order: &types.Order{
			Status: status,
		},
	}
	tst.market.EXPECT().SubmitOrderWithIDGeneratorAndOrderID(
		gomock.Any(),
		gomock.Any(),
		subAccount,
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(1).Return(conf, err)
}

func getParty(t *testing.T, tst *tstEngine) (string, string) {
	t.Helper()

	party := vgcrypto.RandomHash()
	subAccount := DeriveSubAccount(party, tst.marketID, "AMMv1", 0)
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
			Base:                    num.NewUint(2000),
			LowerBound:              num.NewUint(1800),
			UpperBound:              num.NewUint(2200),
			MarginRatioAtLowerBound: ptr.From(num.DecimalOne()),
			MarginRatioAtUpperBound: ptr.From(num.DecimalOne()),
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
			Base:                    num.NewUint(2100),
			LowerBound:              num.NewUint(1900),
			UpperBound:              num.NewUint(2300),
			MarginRatioAtLowerBound: ptr.From(num.DecimalOne()),
			MarginRatioAtUpperBound: ptr.From(num.DecimalOne()),
		},
	}
}

func getCancelSubmission(t *testing.T, party, market string, method types.AMMPoolCancellationMethod) *types.CancelAMM {
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
	market *mocks.MockMarket
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
	market := mocks.NewMockMarket(ctrl)
	risk := mocks.NewMockRisk(ctrl)
	broker := bmocks.NewMockBroker(ctrl)

	marketID := vgcrypto.RandomHash()
	assetID := vgcrypto.RandomHash()

	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	market.EXPECT().GetID().AnyTimes().Return(marketID)
	market.EXPECT().GetSettlementAsset().AnyTimes().Return(assetID)
	col.EXPECT().GetAssetQuantum(assetID).AnyTimes().Return(num.DecimalOne(), nil)

	risk.EXPECT().GetRiskFactors().AnyTimes().Return(&types.RiskFactor{Market: "", Short: num.DecimalOne(), Long: num.DecimalOne()})
	risk.EXPECT().GetScalingFactors().AnyTimes().Return(&types.ScalingFactors{InitialMargin: num.DecimalOne()})
	risk.EXPECT().GetSlippage().AnyTimes().Return(num.DecimalOne())

	teams := cmocks.NewMockTeams(ctrl)
	balanceChecker := cmocks.NewMockAccountBalanceChecker(ctrl)

	mat := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker)

	eng := New(logging.NewTestLogger(), broker, col, market, risk, pos, num.UintOne(), num.DecimalOne(), mat)

	// do an ontick to initialise the idgen
	ctx := vgcontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	eng.OnTick(ctx, time.Now())

	return &tstEngine{
		engine:   eng,
		broker:   broker,
		market:   market,
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
