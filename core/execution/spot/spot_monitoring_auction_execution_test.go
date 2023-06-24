package spot_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/require"
)

func TestMonitoringAuction(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	addAccountWithAmount(tm, "party1", 100000, "ETH")
	addAccountWithAmount(tm, "party2", 5, "BTC")

	tm.market.StartOpeningAuction(ctx)

	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	conf1, err := tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	order2 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party2", 1, 30000)
	_, err = tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order2.Party, crypto.RandomHash())
	require.NoError(t, err)

	tm.market.OnTick(ctx, now.Add(2*time.Second))
	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// at this point party 1 bought 1 BTC and has one outstanding order at price 30k as we've left opening auction
	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30000", haBalance1.Balance.String())
	gaBaseBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "1", gaBaseBalance1.Balance.String())

	// move the price of the remaining order of party1 significantly
	_, err = tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf1.Order.ID, Price: num.NewUint(10000)}, "party1", crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "10000", haBalance1.Balance.String())

	order3 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party2", 1, 10000)
	_, err = tm.market.SubmitOrder(ctx, order3.IntoSubmission(), order3.Party, crypto.RandomHash())
	require.NoError(t, err)

	tm.market.OnTick(ctx, now.Add(2*time.Second))
	md = tm.market.GetMarketData()
	// we're in price monitoring!!!
	require.Equal(t, types.MarketTradingModeMonitoringAuction, md.MarketTradingMode)

	// check that the buyer transfers into holding the expected fees amounts
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "59995", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "10005", haBalance1.Balance.String())

	gaBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30000", gaBalance2.Balance.String())

	tm.market.OnTick(ctx, now.Add(60*time.Minute))
	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// we're out of the monitoring auction. expect the trade to have happened and both parties paying fees
	// both pay infra fee, none gets maker fee as we're in auction
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "59995", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBalance1.Balance.String())
	gaBaseBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "2", gaBaseBalance1.Balance.String())

	gaBaseBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "3", gaBaseBalance2.Balance.String())
	gaalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "39995", gaalance2.Balance.String())
}

func TestMonitoringAuctionInsufficientFundsToCoverFeesOnAmend(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	addAccountWithAmount(tm, "party1", 100000, "ETH")
	addAccountWithAmount(tm, "party2", 5, "BTC")

	tm.market.StartOpeningAuction(ctx)

	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	conf1, err := tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	order2 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party2", 1, 30000)
	_, err = tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order2.Party, crypto.RandomHash())
	require.NoError(t, err)

	tm.market.OnTick(ctx, now.Add(2*time.Second))
	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// move the price of the remaining order of party1 significantly
	_, err = tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf1.Order.ID, Price: num.NewUint(10000)}, "party1", crypto.RandomHash())
	require.NoError(t, err)

	// submit an order to get us into price monitoring auction
	order3 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party2", 1, 10000)
	_, err = tm.market.SubmitOrder(ctx, order3.IntoSubmission(), order3.Party, crypto.RandomHash())
	require.NoError(t, err)

	tm.market.OnTick(ctx, now.Add(2*time.Second))
	md = tm.market.GetMarketData()
	// we're in price monitoring!!!
	require.Equal(t, types.MarketTradingModeMonitoringAuction, md.MarketTradingMode)

	// try to increase the size by 6 for party 1, this would require them to have in their general account 60k + infra fee which they don't have so expect a failure
	_, err = tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf1.Order.ID, Price: num.NewUint(10000), SizeDelta: 6}, "party1", crypto.RandomHash())
	require.Equal(t, "party does not have sufficient balance to cover the trade and fees", err.Error())
}
