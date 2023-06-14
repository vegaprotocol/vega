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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSubmissions(t *testing.T) {
	t.Run("Create and cancel", testSubmissionCreateAndCancel)
	t.Run("Cancel non existing", testCancelNonExistingSubmission)
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
	now := te.tsvc.GetTimeNow().UnixNano()

	expected := &types.LiquidityProvision{
		ID:               lpID,
		MarketID:         te.marketID,
		Party:            party,
		Fee:              num.DecimalFromFloat(0.5),
		CommitmentAmount: lps.CommitmentAmount.Clone(),
		CreatedAt:        now,
		UpdatedAt:        now,
		Status:           types.LiquidityProvisionStatusActive,
		Version:          1,
	}

	// Creating a submission should fire an event
	te.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

	idgen := idgeneration.New(deterministicID)
	err = te.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	require.NoError(t, err)

	got := te.engine.LiquidityProvisionByPartyID(party)
	require.Equal(t, expected, got)

	expected.Status = types.LiquidityProvisionStatusCancelled
	te.broker.EXPECT().Send(
		events.NewLiquidityProvisionEvent(ctx, expected),
	).Times(1)

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

	// Send a submission
	lp1pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "100", Fee: "0.5",
	}
	lp1, err := types.LiquidityProvisionSubmissionFromProto(lp1pb)
	require.NoError(t, err)

	idgen := idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp1, party1, idgen),
	)
	suppliedStake := tng.engine.CalculateSuppliedStake()
	require.Equal(t, lp1.CommitmentAmount, suppliedStake)

	lp2pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "500", Fee: "0.5",
	}
	lp2, err := types.LiquidityProvisionSubmissionFromProto(lp2pb)
	require.NoError(t, err)

	idgen = idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp2, party2, idgen),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount), suppliedStake)

	lp3pb := &commandspb.LiquidityProvisionSubmission{
		MarketId: tng.marketID, CommitmentAmount: "962", Fee: "0.5",
	}
	lp3, err := types.LiquidityProvisionSubmissionFromProto(lp3pb)
	require.NoError(t, err)

	idgen = idgeneration.New(crypto.RandomHash())
	require.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, lp3, party3, idgen),
	)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp1.CommitmentAmount, lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)

	err = tng.engine.CancelLiquidityProvision(ctx, party1)
	require.NoError(t, err)
	suppliedStake = tng.engine.CalculateSuppliedStake()
	require.Equal(t, num.Sum(lp2.CommitmentAmount, lp3.CommitmentAmount), suppliedStake)
}
