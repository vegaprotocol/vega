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
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

const (
	market = "ETH/USD"
)

func TestAmendments(t *testing.T) {
	t.Run("test can amend", testCanAmend)
}

func testCanAmend(t *testing.T) {
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

	lps := getTestSubmitSimpleSubmission()

	idgen := idgeneration.New(crypto.RandomHash())
	// initially submit our provision to be amended, does not matter what's in
	tng.broker.EXPECT().Send(gomock.Any()).Times(1)
	tng.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	err := tng.engine.SubmitLiquidityProvision(ctx, lps, party, idgen)
	assert.NoError(t, err)
	lp := tng.engine.LiquidityProvisionByPartyID(party)
	require.NotNil(t, lp)
	require.EqualValues(t, 1, lp.Version)

	lpa := getTestAmendSimpleSubmission()
	// now we can do a OK can amend
	assert.NoError(t, tng.engine.CanAmend(lpa, party))

	// previously, this tested for an empty string, this is impossible now with the decimal type
	// so let's check for negatives instead
	lpa.Fee = num.DecimalFromFloat(-1)
	assert.EqualError(t,
		tng.engine.CanAmend(lpa, party),
		"invalid liquidity provision fee",
	)

	lpa = getTestAmendSimpleSubmission()
	lpa.Buys = nil
	assert.NoError(t, tng.engine.CanAmend(lpa, party))

	lpa = getTestAmendSimpleSubmission()
	lpa.Sells = nil
	assert.NoError(t, tng.engine.CanAmend(lpa, party))
}

func getTestSubmitSimpleSubmission() *types.LiquidityProvisionSubmission {
	pb := &commandspb.LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: "10000",
		Fee:              "0.5",
		Reference:        "ref-lp-submission-1",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 7, Offset: "10"},
			{Reference: types.PeggedReferenceMid, Proportion: 3, Offset: "15"},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 8, Offset: "10"},
			{Reference: types.PeggedReferenceMid, Proportion: 2, Offset: "15"},
		},
	}
	t, _ := types.LiquidityProvisionSubmissionFromProto(pb)
	return t
}

func getTestAmendSimpleSubmission() *types.LiquidityProvisionAmendment {
	pb := &commandspb.LiquidityProvisionAmendment{
		MarketId:         market,
		CommitmentAmount: "10000",
		Fee:              "0.5",
		Reference:        "ref-lp-submission-1",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 7, Offset: "10"},
			{Reference: types.PeggedReferenceMid, Proportion: 3, Offset: "15"},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 8, Offset: "10"},
			{Reference: types.PeggedReferenceMid, Proportion: 2, Offset: "15"},
		},
	}
	t, _ := types.LiquidityProvisionAmendmentFromProto(pb)
	return t
}
