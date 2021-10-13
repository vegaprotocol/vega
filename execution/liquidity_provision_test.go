package execution_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiquidity_RejectLPSubmissionIfFeeIncorrect(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReferenceBestBid, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReferenceBestAsk, Offset: 20, Proportion: 50},
	}

	// Submitting a zero or smaller fee should cause a reject
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(-0.50),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder02")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())

	// Submitting a fee greater than 1.0 should cause a reject
	lps = &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(1.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder03")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())
}

func TestLiquidity_RejectLPSubmissionIfSideMissing(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReferenceBestBid, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReferenceBestAsk, Offset: 20, Proportion: 50},
	}

	// Submitting a shape with no buys should cause a reject
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Sells:            sells,
	}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())

	// Submitting a shape with no sells should cause a reject
	lps = &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder02")
	require.Error(t, err)
	assert.Equal(t, 0, tm.market.GetLPSCount())
}

func TestLiquidity_PreventCommitmentReduction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 10000000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-C", 1, 9)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Leave auction
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))
	// mark price is set at 10, orders on book

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -10, Proportion: 50},
		{Reference: types.PeggedReferenceBestBid, Offset: -20, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReferenceBestAsk, Offset: 20, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Try to reduce our commitment to below the minimum level
	lps = &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
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

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Start the opening auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create a buy side that has too many items
	buys := make([]*types.LiquidityOrder, 200)
	for i := 0; i < 200; i++ {
		buys[i] = &types.LiquidityOrder{Reference: types.PeggedReferenceBestBid, Offset: int64(-10 - i), Proportion: 1}
	}

	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 10, Proportion: 50},
		{Reference: types.PeggedReferenceBestAsk, Offset: 20, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
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
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
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
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(70000),
		Fee:              num.DecimalFromFloat(-0.1),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReferenceMid, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
		},
	}

	// submit our lp
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
		"invalid liquidity provision fee",
	)

	lpSubmission.Fee = num.DecimalFromFloat(10)

	// submit our lp
	require.EqualError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
		"invalid liquidity provision fee",
	)

	lpSubmission.Fee = num.DecimalZero()

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

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 10000000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReferenceBestBid, Offset: -2, Proportion: 50}}
	sells := []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReferenceBestAsk, Offset: 2, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)

	// Leave auction
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Check we have an accepted LP submission
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Check we have the right number of live orders
	assert.Equal(t, int64(6), tm.market.GetOrdersOnBookCount())

	// FIXME(): REDO THIS TEST
	// Attempt to cancel one of the pegged orders and it is rejected
	// orders := tm.market.GetPeggedOrders("party-A")
	// assert.GreaterOrEqual(t, len(orders), 0)

	// cancelConf, err := tm.market.CancelOrder(ctx, "party-A", orders[0].Id)
	// require.Nil(t, cancelConf)
	// require.Error(t, err)
	// assert.Equal(t, types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED, err)

	// // Attempt to amend one of the pegged orders
	// amend := &commandspb.OrderAmendment{OrderId: orders[0].Id,
	// 	MarketId:  orders[0].MarketId,
	// 	SizeDelta: +5}
	// amendConf, err := tm.market.AmendOrder(ctx, amend, orders[0].PartyId)
	// require.Error(t, err)
	// require.Nil(t, amendConf)
	// assert.Equal(t, types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED, err)
}

// When a liquidity provider submits an order and runs out of margin from both their general
// and margin account, the system should take the required amount from the bond account
func TestLiquidity_CheckThatBondAccountUsedToFundShortfallInInitialMargin(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 5000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReferenceBestBid, Offset: -2, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReferenceBestAsk, Offset: 2, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

	// Leave auction
	tm.market.LeaveAuction(ctx, now.Add(time.Second*20))

	// Check we have an accepted LP submission
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Check we have the right number of live orders
	assert.Equal(t, int64(6), tm.market.GetOrdersOnBookCount())

	// Check that the bond balance has been reduced
	assert.True(t, tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset).LT(num.NewUint(1000)))
}

// When a liquidity provider has a position that requires more margin after a MTM settlement,
// they should use the assets in the bond account after the general and margin account are empty
func TestLiquidity_CheckThatBondAccountUsedToFundShortfallInMaintenanceMargin(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 7000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o31 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order031", types.SideSell, "party-C", 1, 30)
	o31conf, err := tm.market.SubmitOrder(ctx, o31)
	require.NotNil(t, o31conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: -6, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: 6, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

	// Leave auction
	now = now.Add(time.Second * 40)
	tm.market.OnChainTimeUpdate(ctx, now)
	tm.market.LeaveAuction(ctx, now)

	// Check we have an accepted LP submission
	assert.Equal(t, 1, tm.market.GetLPSCount())

	// Check we have the right number of live orders
	assert.Equal(t, int64(7), tm.market.GetOrdersOnBookCount())

	// Check that the bond balance is untouched
	assert.True(t, tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset).EQ(num.NewUint(1000)))

	tm.events = nil
	// Now move the mark price to force MTM settlement
	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 1, 20)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	t.Run("expect bond slashing transfer", func(t *testing.T) {
		// First collect all the orders events
		found := []*proto.TransferResponse{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.TransferResponse:
				for _, v := range evt.TransferResponses() {
					for _, t := range v.Transfers {
						if t.Reference == "TRANSFER_TYPE_BOND_SLASHING" {
							found = append(found, v)
						}
					}
				}
			}
		}

		assert.Len(t, found, 1)
	})
}

func TestLiquidity_CheckThatChangingLPDuringAuctionWorks(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 7000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.2)

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o31 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order031", types.SideSell, "party-C", 1, 30)
	o31conf, err := tm.market.SubmitOrder(ctx, o31)
	require.NotNil(t, o31conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: -6, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: 6, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())

	// Check we have the right amount of bond balance
	assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

	// Amend the commitment
	lps.CommitmentAmount = num.NewUint(2000)
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, num.NewUint(2000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

	// Amend the commitment
	lps.CommitmentAmount = num.NewUint(500)
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)

	// Check we have the right amount of bond balance
	assert.Equal(t, num.NewUint(500), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

	// Change the shape of the lp submission
	buys = []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50}}
	sells = []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50}}
	lps.Buys = buys
	lps.Sells = sells
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
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

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 7000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: -6, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: 6, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, num.NewUint(1000), tm.market.GetBondAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))

	// Now attempt to amend the LP submission with something invalid
	lps.Buys = nil
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.EqualError(t, err, "empty SIDE_BUY shape")

	// Check that the original LP submission is still working fine
	require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
}

// Liquidity fee must be updated when new LP submissions are added or existing ones
// removed
func TestLiquidity_CheckFeeIsCorrectAfterChanges(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 7000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// We shouldn't have a liquidity fee yet
	// TODO	assert.Equal(t, 0.0, tm.market.GetLiquidityFee())

	buys := []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50}}
	sells := []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err := tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)

	// Check the fee is correct
	// TODO	assert.Equal(t, 0.01, tm.market.GetLiquidityFee())

	// Update the fee
	lps.Fee = num.DecimalFromFloat(0.5)
	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)

	// Check the fee is correct
	// TODO	assert.Equal(t, 0.5, tm.market.GetLiquidityFee())
}

func TestLiquidity_CheckWeCanSubmitLPDuringPriceAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)

	hdec := num.DecimalFromFloat(60)
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{
					Horizon:          60,
					HDec:             hdec,
					Probability:      num.DecimalFromFloat(0.15),
					AuctionExtension: 60,
				},
			},
		},
		UpdateFrequency: 600,
	}

	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 10*time.Second)

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 70000000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 1000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 1000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 2000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "party-C", 10, 3000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	assert.Equal(t, types.AuctionTriggerOpening, tm.market.GetMarketData().Trigger)
	// Leave the auction so we can uncross the book
	now = now.Add(time.Second * 11)
	tm.market.OnChainTimeUpdate(ctx, now)
	// ensure we left auction
	assert.Equal(t, types.AuctionTriggerUnspecified, tm.market.GetMarketData().Trigger)

	// Move the price enough that we go into a price auction
	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-B", 3, 3000)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Check we are in price auction
	assert.Equal(t, types.AuctionTriggerPrice, tm.market.GetMarketData().Trigger)

	// Now try to submit a LP submission
	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: -2, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: 2, Proportion: 50},
	}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
	// Only 3 pegged orders as one fails due to price monitoring
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
}

func TestLiquidity_CheckThatExistingPeggedOrdersCountTowardsCommitment(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 7000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 20)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Add a manual pegged order which should be included in commitment calculations
	pegged := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Peggy", types.SideBuy, "party-A", 1, 0)
	pegged.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: -2}
	peggedconf, err := tm.market.SubmitOrder(ctx, pegged)
	require.NotNil(t, peggedconf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: -6, Proportion: 50}}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50},
		{Reference: types.PeggedReferenceMid, Offset: 6, Proportion: 50}}

	// Submitting a correct entry
	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())

	// Leave the auction so we can uncross the book
	now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, now)
	tm.market.OnChainTimeUpdate(ctx, now)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
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
				{
					Horizon:          60,
					HDec:             num.DecimalFromFloat(60),
					Probability:      num.DecimalFromFloat(0.95),
					AuctionExtension: 60,
				},
			},
		},
		UpdateFrequency: 600,
	}

	tm := getTestMarket(t, now, closingAt, pMonitorSettings, nil)
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second*10)

	// Create a new party account with very little funding
	addAccountWithAmount(tm, "party-A", 700000)
	addAccountWithAmount(tm, "party-B", 10000000)
	addAccountWithAmount(tm, "party-C", 10000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Create some normal orders to set the reference prices
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-B", 10, 1000)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 1000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 2000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "party-C", 10, 3000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Submit a LP submission
	buys := []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestBid, Offset: -1, Proportion: 50}}
	sells := []*types.LiquidityOrder{{Reference: types.PeggedReferenceBestAsk, Offset: 1, Proportion: 50}}

	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000),
		Buys:             buys,
		Sells:            sells,
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "party-A", "LPOrder01")
	require.NoError(t, err)
	require.Equal(t, types.LiquidityProvisionStatusPending.String(), tm.market.GetLPSState("party-A").String())

	// Leave the auction so we can uncross the book
	now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, now)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Save the total amount of assets we have in general+margin+bond
	totalFunds := tm.market.GetTotalAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset)

	// Move the price enough that we go into a price auction
	now = now.Add(time.Second * 20)
	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-B", 3, 3000)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Check we are in price auction
	assert.Equal(t, types.AuctionTriggerPrice, tm.market.GetMarketData().Trigger)

	// All pegged orders must be removed
	// TODO assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	// TODO assert.Equal(t, 0, tm.market.GetParkedOrderCount())

	// Check we have not lost any assets
	assert.Equal(t, totalFunds, tm.market.GetTotalAccountBalance(ctx, "party-A", tm.market.GetID(), tm.asset))
}

func TestLpCannotGetClosedOutWhenDeployingOrderForTheFirstTime(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
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
		WithAccountAndAmount(lpparty, 200000).
		WithAccountAndAmount(party1, 1000000000).
		WithAccountAndAmount(party0, 1000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(.3)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(150000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReferenceMid, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
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
		found := map[string]*proto.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvisionStatus{
			"liquidity-submission-1": types.LiquidityProvisionStatusCancelled,
		}

		require.Len(t, found, len(expectedStatus))

		for k, v := range expectedStatus {
			assert.Equal(t, v.String(), found[k].Status.String())
		}
	})
}

func TestCloseOutLPPartyContIssue3086(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.001),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
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
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(2000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 2, Offset: 5},
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
		found := map[string]*proto.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvisionStatus{
			"liquidity-submission-1": types.LiquidityProvisionStatusActive,
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
			Type:        types.OrderTypeLimit,
			Size:        3,
			Remaining:   3,
			Price:       num.NewUint(5000),
			Side:        types.SideSell,
			Party:       ruser3,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        2,
			Remaining:   2,
			Price:       num.NewUint(4500),
			Side:        types.SideSell,
			Party:       ruser2,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        4,
			Remaining:   4,
			Price:       num.NewUint(4500),
			Side:        types.SideBuy,
			Party:       ruser1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}

	tm.events = nil
	// submit the auctions orders
	tm.WithSubmittedOrders(t, mpOrders...)

	// check accounts
	t.Run("margin account is updated", func(t *testing.T) {
		_, err := tm.collateralEngine.GetPartyMarginAccount(
			tm.market.GetID(), ruser2, tm.asset)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account does not exist:")
	})

	t.Run("bond account", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyBondAccount(
			tm.market.GetID(), ruser2, tm.asset)
		assert.NoError(t, err)
		assert.True(t, acc.Balance.IsZero())
	})

	t.Run("general account", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyGeneralAccount(
			ruser2, tm.asset)
		assert.NoError(t, err)
		assert.True(t, acc.Balance.IsZero())
	})

	// deposit funds again for this party
	tm.WithAccountAndAmount(ruser2, 90000000)

	t.Run("general account", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyGeneralAccount(
			ruser2, tm.asset)
		assert.NoError(t, err)
		assert.Equal(t, num.NewUint(90000000), acc.Balance)
	})

	lpSubmission2 := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "ref-lp-submission-2",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 10, Offset: -10},
			{Reference: types.PeggedReferenceMid, Proportion: 13, Offset: -15},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 10, Offset: 20},
			{Reference: types.PeggedReferenceMid, Proportion: 13, Offset: 10},
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
		found := map[string]*proto.LiquidityProvision{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				lp := evt.LiquidityProvision()
				found[lp.Id] = lp
			}
		}

		expectedStatus := map[string]types.LiquidityProvisionStatus{
			"liquidity-submission-2": types.LiquidityProvisionStatusPending,
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
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(20),
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
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(70000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReferenceMid, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
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
		found := proto.Market{}
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
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(20000),
		Fee:              num.DecimalFromFloat(0.1),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 2, Offset: -5},
			{Reference: types.PeggedReferenceMid, Proportion: 2, Offset: -5},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 13, Offset: 5},
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
		var found *proto.Market
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
	lpSubmission2.CommitmentAmount = num.NewUint(60000)

	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission2, lpparty2, "liquidity-submission-2"),
	)

	t.Run("current liquidity fee is again 0.1", func(t *testing.T) {
		// First collect all the orders events
		found := proto.Market{}
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

func TestLiquidityOrderGeneratedSizes(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(2),
			},
		},
	}

	lpparty := "lp-party-1"
	oth1 := "party1"
	oth2 := "party2"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount(lpparty, 100000000000000).
		WithAccountAndAmount(oth1, 500000000000).
		WithAccountAndAmount(oth2, 500000000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.7)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 99, Offset: -201},
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: -200},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 1, Offset: 100},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 2, Offset: 101},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 98, Offset: 102},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("lp submission is pending", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
	})

	// then submit some orders, some for the lp party,
	// end some for the other parrties

	var lpOrders = []*types.Order{
		// Limit Orders
		{
			Type:        types.OrderTypeLimit,
			Size:        10,
			Remaining:   10,
			Price:       num.NewUint(120000),
			Side:        types.SideBuy,
			Party:       lpparty,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        10,
			Remaining:   10,
			Price:       num.NewUint(123000),
			Side:        types.SideSell,
			Party:       lpparty,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}

	// submit the auctions orders
	tm.WithSubmittedOrders(t, lpOrders...)

	// set the mark price and end auction
	var auctionOrders = []*types.Order{
		{
			Type:        types.OrderTypeLimit,
			Size:        1,
			Remaining:   1,
			Price:       num.NewUint(121500),
			Side:        types.SideBuy,
			Party:       oth1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1,
			Remaining:   1,
			Price:       num.NewUint(121500),
			Side:        types.SideSell,
			Party:       oth2,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}

	// submit the auctions orders
	tm.events = nil
	tm.WithSubmittedOrders(t, auctionOrders...)

	// update the time to get out of auction
	tm.market.OnChainTimeUpdate(context.Background(), auctionEnd)

	t.Run("verify LP orders sizes", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]*proto.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if ord := evt.Order(); ord.PartyId == lpparty {
					found[ord.Id] = ord
				}
			}
		}

		expect := map[string]uint64{
			"V0000000000-0000000001": 124,
			"V0000000000-0000000002": 2,
			"V0000000000-0000000003": 2,
			"V0000000000-0000000004": 3,
			"V0000000000-0000000005": 114,
		}

		for id, v := range found {
			size, ok := expect[id]
			assert.True(t, ok, "unexpected order id")
			assert.Equal(t, v.Size, size, id)
		}
	})

	var newOrders = []*types.Order{
		{
			Type:        types.OrderTypeLimit,
			Size:        1000,
			Remaining:   1000,
			Price:       num.NewUint(121100),
			Side:        types.SideBuy,
			Party:       oth1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1000,
			Remaining:   1000,
			Price:       num.NewUint(122200),
			Side:        types.SideSell,
			Party:       oth2,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}

	// submit the auctions orders
	tm.events = nil
	tm.WithSubmittedOrders(t, newOrders...)
}

func TestRejectedMarketStopLiquidityProvision(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(2),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.WithAccountAndAmount(lpparty, 100000000000000)
	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.7)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: -200},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 98, Offset: 102},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("lp submission is pending", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
	})

	tm.events = nil
	require.NoError(
		t,
		tm.market.Reject(context.Background()),
	)

	t.Run("lp submission is stopped", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusStopped.String())
	})
}

func TestParkOrderPanicOrderNotFoundInBook(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(10),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount(lpparty, 100000000000000)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 1*time.Second)
	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.2)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 99, Offset: -201},
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: -200},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 1, Offset: 100},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 2, Offset: 101},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 98, Offset: 102},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("lp submission is pending", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
	})

	// we end the auction

	tm.EndOpeningAuction(t, auctionEnd, false)

	// this are opening auction party
	// we'll cancel their orders
	var (
		party0 = "clearing-auction-party0"
		party1 = "clearing-auction-party1"

		pegged              = "pegged-order-party"
		peggedInitialAmount = num.NewUint(1000000)
		peggedExpiry        = auctionEnd.Add(10 * time.Minute)

		party4 = "party4"
	)

	tm.WithAccountAndAmount(pegged, peggedInitialAmount.Uint64()).
		WithAccountAndAmount(party4, 10000000000)

	// then cancel all remaining orders in the book.
	confs, err := tm.market.CancelAllOrders(context.Background(), party0)
	assert.NoError(t, err)
	assert.Len(t, confs, 1)
	confs, err = tm.market.CancelAllOrders(context.Background(), party1)
	assert.NoError(t, err)
	assert.Len(t, confs, 1)

	// now we place a pegged order, and ensure it's being
	// parked straight away with another party
	peggedO := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideSell, pegged, 10, 0)
	peggedO.ExpiresAt = peggedExpiry.UnixNano()
	peggedO.PeggedOrder = &types.PeggedOrder{
		Reference: types.PeggedReferenceBestAsk,
		Offset:    10,
	}
	peggedOConf, err := tm.market.SubmitOrder(ctx, peggedO)
	assert.NoError(t, err)
	assert.NotNil(t, peggedOConf)

	t.Run("pegged order is PARKED", func(t *testing.T) {
		// First collect all the orders events
		found := &proto.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found = evt.Order()
			}
		}
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.OrderStatusParked.String())
	})

	// assert the general account is equal to the initial amount
	t.Run("no general account monies where taken", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyGeneralAccount(pegged, tm.asset)
		assert.NoError(t, err)
		assert.True(t, peggedInitialAmount.EQ(acc.Balance))
	})

	t.Run("withdraw all funds from the general account", func(t *testing.T) {
		acc, err := tm.collateralEngine.GetPartyGeneralAccount(pegged, tm.asset)
		assert.NoError(t, err)
		assert.NoError(t,
			tm.collateralEngine.DecrementBalance(context.Background(), acc.ID, peggedInitialAmount.Clone()),
		)

		// then ensure balance is 0
		acc, err = tm.collateralEngine.GetPartyGeneralAccount(pegged, tm.asset)
		assert.NoError(t, err)
		assert.True(t, acc.Balance.IsZero())
	})

	tm.events = nil
	t.Run("party place a new order which should unpark the pegged order", func(t *testing.T) {
		o := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, party4, 10, 2400)
		conf, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
		assert.NotNil(t, conf)
		o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, party4, 10, 800)
		conf2, err := tm.market.SubmitOrder(ctx, o2)
		assert.NoError(t, err)
		assert.NotNil(t, conf2)
		tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(10*time.Second))
	})

	t.Run("pegged order is REJECTED", func(t *testing.T) {
		// First collect all the orders events
		found := &proto.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				order := evt.Order()
				if order.PartyId == pegged {
					found = evt.Order()
				}
			}
		}
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.OrderStatusRejected.String())
	})

	// now move the time to expire the pegged
	timeExpires := peggedExpiry.Add(1 * time.Hour)
	tm.market.OnChainTimeUpdate(ctx, timeExpires)
	orders, err := tm.market.RemoveExpiredOrders(ctx, timeExpires.UnixNano())
	assert.NoError(t, err)
	assert.Len(t, orders, 0)

	// tm.dumpPeggedOrders()
	// tm.dumpMarketData()
}

func TestLotsOfPeggedAndNonPeggedOrders(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(2),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount(lpparty, 100000000000000)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 1*time.Second)
	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.7)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 99, Offset: -201},
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: -200},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 1, Offset: 100},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 2, Offset: 101},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 98, Offset: 102},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("lp submission is pending", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
	})

	// we end the auction

	tm.EndOpeningAuction(t, auctionEnd, false)

	// this are opening auction party
	// we'll cancel their orders
	var (
		party0 = "clearing-auction-party0"
		party1 = "clearing-auction-party1"
		party2 = "party2"
	)

	tm.WithAccountAndAmount(party2, 100000000000000)

	t.Run("lp submission is pending", func(t *testing.T) {
		// then cancel all remaining orders in the book.
		confs, err := tm.market.CancelAllOrders(context.Background(), party0)
		assert.NoError(t, err)
		assert.Len(t, confs, 1)
		confs, err = tm.market.CancelAllOrders(context.Background(), party1)
		assert.NoError(t, err)
		assert.Len(t, confs, 1)
	})

	curt := auctionEnd.Add(1 * time.Second)

	t.Run("party submit volume in both side of the book", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			t.Run("buy side", func(t *testing.T) {
				peggedO := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
					fmt.Sprintf("order-pegged-buy-%v", i), types.SideBuy, party2, 1, 0)
				peggedO.PeggedOrder = &types.PeggedOrder{
					Reference: types.PeggedReferenceBestBid,
					Offset:    -20,
				}
				peggedOConf, err := tm.market.SubmitOrder(ctx, peggedO)
				assert.NoError(t, err)
				assert.NotNil(t, peggedOConf)
				o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
					fmt.Sprintf("order-buy-%v", i), types.SideBuy, party2, 1, uint64(1250+(i*10)))
				conf, err := tm.market.SubmitOrder(ctx, o)
				assert.NoError(t, err)
				assert.NotNil(t, conf)
			})

			t.Run("sell side", func(t *testing.T) {
				peggedO := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
					fmt.Sprintf("order-pegged-sell-%v", i), types.SideSell, party2, 1, 0)
				peggedO.PeggedOrder = &types.PeggedOrder{
					Reference: types.PeggedReferenceBestAsk,
					Offset:    10,
				}
				peggedOConf, err := tm.market.SubmitOrder(ctx, peggedO)
				assert.NoError(t, err)
				assert.NotNil(t, peggedOConf)
				o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
					fmt.Sprintf("order-sell-%v", i), types.SideSell, party2, 1, uint64(950+(i*10)))
				conf, err := tm.market.SubmitOrder(ctx, o)
				assert.NoError(t, err)
				assert.NotNil(t, conf)
			})

			tm.market.OnChainTimeUpdate(ctx, curt)
			curt = curt.Add(1 * time.Second)
		}
	})

	t.Run("party submit 10 buy", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			t.Run("submit buy", func(t *testing.T) {
				o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
					fmt.Sprintf("order-buy-%v", i), types.SideBuy, party2, 1, uint64(550+(i*10)))
				conf, err := tm.market.SubmitOrder(ctx, o)
				assert.NoError(t, err)
				assert.NotNil(t, conf)
			})
			tm.market.OnChainTimeUpdate(ctx, curt)
			curt = curt.Add(1 * time.Second)
		}
	})

	t.Run("party submit 20 sell", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			t.Run("submit buy", func(t *testing.T) {
				o := getMarketOrder(tm, curt, types.OrderTypeLimit, types.OrderTimeInForceGTC,
					fmt.Sprintf("order-buy-%v", i), types.SideSell, party2, 1, uint64(450+(i*10)))
				conf, err := tm.market.SubmitOrder(ctx, o)
				assert.NoError(t, err)
				assert.NotNil(t, conf)
			})
			tm.market.OnChainTimeUpdate(ctx, curt)
			curt = curt.Add(1 * time.Second)
		}
	})
}

func TestMarketValueProxyIsUpdatedWithTrades(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(2),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount(lpparty, 100000000000000)

	tm.market.OnMarketValueWindowLengthUpdate(2 * time.Second)
	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.7)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 99, Offset: -201},
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: -200},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 1, Offset: 100},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 2, Offset: 101},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 98, Offset: 102},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("lp submission is pending", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
	})

	// we end the auction
	// This will also generate trades which are included in the MVP.
	tm.EndOpeningAuction(t, auctionEnd, false)

	// at the end of the opening auction, we check the mvp
	md := tm.market.GetMarketData()
	assert.Equal(t, "10000", md.MarketValueProxy)

	// place bunches of order which will match so we increase the trade value for fee purpose.
	var (
		richParty1 = "rich-party-1"
		richParty2 = "rich-party-2"
	)

	tm.WithAccountAndAmount(richParty1, 100000000000000).
		WithAccountAndAmount(richParty2, 100000000000000)

	orders := []*types.Order{
		{
			Type:        types.OrderTypeLimit,
			Size:        1000,
			Remaining:   1000,
			Price:       num.NewUint(1111),
			Side:        types.SideBuy,
			Party:       richParty1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1000,
			Remaining:   1000,
			Price:       num.NewUint(1111),
			Side:        types.SideSell,
			Party:       richParty2,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}

	// now we have place our trade just after the end of the auction
	// period, and the wwindow is of 2 seconds
	tm.WithSubmittedOrders(t, orders...)

	// we increase the time for 1 second
	// the active_window_length =
	// 1 second so factor = t_market_value_window_length / active_window_length = 2.
	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(1*time.Second))
	md = tm.market.GetMarketData()
	assert.Equal(t, "10000", md.MarketValueProxy)

	// we increase the time for another second
	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(2*time.Second))
	md = tm.market.GetMarketData()
	assert.Equal(t, "10000", md.MarketValueProxy)

	// now we increase the time for another second, which makes us slide
	// out of the window, and reset the tradeValue + window
	// so the mvp is again the total stake submitted in the market
	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(3*time.Second))
	md = tm.market.GetMarketData()
	assert.Equal(t, "10000", md.MarketValueProxy)
}

func TestFeesNotPaidToUndeployedLPs(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(2),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount(lpparty, 100000000000000)

	tm.market.OnMarketValueWindowLengthUpdate(2 * time.Second)
	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.7)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 99, Offset: -201},
			{Reference: types.PeggedReferenceBestBid, Proportion: 1, Offset: -1500},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 1, Offset: 100},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 2, Offset: 101},
			{Reference: types.PeggedReferenceBestAsk, Proportion: 98, Offset: 102},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	t.Run("lp submission is pending", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusPending.String())
	})

	// we end the auction
	// This will also generate trades which are included in the MVP.
	tm.EndOpeningAuction(t, auctionEnd, false)

	// at the end of the opening auction, we check the mvp
	md := tm.market.GetMarketData()
	assert.Equal(t, "10000", md.MarketValueProxy)

	// place bunches of order which will match so we increase the trade value for fee purpose.
	var (
		richParty1 = "rich-party-1"
		richParty2 = "rich-party-2"
	)

	tm.WithAccountAndAmount(richParty1, 100000000000000).
		WithAccountAndAmount(richParty2, 100000000000000)

	orders := []*types.Order{
		{
			Type:        types.OrderTypeLimit,
			Size:        1000,
			Remaining:   1000,
			Price:       num.NewUint(1111),
			Side:        types.SideBuy,
			Party:       richParty1,
			TimeInForce: types.OrderTimeInForceGTC,
		},
		{
			Type:        types.OrderTypeLimit,
			Size:        1000,
			Remaining:   1000,
			Price:       num.NewUint(1111),
			Side:        types.SideSell,
			Party:       richParty2,
			TimeInForce: types.OrderTimeInForceGTC,
		},
	}

	// now we have place our trade just after the end of the auction
	// period, and the wwindow is of 2 seconds
	tm.events = nil
	tm.WithSubmittedOrders(t, orders...)

	tm.events = nil

	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(10*time.Second))

	for _, e := range tm.events {
		switch evt := e.(type) {
		case *events.TransferResponse:
			for _, v := range evt.TransferResponses() {
				// ensure no transfer is a LIQUIDITY_FEE_DISTRIBUTE
				assert.NotEqual(t, types.TransferTypeLiquidityFeeDistribute, v.Transfers[0].Type)
			}
		}
	}
}

func TestLPProviderSubmitLimitOrderWhichExpiresLPOrderAreRedeployed(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	auctionEnd := now.Add(10001 * time.Second)
	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			MakerFee:          num.DecimalFromFloat(0.00025),
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: num.DecimalFromFloat(0.001),
			Tau:                   num.DecimalFromFloat(0.00011407711613050422),
			Params: &types.LogNormalModelParams{
				Mu:    num.DecimalZero(),
				R:     num.DecimalFromFloat(0.016),
				Sigma: num.DecimalFromFloat(5),
			},
		},
	}

	lpparty := "lp-party-1"

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount(lpparty, 100000000000000)

	tm.market.OnMarketValueWindowLengthUpdate(2 * time.Second)
	tm.market.OnSuppliedStakeToObligationFactorUpdate(0.7)
	tm.market.OnChainTimeUpdate(ctx, now)

	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000),
		Fee:              num.DecimalFromFloat(0.5),
		Reference:        "ref-lp-submission-1",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestBid, Proportion: 100, Offset: -10},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceBestAsk, Proportion: 100, Offset: 10},
		},
	}

	// submit our lp
	tm.events = nil
	require.NoError(t,
		tm.market.SubmitLiquidityProvision(
			ctx, lpSubmission, lpparty, "liquidity-submission-1"),
	)

	// we end the auction
	// This will also generate trades which are included in the MVP.
	tm.EndOpeningAuction(t, auctionEnd, false)

	t.Run("lp submission is active", func(t *testing.T) {
		// First collect all the orders events
		var found *proto.LiquidityProvision
		var ord *proto.Order
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.LiquidityProvision:
				found = evt.LiquidityProvision()
			case *events.Order:
				ord = evt.Order()
			}
		}
		require.NotNil(t, found)
		// no update to the liquidity fee
		assert.Equal(t, found.Status.String(), types.LiquidityProvisionStatusActive.String())
		// no update to the liquidity fee
		assert.Equal(t, 15, int(ord.Size))
	})

	// then we'll submit an order which would expire
	// we submit the order at the price of the LP shape generated order
	expiringOrder := getMarketOrder(tm, auctionEnd, types.OrderTypeLimit, types.OrderTimeInForceGTT, "GTT-1", types.SideBuy, lpparty, 19, 890)
	expiringOrder.ExpiresAt = auctionEnd.Add(10 * time.Second).UnixNano()

	tm.events = nil
	_, err := tm.market.SubmitOrder(ctx, expiringOrder)
	assert.NoError(t, err)

	// now we ensure we have 2 order on the buy side.
	// one lp of size 6, on normal limit of size 500
	t.Run("lp order size decrease", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]*proto.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found[evt.Order().Id] = evt.Order()
			}
		}

		assert.Len(t, found, 3)

		expected := map[string]struct {
			size   int
			status types.LiquidityProvisionStatus
		}{
			"V0000000000-0000000001": {
				size:   19,
				status: types.LiquidityProvisionStatusCancelled,
			},
			"V0000000000-0000000002": {
				size:   15,
				status: types.LiquidityProvisionStatusActive,
			},
			"V0000000000-0000000007": {
				size:   19,
				status: types.LiquidityProvisionStatusActive,
			},
		}

		// no ensure that the orders in the map matches the size we have
		for k, v := range found {
			assert.Equal(t, expected[k].size, int(v.Size), k)
			assert.Equal(t, expected[k].status.String(), v.Status.String(), k)
		}
	})

	// now the limit order expires, and the LP order size should increase again
	tm.events = nil
	tm.market.OnChainTimeUpdate(ctx, auctionEnd.Add(11*time.Second))              // this is 1 second after order expiry
	tm.market.RemoveExpiredOrders(ctx, auctionEnd.Add(11*time.Second).UnixNano()) // this is 1 second after order expiry

	// now we ensure we have 2 order on the buy side.
	// one lp of size 6, on normal limit of size 500
	t.Run("lp order size increase again after expiry", func(t *testing.T) {
		// First collect all the orders events
		found := map[string]*proto.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				found[evt.Order().Id] = evt.Order()
			}
		}

		assert.Len(t, found, 3)

		expected := map[string]struct {
			size   uint64
			status types.OrderStatus
		}{
			"V0000000000-0000000001": {19, types.OrderStatusActive},
			// no event sent for expired orders
			// this is done by the excution engine, we may want to do
			// that from the market someday
			"V0000000000-0000000007": {19, types.OrderStatusExpired},
			"V0000000000-0000000002": {15, types.OrderStatusActive},
		}

		// no ensure that the orders in the map matches the size we have
		for k, v := range found {
			assert.Equal(t, expected[k].status.String(), v.Status.String(), k)
			assert.Equal(t, expected[k].size, v.Size, k)
		}
	})
}
