package execution_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiquidity_RejectLPSubmissionIfFeeIncorrect(t *testing.T) {
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

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50},
	}

	// Submitting a zero or smaller fee should cause a reject
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "-0.50",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder02")
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

func TestLiquidity_RejectLPSubmissionIfSideMissing(t *testing.T) {
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

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50},
	}

	// Submitting a shape with no buys should cause a reject
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())

	// Submitting a shape with no sells should cause a reject
	lps = &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder02")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())
}

func TestLiquidity_PreventCommitmentReduction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 10000000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Leave auction
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Try to reduce our commitment to below the minimum level
	lps = &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.Error(t, err)
	assert.Equal(t, 1, tm.market.GetLPSCount())
}

// We have a limit to the number of orders in each shape of a liquidity provision submission
// to prevent a user spaming the system. Place an LPSubmission order with too many
// orders in to make it reject it.
func TestLiquidity_TooManyShapeLevels(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create a buy side that has too many items
	buys := make([]*types.LiquidityOrder, 200)
	for i := 0; i < 200; i++ {
		buys[i] = &types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: int64(-10 - i), Proportion: 1}
	}

	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.EqualError(t, err, "SIDE_BUY shape size exceed max (100)")
	assert.Equal(t, 0, tm.market.GetLPSCount())
}

func TestLiquidityProvisionFeeValidation(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	// auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 500000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 70000,
		Fee:              "-0.1",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
		"invalid liquidity provision fee",
	)

	lpSubmission.Fee = "10"

	// submit our lp
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
		"invalid liquidity provision fee",
	)

	lpSubmission.Fee = "0"

	// submit our lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

}

// Check that we are unable to directly cancel or amend a pegged order that was
// created by the LP system
func TestLiquidity_MustNotBeAbleToCancelOrAmendLPOrder(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 10000000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2, Proportion: 50}}
	sells := []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 2, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)

	// Leave auction
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Check we have an accepted LP submission
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Check we have the right number of live orders
	assert.Equal(t, int64(6), tm.market.GetOrdersOnBookCount())

	// Attempt to cancel one of the pegged orders and it is rejected
	orders := tm.market.GetPeggedOrders("trader-A")
	assert.GreaterOrEqual(t, len(orders), 0)

	cancelConf, err := tm.market.CancelOrder(ctx, "trader-A", orders[0].Id)
	require.Nil(t, cancelConf)
	require.Error(t, err)
	assert.Equal(t, types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED, err)

	// Attempt to amend one of the pegged orders
	amend := &types.OrderAmendment{OrderId: orders[0].Id,
		PartyId:   orders[0].PartyId,
		MarketId:  orders[0].MarketId,
		SizeDelta: +5}
	amendConf, err := tm.market.AmendOrder(ctx, amend)
	require.Error(t, err)
	require.Nil(t, amendConf)
	assert.Equal(t, types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED, err)
}

// When a liquidity provider submits an order and runs out of margin from both their general
// and margin account, the system should take the required amount from the bond account
func TestLiquidity_CheckThatBondAccountUsedToFundShortfallInInitialMargin(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 3000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 2, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, uint64(1000), tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))

	// Leave auction
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Check we have an accepted LP submission
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Check we have the right number of live orders
	assert.Equal(t, int64(6), tm.market.GetOrdersOnBookCount())

	// Check that the bond balance has been reduced
	assert.Less(t, tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset), uint64(1000))
}

// When a liquidity provider has a position that requires more margin after a MTM settlement,
// they should use the assets in the bond account after the general and margin account are empty
func TestLiquidity_CheckThatBondAccountUsedToFundShortfallInMaintenanceMargin(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 5000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o31 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order031", types.Side_SIDE_SELL, "trader-C", 1, 30)
	o31conf, err := tm.market.SubmitOrder(ctx, o31)
	require.NotNil(t, o31conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, uint64(1000), tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))

	// Leave auction
	now = now.Add(time.Second * 40)
	tm.market.LeaveAuction(ctx, now)

	// Check we have an accepted LP submission
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Check we have the right number of live orders
	assert.Equal(t, int64(7), tm.market.GetOrdersOnBookCount())

	// Check that the bond balance is untouched
	assert.Equal(t, tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset), uint64(1000))

	// Now move the mark price to force MTM settlement
	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_BUY, "trader-B", 1, 20)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Check the bond account has been reduced to cover the price move
	assert.Less(t, tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset), uint64(1000))
}

func TestLiquidity_CheckThatChangingLPDuringAuctionWorks(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 7000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o31 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order031", types.Side_SIDE_SELL, "trader-C", 1, 30)
	o31conf, err := tm.market.SubmitOrder(ctx, o31)
	require.NotNil(t, o31conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_PENDING.String(), tm.market.GetLPSState("trader-A").String())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())

	// Check we have the right amount of bond balance
	assert.Equal(t, uint64(1000), tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))

	// Amend the commitment
	lps.CommitmentAmount = 2000
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, uint64(2000), tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))

	// Amend the commitment
	lps.CommitmentAmount = 500
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, uint64(500), tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))

	// Change the shape of the lp submission
	buys = []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50}}
	sells = []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50}}
	lps.Buys = buys
	lps.Sells = sells
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

// If we submit a valid LP submission but then try ot alter it to something non valid
// the amendment should be rejected and the original submission is still valid
func TestLiquidity_CheckThatFailedAmendDoesNotBreakExistingLP(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 7000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_PENDING.String(), tm.market.GetLPSState("trader-A").String())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, uint64(1000), tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))

	// Now attempt to amend the LP submission with something invalid
	lps.Buys = nil
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	// We will not get an error because the previous submission will be re-used
	require.NoError(t, err)

	// Check that the original LP submission is still working fine
	require.Equal(t, types.LiquidityProvision_STATUS_PENDING.String(), tm.market.GetLPSState("trader-A").String())
}

// Liquidity fee must be updated when new LP submissions are added or existing ones
// removed
func TestLiquidity_CheckFeeIsCorrectAfterChanges(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 7000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// We shouldn't have a liquidity fee yet
	// TODO	assert.Equal(t, 0.0, tm.market.GetLiquidityFee())

	buys := []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50}}
	sells := []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)

	// Check the fee is correct
	// TODO	assert.Equal(t, 0.01, tm.market.GetLiquidityFee())

	// Update the fee
	lps.Fee = "0.5"
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)

	// Check the fee is correct
	// TODO	assert.Equal(t, 0.5, tm.market.GetLiquidityFee())
}

func TestLiquidity_CheckWeCanSubmitLPDuringPriceAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)

	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.15, AuctionExtension: 60},
			},
		},
		UpdateFrequency: 600,
	}

	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 700000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 1000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 1000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 2000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_SELL, "trader-C", 10, 3000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Leave the auction so we can uncross the book
	now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, now)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Move the price enough that we go into a price auction
	now = now.Add(time.Second * 20)
	o5 := getMarketOrder(tm, now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_IOC, "Order05", types.Side_SIDE_BUY, "trader-B", 2, 0)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Check we are in price auction
	assert.Equal(t, types.AuctionTrigger_AUCTION_TRIGGER_PRICE, tm.market.GetMarketData().Trigger)

	// Now try to submit a LP submission
	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -2, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 2, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_PENDING.String(), tm.market.GetLPSState("trader-A").String())
	// Only 3 pegged orders as one fails due to price monitoring
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
}

func TestLiquidity_CheckThatExistingPeggedOrdersCountTowardsCommitment(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 7000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Add a manual pegged order which should be included in commitment calculations
	pegged := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Peggy", types.Side_SIDE_BUY, "trader-A", 1, 0)
	pegged.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2}
	peggedconf, err := tm.market.SubmitOrder(ctx, pegged)
	require.NotNil(t, peggedconf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50}}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_PENDING.String(), tm.market.GetLPSState("trader-A").String())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())

	// Leave the auction so we can uncross the book
	now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, now)
	tm.market.OnChainTimeUpdate(ctx, now)
	assert.Equal(t, 5, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())

	// TODO Check that the liquidity provision has taken into account the pegged order we already had
}

// When a price monitoring auction is started, make sure we cancel all the pegged orders and
// that no fees are charged to the liquidity providers
func TestLiquidity_CheckNoPenalityWhenGoingIntoPriceAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)

	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.95, AuctionExtension: 60},
			},
		},
		UpdateFrequency: 600,
	}

	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)
	ctx := context.Background()

	// Create a new trader account with very little funding
	addAccountWithAmount(tm, "trader-A", 700000)
	addAccountWithAmount(tm, "trader-B", 10000000)
	addAccountWithAmount(tm, "trader-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-B", 10, 1000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 1000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 2000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_SELL, "trader-C", 10, 3000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Submit a LP submission
	buys := []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50}}
	sells := []*types.LiquidityOrder{{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50}}

	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_PENDING.String(), tm.market.GetLPSState("trader-A").String())

	// Leave the auction so we can uncross the book
	now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, now)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Save the total amount of assets we have in general+margin+bond
	totalFunds := tm.market.GetTotalAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset)

	// Move the price enough that we go into a price auction
	now = now.Add(time.Second * 20)
	o5 := getMarketOrder(tm, now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_IOC, "Order05", types.Side_SIDE_BUY, "trader-B", 2, 0)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Check we are in price auction
	assert.Equal(t, types.AuctionTrigger_AUCTION_TRIGGER_PRICE, tm.market.GetMarketData().Trigger)

	// All pegged orders must be removed
	// TODO assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	// TODO assert.Equal(t, 0, tm.market.GetParkedOrderCount())

	// Check we have not lost any assets
	assert.Equal(t, totalFunds, tm.market.GetTotalAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))
}

func TestLpCanResubmitAfterBeingClosedOut(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	var (
		lpparty = "lp-party-1"
		party0  = "party-0"
		party1  = "party-1"
	)
	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 300000).
		WithAccountAndAmount(party1, 1000000000).
		WithAccountAndAmount(party0, 1000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(.3)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 150000,
		Fee:              "0.01",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	tm.events = nil
	tm.EndOpeningAuction(t, auctionEnd, false)

	// make sure LP order is deployed
	t.Run("expect commitment statuses", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]types.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-1": types.LiquidityProvision_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

	// now set the markprice
	mpOrders := []*types.Order{
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        100,
			Remaining:   100,
			Price:       2000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     party1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        100,
			Remaining:   100,
			Price:       2000,
			Side:        types.Side_SIDE_BUY,
			PartyId:     party0,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
	}

	// submit the auctions orders
	tm.WithSubmittedOrders(t, mpOrders...)

	// make sure LP order is cancelled
	t.Run("expect commitment statuses", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]types.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-1": types.LiquidityProvision_STATUS_CANCELLED,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

	// now redeposit-funds
	// log more
	tm.WithAccountAndAmount(lpparty, 25000000)

	tm.events = nil
	// an re-submit the lp
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-2"),
	)

	// make sure LP order is deployed
	t.Run("new LP order is active", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]types.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-2": types.LiquidityProvision_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

}

func TestCloseOutLPTraderContIssue3086(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	var (
		ruser1 = "ruser_1"
		ruser2 = "ruser_2"
		ruser3 = "ruser3"
	)
	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(ruser1, 500000).
		WithAccountAndAmount(ruser2, 8600).
		WithAccountAndAmount(ruser3, 10000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(.2)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 2000,
		Fee:              "0.01",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 2, Offset: 5},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, ruser2, "liquidity-submission-1"),
	)

	tm.events = nil
	tm.EndOpeningAuction(t, auctionEnd, false)

	// make sure LP order is deployed
	t.Run("new LP order is active", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]types.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-1": types.LiquidityProvision_STATUS_ACTIVE,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

	// now add a couple of orders
	// now set the markprice
	mpOrders := []*types.Order{
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        3,
			Remaining:   3,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     ruser3,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        2,
			Remaining:   2,
			Price:       4500,
			Side:        types.Side_SIDE_SELL,
			PartyId:     ruser2,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
		{
			Type:        types.Order_TYPE_LIMIT,
			Size:        4,
			Remaining:   4,
			Price:       4500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     ruser1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		},
	}

	tm.events = nil
	// submit the auctions orders
	tm.WithSubmittedOrders(t, mpOrders...)

	// check accounts
	t.Run("margin account is updated", func(t *testing.T) {
		_, err := tm.collateralEngine.GetPartyMarginAccount(
			tm.market.GetID(), ruser2, tm.asset)
		assert.EqualError(t, err, collateral.ErrAccountDoesNotExist.Error())
	})

	t.Run("bond account", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(
			tm.market.GetID(), ruser2, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, 0, int(acc.Balance))
	})

	t.Run("general account", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyGeneralAccount(
			ruser2, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, 0, int(acc.Balance))
	})

	// deposit funds again for this party
	tm.WithAccountAndAmount(ruser2, 90000000)

	t.Run("general account", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyGeneralAccount(
			ruser2, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, 90000000, int(acc.Balance))
	})

	lpSubmission2 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200000,
		Fee:              "0.01",
		Reference:        "ref-lp-submission-2",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -10},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 20},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: 10},
		},
	}

	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission2, ruser2, "liquidity-submission-2"),
	)

	// make sure LP order is deployed
	t.Run("new LP order is active", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]types.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvision_Status{
			"liquidity-submission-2": types.LiquidityProvision_STATUS_PENDING,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})

}

func TestLiquidityFeeIsSelectedProperly(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	lpparty := "lp-party-1"
	lpparty2 := "lp-party-2"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount(lpparty, 500000000000).
		WithAccountAndAmount(lpparty2, 500000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 70000,
		Fee:              "0.5",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("current liquidity fee is 0.5", func(t *testing.T) {
		// First collect all the orders events
		found := types.Market{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.MarketUpdated:
				found = evt.Market()
			}
		}

		assert.Equal(t, found.Fees.Factors.LiquidityFee, "0.5")
	})

	tm.EndOpeningAuction(t, auctionEnd, false)

	// now we submit a second LP, with a lower fee,
	// but we still need the first LP to cover liquidity
	// so its fee is selected
	lpSubmission2 := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 20000,
		Fee:              "0.1",
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission2, lpparty2, "liquidity-submission-2"),
	)

	t.Run("current liquidity fee is still 0.5", func(t *testing.T) {
		// First collect all the orders events
		var found *types.Market
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.MarketUpdated:
				mkt := evt.Market()
				found = &mkt
			}
		}

		// no update to the liquidity fee
		assert.Nil(t, found)
	})

	// now submit again the commitment, but we coverall othe target stake
	// so our fee should be selected
	tm.events = nil
	lpSubmission2.CommitmentAmount = 60000

	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission2, lpparty2, "liquidity-submission-2"),
	)

	t.Run("current liquidity fee is still 0.5", func(t *testing.T) {
		// First collect all the orders events
		found := types.Market{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.MarketUpdated:
				found = evt.Market()
			}
		}
		// no update to the liquidity fee
		assert.Equal(t, found.Fees.Factors.LiquidityFee, "0.1")
	})

}
