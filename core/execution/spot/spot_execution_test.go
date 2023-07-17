package spot_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/require"
)

func TestOpeningAuction(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	addAccountWithAmount(tm, "party1", 10, "BTC")
	addAccountWithAmount(tm, "party1", 100000, "ETH")
	addAccountWithAmount(tm, "party2", 5, "BTC")
	addAccountWithAmount(tm, "party3", 500, "ETH")
	addAccountWithAmount(tm, "party3", 2, "BTC")
	addAccountWithAmount(tm, "party4", 10000, "ETH")

	tm.market.StartOpeningAuction(ctx)

	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	order2 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party2", 1, 32000)
	tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order2.Party, crypto.RandomHash())

	gaBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "4", gaBalance2.Balance.String())

	haBalance2, err := tm.collateralEngine.GetPartyHoldingAccount("party2", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "1", haBalance2.Balance.String())

	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeOpeningAuction, md.MarketTradingMode)

	order3 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party3", 1, 30000)
	tm.market.SubmitOrder(ctx, order3.IntoSubmission(), order3.Party, crypto.RandomHash())

	haBalance3, err := tm.collateralEngine.GetPartyHoldingAccount("party3", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "1", haBalance3.Balance.String())

	gaBalance3, err := tm.collateralEngine.GetPartyGeneralAccount("party3", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "1", gaBalance3.Balance.String())

	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeOpeningAuction, md.MarketTradingMode)
	tm.market.OnTick(ctx, now.Add(2*time.Second))
	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// trade has been done between party1 and party3, lets check balances
	// party 1 had 60k ETH in holding and 40k ETH in their general account. The bought 1 BTC for 30k ETH at the end of the opening auction so now they have:
	// 11 BTC in their BTC general account (10 they had + 1 they bought)
	// 40k ETH in their ETH general account
	// 30k ETH in their holding account
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())

	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30000", haBalance1.Balance.String())

	gaBaseBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "11", gaBaseBalance1.Balance.String())

	// party 3 has 2 BTC and 500 ETH
	// they sold 1 BTC and received 30k ETH so now they have:
	// 1 BTC
	// 30500 ETH
	// and nothing in the holding account
	gaBalance3, err = tm.collateralEngine.GetPartyGeneralAccount("party3", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30500", gaBalance3.Balance.String())

	gaBaseBalance3, err := tm.collateralEngine.GetPartyGeneralAccount("party3", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "1", gaBaseBalance3.Balance.String())

	haBalance3, err = tm.collateralEngine.GetPartyHoldingAccount("party3", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBalance3.Balance.String())
}

func TestAmend(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	addAccountWithAmount(tm, "party1", 100000, "ETH")
	tm.market.StartOpeningAuction(ctx)

	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	conf, err := tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	// increase price reduce size, now we expect the holding to have only 40k and general 60k
	conf, err = tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf.Order.ID, Price: num.NewUint(40000), SizeDelta: -1}, "party1", crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", haBalance1.Balance.String())

	// increase size, reduce price
	_, err = tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf.Order.ID, Price: num.NewUint(20000), SizeDelta: 2}, "party1", crypto.RandomHash())
	require.NoError(t, err)

	// should be back to 60k in holding and 40k in general
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())
}

func TestCancelAll(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	addAccountWithAmount(tm, "party1", 100000, "ETH")
	tm.market.StartOpeningAuction(ctx)

	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	_, err := tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	order2 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 15000)
	_, err = tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "10000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "90000", haBalance1.Balance.String())

	tm.market.CancelAllOrders(ctx, "party1")
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "100000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBalance1.Balance.String())
}

func TestCancelSingleOrder(t *testing.T) {
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

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	order2 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 15000)
	conf2, err := tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "10000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "90000", haBalance1.Balance.String())

	// cancel the second order, now we should have
	tm.market.CancelOrder(ctx, "party1", conf2.Order.ID, crypto.RandomHash())
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	// cancel the first order to have the full amount back in the general account
	tm.market.CancelOrder(ctx, "party1", conf1.Order.ID, crypto.RandomHash())
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "100000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBalance1.Balance.String())
}

func TestInsufficientCoverInSubmit(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	addAccountWithAmount(tm, "party1", 100000, "ETH")
	tm.market.StartOpeningAuction(ctx)

	// submit an order with insufficient cover should fail
	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 4, 30000)
	_, err := tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.Equal(t, "party does not have sufficient balance to cover the trade and fees", err.Error())

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "100000", gaBalance1.Balance.String())

	// now submit a valid one
	order1 = getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	_, err = tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	// and again an invalid one
	order2 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 30000)
	_, err = tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order2.Party, crypto.RandomHash())
	require.Equal(t, "party does not have sufficient balance to cover the trade and fees", err.Error())

	// and again a valid one
	order2 = getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 2, 20000)
	_, err = tm.market.SubmitOrder(ctx, order2.IntoSubmission(), order2.Party, crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "0", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "100000", haBalance1.Balance.String())
}

func TestNoValidAccount(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, crypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)

	tm.market.StartOpeningAuction(ctx)

	// submit an order with insufficient cover should fail
	order1 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideBuy, "party1", 4, 30000)
	_, err := tm.market.SubmitOrder(ctx, order1.IntoSubmission(), order1.Party, crypto.RandomHash())
	require.Error(t, common.ErrPartyInsufficientAssetBalance, err)
}

func TestPartialFill(t *testing.T) {
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

	// expect one trade of size 1
	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30000", haBalance1.Balance.String())
	gaBaseBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "1", gaBaseBalance1.Balance.String())

	// party 2 should get the 30k quote and should have 1 less base
	gaBaseBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "4", gaBaseBalance2.Balance.String())
	haBaseBalance2, err := tm.collateralEngine.GetPartyHoldingAccount("party2", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBaseBalance2.Balance.String())
	gaBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "30000", gaBalance2.Balance.String())

	tm.market.OnTick(ctx, now.Add(2*time.Second))
	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// increase the remaining size to 2, decrease the size to 29900
	conf, err := tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf1.Order.ID, SizeDelta: 1, Price: num.NewUint(29900)}, "party1", crypto.RandomHash())
	require.NoError(t, err)
	require.Equal(t, 0, len(conf.Trades))
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "10200", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "59800", haBalance1.Balance.String())

	tm.market.OnTick(ctx, now.Add(2*time.Second))
	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// now fill some and then cancel
	order3 := getGTCLimitOrder(tm, now, crypto.RandomHash(), types.SideSell, "party2", 1, 29900)
	conf3, err := tm.market.SubmitOrder(ctx, order3.IntoSubmission(), order3.Party, crypto.RandomHash())
	require.NoError(t, err)
	require.Equal(t, 1, len(conf3.Trades))

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "10320", gaBalance1.Balance.String()) // maker fees paid to party1
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "29900", haBalance1.Balance.String())
	gaBaseBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "2", gaBaseBalance1.Balance.String())

	gaBaseBalance2, err = tm.collateralEngine.GetPartyGeneralAccount("party2", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "3", gaBaseBalance2.Balance.String())
	haBaseBalance2, err = tm.collateralEngine.GetPartyHoldingAccount("party2", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBaseBalance2.Balance.String())
	gaBalance2, err = tm.collateralEngine.GetPartyGeneralAccount("party2", tm.quoteAsset)
	require.NoError(t, err)
	// party2 is the aggressor so they pay the fees which are deducted from their 29990 payment leaving 29750 (240 paid in fees)
	require.Equal(t, "59750", gaBalance2.Balance.String())

	// cancel all orders for party1
	tm.market.CancelAllOrders(ctx, "party1")
	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40220", gaBalance1.Balance.String()) // maker fees paid to party1
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBalance1.Balance.String())
	gaBaseBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.baseAsset)
	require.NoError(t, err)
	require.Equal(t, "2", gaBaseBalance1.Balance.String())
}

func TestIncreaseHolding(t *testing.T) {
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

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	_, err = tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf1.Order.ID, Price: num.NewUint(25000), SizeDelta: 1}, "party1", crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "25000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "75000", haBalance1.Balance.String())
}

func TestDecreaseHolding(t *testing.T) {
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

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "40000", gaBalance1.Balance.String())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "60000", haBalance1.Balance.String())

	_, err = tm.market.AmendOrder(ctx, &types.OrderAmendment{OrderID: conf1.Order.ID, Price: num.NewUint(45000), SizeDelta: -1}, "party1", crypto.RandomHash())
	require.NoError(t, err)

	gaBalance1, err = tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "55000", gaBalance1.Balance.String())
	haBalance1, err = tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "45000", haBalance1.Balance.String())
}
