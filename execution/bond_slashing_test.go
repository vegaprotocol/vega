package execution_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setMarkPrice(t *testing.T, mkt *testMarket, duration *types.AuctionDuration, now time.Time, price uint64) {
	// all parties
	parties := []string{"oo-p1", "oo-p4", "oo-p2", "oo-p3"}
	// create accounts for the parties
	for _, p := range parties {
		addAccount(mkt, p)
	}
	orders := []*types.Order{
		{
			MarketId:    mkt.market.GetID(),
			PartyId:     parties[0],
			Side:        types.Side_SIDE_BUY,
			Price:       price - 10,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
			CreatedAt:   now.UnixNano(),
			Reference:   "oo-no-trade-buy",
		},
		{
			MarketId:    mkt.market.GetID(),
			PartyId:     parties[2],
			Side:        types.Side_SIDE_BUY,
			Price:       price,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
			Type:        types.Order_TYPE_LIMIT,
			CreatedAt:   now.UnixNano(),
			Reference:   "oo-trade-buy",
		},
		{
			MarketId:    mkt.market.GetID(),
			PartyId:     parties[3],
			Side:        types.Side_SIDE_SELL,
			Price:       price,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
			Type:        types.Order_TYPE_LIMIT,
			CreatedAt:   now.UnixNano(),
			Reference:   "oo-trade-sell",
		},
		{
			MarketId:    mkt.market.GetID(),
			PartyId:     parties[1],
			Side:        types.Side_SIDE_SELL,
			Price:       price + 10,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
			CreatedAt:   now.UnixNano(),
			Reference:   "oo-no-trade-sell",
		},
	}
	for _, o := range orders {
		_, err := mkt.market.SubmitOrder(context.Background(), o)
		require.NoError(t, err)

	}
	// now fast-forward the market so the auction ends
	now = now.Add(time.Duration(duration.Duration+1) * time.Second)
	mkt.market.OnChainTimeUpdate(context.Background(), now)

	// opening auction ended, mark-price set
	mktData := mkt.market.GetMarketData()
	require.NotNil(t, mktData)
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, mktData.MarketTradingMode)
}

func TestAcceptLiquidityProvisionWithSufficientFunds(t *testing.T) {
	mainParty := "mainParty"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, closingAt, nil, openingAuction)
	initialMarkPrice := uint64(99)
	ctx := context.Background()

	asset := tm.asset

	// end opening auction
	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	mainPartyInitialDeposit := uint64(794) // 794 is the amount required to cover the initial margin on open orderss
	addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)

	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 5, initialMarkPrice+2)

	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp1 := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, "id-lp1")
	require.NoError(t, err)

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp1.CommitmentAmount, bondAcc.Balance)
}

func TestRejectLiquidityProvisionWithInsufficientFundsForInitialMargin(t *testing.T) {
	mainParty := "mainParty"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, closingAt, nil, openingAuction)
	initialMarkPrice := uint64(99)
	ctx := context.Background()

	asset := tm.asset

	// end opening auction
	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	mainPartyInitialDeposit := uint64(199) // 199 is the minimum required amount to meet the commitment amount and maintenance margin on resulting orders
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Transfers[0].ToAccount

	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 5, initialMarkPrice+2)

	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp1 := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, "id-lp1")
	require.Error(t, err)

	assert.Equal(t, 0, tm.market.GetLPSCount())

	var zero uint64 = 0
	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, zero, bondAcc.Balance)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.Equal(t, zero, insurancePool.Balance)

	//TODO: JEREMY: funds are staying in margin ACCOUNT, let's
	// fix that latert.
	marginAcc, err := tm.collateralEngine.GetPartyMarginAccount(tm.mktCfg.Id, mainParty, asset)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, genAcc.Balance, mainPartyInitialDeposit-marginAcc.Balance)

}

func TestCloseoutLPWhenCannotCoverMargin(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, closingAt, nil, openingAuction)
	initialMarkPrice := uint64(99)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	var mainPartyInitialDeposit uint64 = 527 // 794 is the minimum amount to cover additional orders after orderBuyAux1 fills
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Transfers[0].ToAccount
	addAccount(tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 10, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-2", types.Side_SIDE_SELL, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, "id-lp1")
	require.NoError(t, err)

	require.Equal(t, 1, tm.market.GetLPSCount())

	var zero uint64 = 0
	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp.CommitmentAmount, bondAcc.Balance)

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, genAcc.Balance, zero)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeLPCloseout := insurancePool.Balance
	require.Equal(t, zero, insurancePoolBalanceBeforeLPCloseout)

	orderBuyAux1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party2-buy-order-1", types.Side_SIDE_BUY, auxParty1, orderSell1.Size+1, orderSell1.Price)
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	assert.Equal(t, 0, tm.market.GetLPSCount())

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, zero, genAcc.Balance)

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, zero, bondAcc.Balance)

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	insurancePoolBalanceAfterLPCloseout := insurancePool.Balance
	require.Greater(t, insurancePoolBalanceAfterLPCloseout, insurancePoolBalanceBeforeLPCloseout)
}

func TestBondAccountNotUsedForMarginShortageWhenEnoughMoneyInGeneral(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	initialMarkPrice := uint64(99)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, closingAt, nil, openingAuction)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	var mainPartyInitialDeposit uint64 = 1020 // 1020 is the minimum required amount to cover margin without dipping into the bond account
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Transfers[0].ToAccount
	addAccount(tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-2", types.Side_SIDE_SELL, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, initialMarkPrice-2)
	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)
	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, "id-lp1")
	require.NoError(t, err)

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp.CommitmentAmount, bondAcc.Balance)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeMarketMove := insurancePool.Balance
	var zero uint64 = 0
	require.Equal(t, zero, insurancePoolBalanceBeforeMarketMove)

	orderBuyAux1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party2-buy-order-1", types.Side_SIDE_BUY, auxParty1, orderSell1.Size+1, orderSell1.Price)
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, zero, genAcc.Balance)

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp.CommitmentAmount, bondAcc.Balance)

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	insurancePoolBalanceAfterMarketMove := insurancePool.Balance

	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.Equal(t, zero, insurancePoolBalanceAfterMarketMove)
}

func TestBondAccountUsedForMarginShortage_PenaltyPaidFromBondAccount(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	initialMarkPrice := uint64(99)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, closingAt, nil, openingAuction)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	bondPenaltyParameter := 0.1
	tm.market.BondPenaltyFactorUpdate(ctx, bondPenaltyParameter)
	// No fees
	tm.market.OnFeeFactorsInfrastructureFeeUpdate(ctx, 0)
	tm.market.OnFeeFactorsMakerFeeUpdate(ctx, 0)

	var mainPartyInitialDeposit uint64 = 1000 // 1020 is the minimum required amount to cover margin without dipping into the bond account
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Transfers[0].ToAccount
	mainPartyMarginAccID := fmt.Sprintf("%smainParty%s3", tm.market.GetID(), tm.asset)
	addAccount(tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-2", types.Side_SIDE_SELL, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.0",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, "id-lp1")
	require.NoError(t, err)

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	genAccBalanceBeforeMarketMove := genAcc.Balance
	var zero uint64 = 0
	require.Greater(t, genAccBalanceBeforeMarketMove, zero)

	marginAcc, err := tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)
	marginAccBalanceBeforeMarketMove := marginAcc.Balance

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceBeforeMarketMove := bondAcc.Balance
	require.Equal(t, lp.CommitmentAmount, bondAccBalanceBeforeMarketMove)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeMarketMove := insurancePool.Balance
	require.Equal(t, zero, insurancePoolBalanceBeforeMarketMove)

	orderBuyAux1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party2-buy-order-1", types.Side_SIDE_BUY, auxParty1, orderSell1.Size+1, orderSell1.Price)
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	genAccBalanceAfterMarketMove := genAcc.Balance
	require.Equal(t, zero, genAccBalanceAfterMarketMove)

	marginAcc, err = tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)
	marginAccBalanceAfterMarketMove := marginAcc.Balance

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceAfterMarketMove := bondAcc.Balance
	require.Less(t, bondAccBalanceAfterMarketMove, bondAccBalanceBeforeMarketMove)
	require.Greater(t, bondAccBalanceAfterMarketMove, zero)

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	insurancePoolBalanceAfterMarketMove := insurancePool.Balance

	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.Greater(t, insurancePoolBalanceAfterMarketMove, insurancePoolBalanceBeforeMarketMove)

	genAccBalanceChange := int64(genAccBalanceAfterMarketMove) - int64(genAccBalanceBeforeMarketMove)
	marginAccBalanceChange := int64(marginAccBalanceAfterMarketMove) - int64(marginAccBalanceBeforeMarketMove)
	insurancePoolBalanceChange := int64(insurancePoolBalanceAfterMarketMove) - int64(insurancePoolBalanceBeforeMarketMove)
	expectedBondAccBalance := int64(bondAccBalanceBeforeMarketMove) - marginAccBalanceChange - genAccBalanceChange - insurancePoolBalanceChange

	require.Equal(t, expectedBondAccBalance, int64(bondAccBalanceAfterMarketMove))
}

func TestBondAccountUsedForMarginShortagePenaltyPaidFromMarginAccount_NoCloseout(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	initialMarkPrice := uint64(99)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, closingAt, nil, openingAuction)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	bondPenaltyParameter := 0.1
	tm.market.BondPenaltyFactorUpdate(ctx, bondPenaltyParameter)

	var mainPartyInitialDeposit uint64 = 800
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Transfers[0].ToAccount
	mainPartyMarginAccID := fmt.Sprintf("%smainParty%s3", tm.market.GetID(), tm.asset)
	addAccount(tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-2", types.Side_SIDE_SELL, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, "id-lp1")
	require.NoError(t, err)

	var zero uint64 = 0
	marginAcc, err := tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)
	marginAccBalanceBeforeMarketMove := marginAcc.Balance
	require.Greater(t, marginAccBalanceBeforeMarketMove, zero)

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceBeforeMarketMove := bondAcc.Balance
	require.Equal(t, lp.CommitmentAmount, bondAccBalanceBeforeMarketMove)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeMarketMove := insurancePool.Balance

	// Add sell order so LP can be closed out
	orderSellAux1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party2-buy-order-1", types.Side_SIDE_SELL, auxParty1, 10, orderSell1.Price+1)
	confirmationSellAux1, err := tm.market.SubmitOrder(ctx, orderSellAux1)
	require.NotNil(t, confirmationSellAux1)
	require.NoError(t, err)
	require.Equal(t, 0, len(confirmationSellAux1.Trades))

	orderBuyAux1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party2-buy-order-1", types.Side_SIDE_BUY, auxParty1, orderSell1.Size+1, orderSell1.Price)
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, zero, genAcc.Balance)

	marginAccount, err := tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.Greater(t, marginAccount.Balance, zero)

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceAfterMarketMove := bondAcc.Balance
	require.Less(t, bondAccBalanceAfterMarketMove, bondAccBalanceBeforeMarketMove)
	require.Equal(t, bondAccBalanceAfterMarketMove, zero)

	require.Equal(t, 1, tm.market.GetLPSCount())

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	insurancePoolBalanceAfterMarketMove := insurancePool.Balance

	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.Greater(t, insurancePoolBalanceAfterMarketMove, insurancePoolBalanceBeforeMarketMove)
	require.Equal(t, zero, bondAccBalanceAfterMarketMove)
}

func TestBondAccountUsedForMarginShortagePenaltyNotPaidOnTransitionFromAuction(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	ctx := context.Background()
	closingAt := time.Unix(10000000000, 0)
	openingAuctionDuration := &types.AuctionDuration{Duration: 10}
	tm := getTestMarket2(t, now, closingAt, nil, openingAuctionDuration, true)

	mktData := tm.market.GetMarketData()
	require.NotNil(t, mktData)
	require.Equal(t, types.Market_TRADING_MODE_OPENING_AUCTION, mktData.MarketTradingMode)

	initialMarkPrice := uint64(99)

	asset, err := tm.mktCfg.GetAsset()
	require.NoError(t, err)

	var mainPartyInitialDeposit uint64 = 784 // 794 is the minimum required amount to cover margin without dipping into the bond account
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Transfers[0].ToAccount
	addAccount(tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-sell-order-1", types.Side_SIDE_SELL, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "party1-buy-order-1", types.Side_SIDE_BUY, mainParty, 4, initialMarkPrice-2)
	confirmationBuy1, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy1)
	assert.NoError(t, err)
	require.Equal(t, 0, len(confirmationBuy1.Trades))

	lp := &commandspb.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 200,
		Fee:              "0.05",
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: 0},
		},
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 0},
		},
	}

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	genAccBalanceBeforeLPSubmission := genAcc.Balance
	var zero uint64 = 0
	require.Greater(t, genAccBalanceBeforeLPSubmission, zero)

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, "id-lp1")
	require.NoError(t, err)

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	genAccBalanceDuringAuction := genAcc.Balance
	require.Greater(t, genAccBalanceDuringAuction, zero)
	require.Equal(t, genAccBalanceDuringAuction, genAccBalanceBeforeLPSubmission-lp.CommitmentAmount)

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceDuringAuction := bondAcc.Balance
	require.Equal(t, lp.CommitmentAmount, bondAccBalanceDuringAuction)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolDuringAuction := insurancePool.Balance
	require.Equal(t, zero, insurancePoolDuringAuction)

	//End auction
	setMarkPrice(t, tm, openingAuctionDuration, now, initialMarkPrice)

	mktData = tm.market.GetMarketData()
	require.NotNil(t, mktData)
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, mktData.MarketTradingMode)

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	genAccBalanceAfterOpeniningAuction := genAcc.Balance
	require.Equal(t, zero, genAccBalanceAfterOpeniningAuction)

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.Id, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceAfterOpeningAuction := bondAcc.Balance
	require.Less(t, bondAccBalanceAfterOpeningAuction, bondAccBalanceDuringAuction)
	require.Greater(t, bondAccBalanceAfterOpeningAuction, zero)
	require.Less(t, bondAccBalanceAfterOpeningAuction, lp.CommitmentAmount)

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	insurancePoolBalanceAfterOpeningAuction := insurancePool.Balance
	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.Equal(t, insurancePoolBalanceAfterOpeningAuction, insurancePoolDuringAuction)
	require.Equal(t, insurancePoolBalanceAfterOpeningAuction, zero)
}
