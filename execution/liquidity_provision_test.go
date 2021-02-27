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
