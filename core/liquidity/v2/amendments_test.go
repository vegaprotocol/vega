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
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/libs/crypto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
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
