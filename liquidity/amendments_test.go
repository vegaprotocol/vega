package liquidity_test

import (
	"context"
	"testing"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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

	lpa := getTestAmendSimpleSubmission()

	// initially submit our provision to be amended, does not matter what's in
	tng.broker.EXPECT().Send(gomock.Any()).Times(1)
	_, err := tng.engine.AmendLiquidityProvision(ctx, lpa, party)
	assert.NoError(t, err)

	// now we can do a OK can amend
	assert.NoError(t, tng.engine.CanAmend(lpa, party))

	lpa = getTestAmendSimpleSubmission()
	// previously, this tested for an empty string, this is impossible now with the decimal type
	// so let's check for negatives instead
	lpa.Fee = num.DecimalFromFloat(-1)
	assert.EqualError(t,
		tng.engine.CanAmend(lpa, party),
		"invalid liquidity provision fee",
	)

	lpa = getTestAmendSimpleSubmission()
	lpa.Buys = nil
	assert.EqualError(t,
		tng.engine.CanAmend(lpa, party),
		"empty SIDE_BUY shape",
	)

	lpa = getTestAmendSimpleSubmission()
	lpa.Sells = nil
	assert.EqualError(t,
		tng.engine.CanAmend(lpa, party),
		"empty SIDE_SELL shape",
	)
}

func getTestAmendSimpleSubmission() *types.LiquidityProvisionAmendment {
	pb := &commandspb.LiquidityProvisionAmendment{
		MarketId:         market,
		CommitmentAmount: "10000",
		Fee:              "0.5",
		Reference:        "ref-lp-submission-1",
		Buys: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 7, Offset: -10},
			{Reference: types.PeggedReferenceMid, Proportion: 3, Offset: -15},
		},
		Sells: []*proto.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 8, Offset: 10},
			{Reference: types.PeggedReferenceMid, Proportion: 2, Offset: 15},
		},
	}
	t, _ := types.LiquidityProvisionAmendmentFromProto(pb)
	return t
}
