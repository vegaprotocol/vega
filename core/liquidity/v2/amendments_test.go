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
	"code.vegaprotocol.io/vega/libs/crypto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/liquidity"
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
		now   = time.Now()
		tng   = newTestEngine(t, now)
	)
	defer tng.ctrl.Finish()

	assert.EqualError(t,
		tng.engine.CanAmend(nil, party),
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
	tng.broker.EXPECT().Send(gomock.Any()).Times(2)
	err := tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	assert.NoError(t, err)
	lp := tng.engine.LiquidityProvisionByPartyID(party)
	require.NotNil(t, lp)
	require.EqualValues(t, 1, lp.Version)

	lpa, _ := types.LiquidityProvisionAmendmentFromProto(&commandspb.LiquidityProvisionAmendment{
		MarketId:         market,
		CommitmentAmount: "100000",
		Fee:              "0.8",
		Reference:        "ref-lp-submission-1",
	})
	// now we can do a OK can amend
	assert.NoError(t, tng.engine.CanAmend(lpa, party))

	assert.NoError(t, tng.engine.AmendLiquidityProvision(ctx, lpa, party))

	lp = tng.engine.LiquidityProvisionByPartyID(party)
	assert.Equal(t, lpa.CommitmentAmount.String(), lp.CommitmentAmount.String())
	assert.Equal(t, lpa.Fee.String(), lp.Fee.String())
	assert.EqualValues(t, 2, lp.Version)

	// previously, this tested for an empty string, this is impossible now with the decimal type
	// so let's check for negatives instead
	lpa.Fee = num.DecimalFromFloat(-1)
	assert.EqualError(t,
		tng.engine.CanAmend(lpa, party),
		"invalid liquidity provision fee",
	)
}
