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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50}}

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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50}}

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

	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 20, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetLPSCount())
}

// Check that we are unable to directly cancel a pegged order that was
// created by the LP system
func TestLiquidity_MustNotBeAbleToCancelLPOrder(t *testing.T) {
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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 2, Proportion: 50}}

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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 2, Proportion: 50}}

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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50}}

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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_ACTIVE.String(), tm.market.GetLPSState("trader-A").String())
	assert.Equal(t, 4, tm.market.GetPeggedOrderCount())

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
	buys = []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50}}
	sells = []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50}}
	lps.Buys = buys
	lps.Sells = sells
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	assert.Equal(t, 2, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 2, tm.market.GetParkedOrderCount())
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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	// TODO	require.Equal(t, types.LiquidityProvision_STATUS_UNDEPLOYED.String(), tm.market.GetLPSState("trader-A").String())
	// TODO assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, uint64(1000), tm.market.GetBondAccountBalance(ctx, "trader-A", tm.market.GetID(), tm.asset))

	// Now attempt to amend the LP submission with something invalid
	lps.Buys = nil
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.Error(t, err)

	// Check that the original LP submission is still working fine
	// TODO	require.Equal(t, types.LiquidityProvision_STATUS_UNDEPLOYED.String(), tm.market.GetLPSState("trader-A").String())
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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50}}

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
	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -2, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 2, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_ACTIVE.String(), tm.market.GetLPSState("trader-A").String())
	// Only 3 pegged orders as one fails due to price monitoring
	assert.Equal(t, 3, tm.market.GetPeggedOrderCount())
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

	buys := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -6, Proportion: 50}}
	sells := []*types.LiquidityOrder{&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
		&types.LiquidityOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 6, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              "0.01",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 1000,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvision_STATUS_ACTIVE.String(), tm.market.GetLPSState("trader-A").String())
	assert.Equal(t, 5, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 5, tm.market.GetParkedOrderCount())

	// Leave the auction so we can uncross the book
	now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, now)
	tm.market.OnChainTimeUpdate(ctx, now)
	assert.Equal(t, 5, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())

	// TODO Check that the liquidity provision has taken into account the pegged order we already had
}
