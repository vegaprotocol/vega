package spot_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"github.com/stretchr/testify/require"
)

func setupToLeaveOpeningAuction(t *testing.T) (*testMarket, context.Context) {
	t.Helper()
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	addAccountWithAmount(tm, "party1", 100000, "ETH")
	addAccountWithAmount(tm, "party2", 5, "BTC")

	tm.market.StartOpeningAuction(ctx)

	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	_, err := tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	order2 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party2", 1, 30000)
	_, err = tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order2.Party, crypto.RandomHash())
	require.NoError(t, err)

	tm.now = now.Add(2 * time.Second)
	tm.market.OnTick(ctx, tm.now)
	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)
	return tm, ctx
}

func TestNoFeesInOpenintAuction(t *testing.T) {
	tm, _ := setupToLeaveOpeningAuction(t)

	// at this point party 1 bought 1 BTC and has one outstanding order at price 30k as we've left opening auction
	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())

	gaBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30000", gaBalance2.Balance.String())
}

func TestSellerAggressorInContinuousMode(t *testing.T) {
	tm, ctx := setupToLeaveOpeningAuction(t)

	// there's already an open buy order for 30k in the book from the setup so we only need to setup the
	// sell aggressive order
	// setup the aggressive sell order

	// lets check first what is the balance of party2 in the quote asset before the trade
	gaBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30000", gaBalance2.Balance.String()) // the 40k they had from before + the maker fee received

	sellOrder := getGTCLimitOrder(tm, tm.now, crypto.RandomHash(), types.SideSell, "party2", 1, 30000)
	conf, err := tm.market.SubmitOrder(ctx, sellOrder.IntoSubmission(), sellOrder.Party, crypto.RandomHash())
	require.NoError(t, err)
	require.Equal(t, 1, len(conf.Trades))

	// the trade price is 30000, however the aggressor, i.e. the seller pays the fee, instead of getting 30000 they get 30000 - fee
	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40120", gaBalance1.Balance.String()) // the 40k they had from before + the maker fee received

	// they had 30k, they get 30k - fees added to their general account in the quote asset
	gaBalance2, err = tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "59850", gaBalance2.Balance.String())
}

func TestBuyAggressorInContinuousMode(t *testing.T) {
	tm, ctx := setupToLeaveOpeningAuction(t)
	// first lets close the open buy trade in the book

	sellOrder := getGTCLimitOrder(tm, tm.now, crypto.RandomHash(), types.SideSell, "party2", 2, 30000)
	conf, err := tm.market.SubmitOrder(ctx, sellOrder.IntoSubmission(), sellOrder.Party, crypto.RandomHash())
	require.NoError(t, err)
	require.Equal(t, 1, len(conf.Trades))

	// confirm the quote asset balance of the seller after the trade
	gaBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "59850", gaBalance2.Balance.String())

	// now place a buy order to trade against the remaining size of party2, but first let's confirm the quote balance of party1
	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40120", gaBalance1.Balance.String()) // the 40k they had from before + the maker fee received

	// this should be enough to cover the trade + fees
	buyOrder := getGTCLimitOrder(tm, tm.now, crypto.RandomHash(), types.SideBuy, "party1", 1, 30000)
	conf, err = tm.market.SubmitOrder(ctx, buyOrder.IntoSubmission(), buyOrder.Party, crypto.RandomHash())
	require.NoError(t, err)
	require.Equal(t, 1, len(conf.Trades))

	// expect the buyer to have paid the 30k + fees
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "9970", gaBalance1.Balance.String())

	// confirm the quote asset balance of the seller increased by 30k after the trade + 120 from maker fees
	gaBalance2, err = tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "89970", gaBalance2.Balance.String())
}
