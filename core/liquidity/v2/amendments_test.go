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

	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	market = "ETH/USD"
)

func TestAmendments(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		tng   = newTestEngine(t)
	)
	defer tng.ctrl.Finish()

	assert.EqualError(t,
		tng.engine.CanAmend(nil, party, true),
		liquidity.ErrPartyHaveNoLiquidityProvision.Error(),
	)

	lps, _ := types.LiquidityProvisionSubmissionFromProto(&commandspb.LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: "10000",
		Fee:              "0.5",
		Reference:        "ref-lp-submission-1",
	})

	now := time.Now()
	zero := num.UintZero()
	zeroD := num.DecimalZero()

	idgen := idgeneration.New(crypto.RandomHash())
	// initially submit our provision to be amended, does not matter what's in
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tng.auctionState.EXPECT().InAuction().Return(false).AnyTimes()
	tng.auctionState.EXPECT().IsOpeningAuction().Return(false).AnyTimes()

	_, err := tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	assert.NoError(t, err)
	tng.engine.ResetSLAEpoch(now, zero, zero, zeroD)
	tng.engine.ApplyPendingProvisions(ctx, now)
	originalLp := tng.engine.LiquidityProvisionByPartyID(party)

	require.NotNil(t, originalLp)
	require.EqualValues(t, 1, originalLp.Version)

	lpa, _ := types.LiquidityProvisionAmendmentFromProto(&commandspb.LiquidityProvisionAmendment{
		MarketId:         market,
		CommitmentAmount: "100000",
		Fee:              "0.8",
		Reference:        "ref-lp-submission-1",
	})
	// now we can do a OK can amend
	assert.NoError(t, tng.engine.CanAmend(lpa, party, true))

	_, err = tng.engine.AmendLiquidityProvision(ctx, lpa, party, true)
	assert.NoError(t, err)

	// first validate that the amendment is pending
	pendingLp := tng.engine.PendingProvisionByPartyID(party)

	assert.Equal(t, lpa.CommitmentAmount.String(), pendingLp.CommitmentAmount.String())
	assert.Equal(t, lpa.Fee.String(), pendingLp.Fee.String())

	lp := tng.engine.LiquidityProvisionByPartyID(party)
	assert.Equal(t, originalLp.CommitmentAmount.String(), lp.CommitmentAmount.String())
	assert.Equal(t, originalLp.Fee.String(), lp.Fee.String())
	assert.Equal(t, originalLp.Version, lp.Version)

	// amendment should take place at the start of new epoch
	tng.engine.ResetSLAEpoch(now, zero, zero, zeroD)
	tng.engine.ApplyPendingProvisions(ctx, now)

	lp = tng.engine.LiquidityProvisionByPartyID(party)
	assert.Equal(t, lpa.CommitmentAmount.String(), lp.CommitmentAmount.String())
	assert.Equal(t, lpa.Fee.String(), lp.Fee.String())
	assert.EqualValues(t, 2, lp.Version)

	// previously, this tested for an empty string, this is impossible now with the decimal type
	// so let's check for negatives instead
	lpa.Fee = num.DecimalFromFloat(-1)
	assert.EqualError(t,
		tng.engine.CanAmend(lpa, party, true),
		"invalid liquidity provision fee",
	)
}

func TestCancelTroughAmendmentDuringOpeningAuction(t *testing.T) {
	var (
		party = "party-1"
		ctx   = context.Background()
		tng   = newTestEngine(t)
	)
	defer tng.ctrl.Finish()

	assert.EqualError(t,
		tng.engine.CanAmend(nil, party, true),
		liquidity.ErrPartyHaveNoLiquidityProvision.Error(),
	)

	lps, _ := types.LiquidityProvisionSubmissionFromProto(&commandspb.LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: "10000",
		Fee:              "0.5",
		Reference:        "ref-lp-submission-1",
	})

	idgen := idgeneration.New(crypto.RandomHash())
	// initially submit our provision to be amended, does not matter what's in
	tng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	// set opening auction
	tng.auctionState.EXPECT().InAuction().Return(true).AnyTimes()
	tng.auctionState.EXPECT().IsOpeningAuction().Return(true).AnyTimes()

	applied, err := tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	assert.NoError(t, err)
	assert.True(t, applied)

	// amend to zero - cancel
	lpa, _ := types.LiquidityProvisionAmendmentFromProto(&commandspb.LiquidityProvisionAmendment{
		MarketId:         market,
		CommitmentAmount: "0",
		Fee:              "0",
	})

	applied, err = tng.engine.AmendLiquidityProvision(ctx, lpa, party, true)
	assert.NoError(t, err)
	assert.True(t, applied)

	// should not be in pending
	pendingLp := tng.engine.PendingProvisionByPartyID(party)
	assert.Nil(t, pendingLp)

	// should not be in current
	currentLp := tng.engine.LiquidityProvisionByPartyID(party)
	assert.Nil(t, currentLp)

	// LP is able to submit again - since they cancelled before
	applied, err = tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	assert.NoError(t, err)
	assert.True(t, applied)

	lp := tng.engine.LiquidityProvisionByPartyID(party)
	assert.Equal(t, lps.CommitmentAmount.String(), lp.CommitmentAmount.String())
	assert.Equal(t, lps.Fee.String(), lp.Fee.String())
	assert.EqualValues(t, 1, lp.Version)
}
