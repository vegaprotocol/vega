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

package liquidity_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmissions(t *testing.T) {
	t.Run("Create and cancel", testSubmissionCreateAndCancel)
	t.Run("Cancel non existing", testCancelNonExistingSubmission)
	t.Run("Can to submit when current or pending LP exists", testFailsWhenLPExists)
}

func testFailsWhenLPExists(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		te    = newTestEngine(t)
	)
	defer te.ctrl.Finish()

	require.Nil(t, te.engine.LiquidityProvisionByPartyID("some-party"))

	lps1 := &commandspb.LiquidityProvisionSubmission{
		MarketId: te.marketID, CommitmentAmount: "100", Fee: "0.5",
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lps1)
	require.NoError(t, err)

	deterministicID := crypto.RandomHash()
	idGen := idgeneration.New(deterministicID)

	lpID := idGen.NextID()
	now := te.tsvc.GetTimeNow()
	nowNano := now.UnixNano()

	expected := &types.LiquidityProvision{
		ID:               lpID,
		MarketID:         te.marketID,
		Party:            party,
		Fee:              num.DecimalFromFloat(0.5),
		CommitmentAmount: lps.CommitmentAmount.Clone(),
		CreatedAt:        nowNano,
		UpdatedAt:        nowNano,
		Status:           types.LiquidityProvisionStatusActive,
		Version:          1,
	}

	// Creating a submission should fire an event
	te.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	te.auctionState.EXPECT().IsOpeningAuction().Return(false).AnyTimes()

	idgen := idgeneration.New(deterministicID)
	_, err = te.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	require.NoError(t, err)

	// first validate that the amendment is pending
	pendingLp := te.engine.PendingProvisionByPartyID(party)
	assert.Equal(t, expected.CommitmentAmount.String(), pendingLp.CommitmentAmount.String())
	assert.Equal(t, expected.Fee.String(), pendingLp.Fee.String())

	idgen = idgeneration.New(deterministicID)

	lps.CommitmentAmount = num.NewUint(1000)
	_, err = te.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	require.Error(t, err, "liquidity provision already exists")
}

func testSubmissionCreateAndCancel(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		te    = newTestEngine(t)
	)
	defer te.ctrl.Finish()

	require.Nil(t, te.engine.LiquidityProvisionByPartyID("some-party"))

	lps1 := &commandspb.LiquidityProvisionSubmission{
		MarketId: te.marketID, CommitmentAmount: "100", Fee: "0.5",
	}
	lps, err := types.LiquidityProvisionSubmissionFromProto(lps1)
	require.NoError(t, err)

	deterministicID := crypto.RandomHash()
	idGen := idgeneration.New(deterministicID)

	lpID := idGen.NextID()
	now := te.tsvc.GetTimeNow()
	nowNano := now.UnixNano()

	expected := &types.LiquidityProvision{
		ID:               lpID,
		MarketID:         te.marketID,
		Party:            party,
		Fee:              num.DecimalFromFloat(0.5),
		CommitmentAmount: lps.CommitmentAmount.Clone(),
		CreatedAt:        nowNano,
		UpdatedAt:        nowNano,
		Status:           types.LiquidityProvisionStatusActive,
		Version:          1,
	}

	// Creating a submission should fire an event
	te.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	te.auctionState.EXPECT().IsOpeningAuction().Return(false).AnyTimes()

	idgen := idgeneration.New(deterministicID)
	_, err = te.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	require.NoError(t, err)

	// first validate that the amendment is pending
	pendingLp := te.engine.PendingProvisionByPartyID(party)
	assert.Equal(t, expected.CommitmentAmount.String(), pendingLp.CommitmentAmount.String())
	assert.Equal(t, expected.Fee.String(), pendingLp.Fee.String())

	got := te.engine.LiquidityProvisionByPartyID(party)
	require.Nil(t, got)

	zero := num.UintZero()

	te.engine.ResetSLAEpoch(now, zero, zero, num.DecimalZero())
	te.engine.ApplyPendingProvisions(ctx, now)

	got = te.engine.LiquidityProvisionByPartyID(party)
	require.Equal(t, expected.CommitmentAmount.String(), got.CommitmentAmount.String())
	require.Equal(t, expected.Fee, got.Fee)
	require.Equal(t, expected.Version, got.Version)

	expected.Status = types.LiquidityProvisionStatusCancelled
	te.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).AnyTimes()

	err = te.engine.CancelLiquidityProvision(ctx, party)
	require.NoError(t, err)
	require.Nil(t, te.engine.LiquidityProvisionByPartyID(party),
		"Party '%s' should not be a LiquidityProvider after Committing 0 amount", party)
}

func testCancelNonExistingSubmission(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		tng   = newTestEngine(t)
	)
	defer tng.ctrl.Finish()

	err := tng.engine.CancelLiquidityProvision(ctx, party)
	require.Error(t, err)
}

func TestCalculateSuppliedStake(t *testing.T) {
	var (
		party1 = "party-1"
		party2 = "party-2"
		party3 = "party-3"
		ctx    = context.Background()
		tng    = newTestEngine(t)
	)
	defer tng.ctrl.Finish()

	// We don't care about the following calls
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	tng.auctionState.EXPECT().IsOpeningAuction().Return(false).AnyTimes()

	zero := num.UintZero()
	tng.orderbook.EXPECT().GetBestStaticBidPrice().Return(zero, nil).AnyTimes()
	tng.orderbook.EXPECT().GetBestStaticAskPrice().Return(zero, nil).AnyTimes()

	tng.auctionState.EXPECT().InAuction().Return(false).AnyTimes()

	// Send a submission
	lp1pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "100", Fee: "0.5",
	}
	lp1, err := types.LiquidityProvisionSubmissionFromProto(lp1pb)
	require.NoError(t, err)

	idgen := idgeneration.New(crypto.RandomHash())
	_, err = tng.engine.SubmitLiquidityProvision(ctx, lp1, party1, idgen)
	require.NoError(t, err)

	now := tng.tsvc.GetTimeNow()

	tng.engine.ApplyPendingProvisions(ctx, now)
	tng.engine.ResetSLAEpoch(time.Now(), zero, zero, num.DecimalOne())

	suppliedStake := tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount, suppliedStake)

	lp2pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "500", Fee: "0.5",
	}
	lp2, err := types.LiquidityProvisionSubmissionFromProto(lp2pb)
	require.NoError(t, err)

	idgen = idgeneration.New(crypto.RandomHash())
	_, err = tng.engine.SubmitLiquidityProvision(ctx, lp2, party2, idgen)
	require.NoError(t, err)

	tng.engine.ResetSLAEpoch(now, zero, zero, num.DecimalZero())
	tng.engine.ApplyPendingProvisions(ctx, now)

	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount), suppliedStake)

	lp3pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "962", Fee: "0.5",
	}
	lp3, err := types.LiquidityProvisionSubmissionFromProto(lp3pb)
	require.NoError(t, err)

	idgen = idgeneration.New(crypto.RandomHash())
	_, err = tng.engine.SubmitLiquidityProvision(ctx, lp3, party3, idgen)
	require.NoError(t, err)

	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)

	err = tng.engine.CancelLiquidityProvision(ctx, party1)
	require.NoError(t, err)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)
}
