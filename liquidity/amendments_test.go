package liquidity_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/liquidity"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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

	sub := getTestAmendSimpleSubmission()

	// initially submit our provision to be amended, does not matter what's in
	tng.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.NoError(t,
		tng.engine.SubmitLiquidityProvision(ctx, sub, party, "some-id-1"),
	)

	// now we can do a OK can amend
	assert.NoError(t, tng.engine.CanAmend(sub, party))

	sub = getTestAmendSimpleSubmission()
	sub.Fee = ""
	assert.EqualError(t,
		tng.engine.CanAmend(sub, party),
		"invalid liquidity provision fee",
	)

	sub = getTestAmendSimpleSubmission()
	sub.Buys = nil
	assert.EqualError(t,
		tng.engine.CanAmend(sub, party),
		"empty SIDE_BUY shape",
	)

	sub = getTestAmendSimpleSubmission()
	sub.Sells = nil
	assert.EqualError(t,
		tng.engine.CanAmend(sub, party),
		"empty SIDE_SELL shape",
	)
}

func getTestAmendSimpleSubmission() *types.LiquidityProvisionSubmission {
	return &types.LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: 10000,
		Fee:              "0.5",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 7, Offset: -10},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 3, Offset: -15},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 8, Offset: 10},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: 15},
		},
	}
}
