package execution_test

import (
	vegacontext "code.vegaprotocol.io/vega/libs/context"
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
	tm := getTestMarketWithDP(t, now, closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 30}, 3)
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, randomSha256Hash())

	tm.market.OnChainTimeUpdate(ctx, now)

	addAccountWithAmount(tm, "party-0", 100000000)
	addAccountWithAmount(tm, "party-1", 100000000)
	addAccountWithAmount(tm, "party-2", 100000000)
	addAccountWithAmount(tm, "party-3", 100000000)
	addAccountWithAmount(tm, "party-4", 100000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(5))

	orders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "opening1", types.SideBuy, "party-3", 10, 3000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "opening2", types.SideBuy, "party-3", 10, 4000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "opening3", types.SideSell, "party-4", 10, 4000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "opening4", types.SideSell, "party-4", 10, 5500),
	}
	for _, o := range orders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-0", 20, 3500)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-1", 20, 4000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideBuy, "party-2", 10, 5500)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "party-2", 10, 5000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	lporder := types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000000),
		Fee:              num.DecimalFromFloat(0.01),
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceBestBid,
				Proportion: 10,
				Offset:     num.NewUint(1000),
			},
			{
				Reference:  types.PeggedReferenceMid,
				Proportion: 13,
				Offset:     num.NewUint(1500),
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceBestAsk,
				Proportion: 10,
				Offset:     num.NewUint(2000),
			},
			{
				Reference:  types.PeggedReferenceBestAsk,
				Proportion: 13,
				Offset:     num.NewUint(1000),
			},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, &lporder, "party-2", "lp-order-01", randomSha256Hash())
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

	expMargin := num.NewUint(24001200)
	// expMargin := num.NewUint(30622560)
	assert.True(t, marginAccount.Balance.EQ(expMargin), "Expected: "+expMargin.String()+" got "+marginAccount.Balance.String())

	expGeneral := num.NewUint(74998800)
	// expGeneral := num.NewUint(68377440)
	// but also some funds left in the genearal
	generalAccount, err = tm.collateralEngine.GetPartyGeneralAccount("party-2", tm.asset)
	assert.NoError(t, err)
	assert.True(t, generalAccount.Balance.EQ(expGeneral), "Expected: "+expGeneral.String()+" got "+generalAccount.Balance.String())
}
