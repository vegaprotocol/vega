package execution_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssue2876(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{Duration: 30})
	ctx := context.Background()

	tm.market.OnChainTimeUpdate(ctx, now)

	addAccountWithAmount(tm, "party-0", 100000000)
	addAccountWithAmount(tm, "party-1", 100000000)
	addAccountWithAmount(tm, "party-2", 100000000)
	addAccountWithAmount(tm, "party-3", 100000000)
	addAccountWithAmount(tm, "party-4", 100000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(5)

	orders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "opening1", types.Side_SIDE_BUY, "party-3", 10, 3000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "opening2", types.Side_SIDE_BUY, "party-3", 10, 4000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "opening3", types.Side_SIDE_SELL, "party-4", 10, 4000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "opening4", types.Side_SIDE_SELL, "party-4", 10, 5500),
	}
	for _, o := range orders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "party-0", 20, 3500)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "party-1", 20, 4000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_BUY, "party-2", 10, 5500)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_SELL, "party-2", 10, 5000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	lporder := types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000000),
		Fee:              num.DecimalFromFloat(0.01),
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
				Proportion: 10,
				Offset:     -1000,
			},
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_MID,
				Proportion: 13,
				Offset:     -1500,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
				Proportion: 10,
				Offset:     2000,
			},
			{
				Reference:  types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
				Proportion: 13,
				Offset:     1000,
			},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, &lporder, "party-2", "lp-order-01")
	assert.NoError(t, err)

	bondAccount, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, "party-2", tm.market.GetID(), tm.asset)
	assert.NoError(t, err)
	// we expect the whole commitment to be there
	assert.True(t, bondAccount.Balance.EQ(num.NewUint(1000000)))

	// but also some margin to cover the orders
	marginAccount, err := tm.collateralEngine.GetPartyMarginAccount(tm.market.GetID(), "party-2", tm.asset)
	assert.NoError(t, err)
	assert.True(t, marginAccount.Balance.EQ(num.NewUint(27000)))

	// but also some funds left in the genearal
	generalAccount, err := tm.collateralEngine.GetPartyGeneralAccount("party-2", tm.asset)
	assert.NoError(t, err)
	assert.True(t, generalAccount.Balance.EQ(num.NewUint(98973000)))

	// now let's move time and see
	// this should end the opening auction
	now = now.Add(31 * time.Second)

	tm.market.OnChainTimeUpdate(ctx, now)

	bondAccount, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, "party-2", tm.market.GetID(), tm.asset)
	assert.NoError(t, err)
	// we expect the whole commitment to be there
	assert.True(t, bondAccount.Balance.EQ(num.NewUint(1000000)))

	// but also some margin to cover the orders
	marginAccount, err = tm.collateralEngine.GetPartyMarginAccount(tm.market.GetID(), "party-2", tm.asset)
	assert.NoError(t, err)
	expMargin := num.NewUint(30622560)
	// expMargin := num.NewUint(13200)
	assert.True(t, marginAccount.Balance.EQ(expMargin), "Expected: "+expMargin.String()+" got "+marginAccount.Balance.String())

	expGeneral := num.NewUint(68377440)
	// expGeneral := num.NewUint(98986800)
	// but also some funds left in the genearal
	generalAccount, err = tm.collateralEngine.GetPartyGeneralAccount("party-2", tm.asset)
	assert.NoError(t, err)
	assert.True(t, generalAccount.Balance.EQ(expGeneral), "Expected: "+expGeneral.String()+" got "+generalAccount.Balance.String())
}
