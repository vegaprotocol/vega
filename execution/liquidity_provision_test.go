package execution_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarket_RejectLPSubmissionIfFeeIncorrect(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50}}

	// Submitting a zero or smaller fee should cause a reject
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.00",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())

	// Submitting a zero or smaller fee should cause a reject
	lps = &types.LiquidityProvisionSubmission{
		Fee:              "-0.50",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder02")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())

	// Submitting a fee greater than 1.0 should cause a reject
	lps = &types.LiquidityProvisionSubmission{
		Fee:              "1.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder03")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())
}
