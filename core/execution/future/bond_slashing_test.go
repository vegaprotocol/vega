// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package future_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:unparam
func setMarkPrice(t *testing.T, mkt *testMarket, duration *types.AuctionDuration, now time.Time, price uint64) {
	t.Helper()
	// all parties
	parties := []string{"oo-p1", "oo-p4", "oo-p2", "oo-p3"}
	// create accounts for the parties
	for _, p := range parties {
		addAccount(t, mkt, p)
	}
	delta := num.NewUint(10)
	mPrice := num.NewUint(price)
	orders := []*types.Order{
		{
			MarketID:    mkt.market.GetID(),
			Party:       parties[0],
			Side:        types.SideBuy,
			Price:       num.UintZero().Sub(mPrice, delta),
			Size:        1,
			Remaining:   1,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			CreatedAt:   now.UnixNano(),
			Reference:   "oo-no-trade-buy",
		},
		{
			MarketID:    mkt.market.GetID(),
			Party:       parties[2],
			Side:        types.SideBuy,
			Price:       mPrice,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.OrderTimeInForceGFA,
			Type:        types.OrderTypeLimit,
			CreatedAt:   now.UnixNano(),
			Reference:   "oo-trade-buy",
		},
		{
			MarketID:    mkt.market.GetID(),
			Party:       parties[3],
			Side:        types.SideSell,
			Price:       mPrice,
			Size:        1,
			Remaining:   1,
			TimeInForce: types.OrderTimeInForceGFA,
			Type:        types.OrderTypeLimit,
			CreatedAt:   now.UnixNano(),
			Reference:   "oo-trade-sell",
		},
		{
			MarketID:    mkt.market.GetID(),
			Party:       parties[1],
			Side:        types.SideSell,
			Price:       num.Sum(mPrice, delta),
			Size:        1,
			Remaining:   1,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
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
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	mkt.now = now
	mkt.market.OnTick(ctx, now)

	// opening auction ended, mark-price set
	mktData := mkt.market.GetMarketData()
	require.NotNil(t, mktData)
	require.Equal(t, types.MarketTradingModeContinuous, mktData.MarketTradingMode)
}

const lpprov = "lpprov-party"

func addSimpleLP(t *testing.T, mkt *testMarket, amt uint64) {
	t.Helper()

	lps := &types.LiquidityProvisionSubmission{
		Fee:              num.DecimalFromFloat(0.01),
		MarketID:         mkt.market.GetID(),
		CommitmentAmount: num.NewUint(amt),
	}
	require.NoError(t, mkt.market.SubmitLiquidityProvision(
		context.Background(), lps, lpprov, vgcrypto.RandomHash(),
	))
}

func TestAcceptLiquidityProvisionWithSufficientFunds(t *testing.T) {
	mainParty := "mainParty"
	now := time.Unix(10, 0)
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, nil, openingAuction)
	initialMarkPrice := uint64(99)
	ctx := context.Background()

	asset := tm.asset

	addAccountWithAmount(tm, lpprov, 50000000)
	addSimpleLP(t, tm, 5000000)
	// end opening auction
	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	mainPartyInitialDeposit := uint64(794) // 794 is the amount required to cover the initial margin on open orderss
	addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)

	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-1", types.SideSell, mainParty, 5, initialMarkPrice+2)

	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-buy-order-1", types.SideBuy, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp1 := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200),
		Fee:              num.DecimalFromFloat(0.05),
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, vgcrypto.RandomHash())
	require.NoError(t, err)

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp1.CommitmentAmount, bondAcc.Balance)
}

func TestRejectLiquidityProvisionWithInsufficientFundsForInitialMargin(t *testing.T) {
	mainParty := "mainParty"
	now := time.Unix(10, 0)
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, nil, openingAuction)
	initialMarkPrice := uint64(99)
	ctx := context.Background()

	asset := tm.asset

	addAccountWithAmount(tm, lpprov, 5000000)
	addSimpleLP(t, tm, 5000000)
	// end opening auction
	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	mainPartyInitialDeposit := uint64(347) // 348 is the minimum required amount to meet the commitment amount and maintenance margin on resulting orders
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Entries[0].ToAccount

	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-1", types.SideSell, mainParty, 5, initialMarkPrice+2)

	confirmationSell, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-buy-order-1", types.SideBuy, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp1 := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200),
		Fee:              num.DecimalFromFloat(0.05),
	}

	// Assure that at least the commitment amount can be covered with the initial deposit, otherwise it's a trivial failure (LP can't even afford the bond)
	require.Greater(t, mainPartyInitialDeposit, lp1.CommitmentAmount.Uint64())

	numLpsPriorToSubmission := tm.market.GetLPSCount()

	err = tm.market.SubmitLiquidityProvision(ctx, lp1, mainParty, vgcrypto.RandomHash())
	require.Error(t, err)

	assert.Equal(t, numLpsPriorToSubmission, tm.market.GetLPSCount())

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, num.UintZero(), bondAcc.Balance)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.Equal(t, num.UintZero(), insurancePool.Balance)

	// TODO: JEREMY: funds are staying in margin ACCOUNT, let's
	// fix that latert.
	marginAcc, err := tm.collateralEngine.GetPartyMarginAccount(tm.mktCfg.ID, mainParty, asset)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)

	exp := num.UintZero().Sub(num.NewUint(mainPartyInitialDeposit), marginAcc.Balance)
	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID.ID())
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, genAcc.Balance, exp)
}

func TestCloseoutLPWhenCannotCoverMargin(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, nil, openingAuction)
	initialMarkPrice := uint64(99)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	var mainPartyInitialDeposit uint64 = 527 // 794 is the minimum amount to cover additional orders after orderBuyAux1 fills
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Entries[0].ToAccount.ID()
	addAccount(t, tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-1", types.SideSell, mainParty, 10, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-2", types.SideSell, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-buy-order-1", types.SideBuy, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200),
		Fee:              num.DecimalFromFloat(0.05),
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, vgcrypto.RandomHash())
	require.NoError(t, err)

	require.Equal(t, 1, tm.market.GetLPSCount())

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp.CommitmentAmount, bondAcc.Balance)

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, genAcc.Balance, num.UintZero())

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeLPCloseout := insurancePool.Balance.Clone()
	require.Equal(t, num.UintZero(), insurancePoolBalanceBeforeLPCloseout)

	orderBuyAux1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party2-buy-order-1", types.SideBuy, auxParty1, orderSell1.Size+1, orderSell1.Price.Uint64())
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	assert.Equal(t, 0, tm.market.GetLPSCount())

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, num.UintZero(), genAcc.Balance)

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, num.UintZero(), bondAcc.Balance)

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	insurancePoolBalanceAfterLPCloseout := insurancePool.Balance.Clone()
	require.Greater(t, insurancePoolBalanceAfterLPCloseout, insurancePoolBalanceBeforeLPCloseout)
}

func TestBondAccountNotUsedForMarginShortageWhenEnoughMoneyInGeneral(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	initialMarkPrice := uint64(99)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, nil, openingAuction)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	var mainPartyInitialDeposit uint64 = 1020 // 1020 is the minimum required amount to cover margin without dipping into the bond account
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Entries[0].ToAccount.ID()
	addAccount(t, tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-1", types.SideSell, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-2", types.SideSell, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-buy-order-1", types.SideBuy, mainParty, 4, initialMarkPrice-2)
	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)
	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200),
		Fee:              num.DecimalFromFloat(0.05),
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, vgcrypto.RandomHash())
	require.NoError(t, err)

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp.CommitmentAmount, bondAcc.Balance)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeMarketMove := insurancePool.Balance.Clone()
	require.Equal(t, num.UintZero(), insurancePoolBalanceBeforeMarketMove)

	orderBuyAux1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party2-buy-order-1", types.SideBuy, auxParty1, orderSell1.Size+1, orderSell1.Price.Uint64())
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.Equal(t, num.UintZero(), genAcc.Balance)

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.Equal(t, lp.CommitmentAmount, bondAcc.Balance)

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	insurancePoolBalanceAfterMarketMove := insurancePool.Balance.Clone()

	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.Equal(t, num.UintZero(), insurancePoolBalanceAfterMarketMove)
}

func TestBondAccountUsedForMarginShortage_PenaltyPaidFromBondAccount(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	initialMarkPrice := uint64(99)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, nil, openingAuction)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	bondPenaltyParameter := 0.1
	tm.market.OnMarketLiquidityV2BondPenaltyFactorUpdate(num.DecimalFromFloat(bondPenaltyParameter))
	// No fees
	tm.market.OnFeeFactorsInfrastructureFeeUpdate(ctx, num.DecimalFromFloat(0))
	tm.market.OnFeeFactorsMakerFeeUpdate(ctx, num.DecimalFromFloat(0))

	var mainPartyInitialDeposit uint64 = 1000 // 1020 is the minimum required amount to cover margin without dipping into the bond account
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Entries[0].ToAccount.ID()
	mainPartyMarginAccID := fmt.Sprintf("%smainParty%s3", tm.market.GetID(), tm.asset)
	addAccount(t, tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-1", types.SideSell, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-2", types.SideSell, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-buy-order-1", types.SideBuy, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200),
		Fee:              num.DecimalFromFloat(0.0),
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, vgcrypto.RandomHash())
	require.NoError(t, err)

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.False(t, genAcc.Balance.IsZero())
	genAccBalanceBeforeMarketMove := genAcc.Balance.Clone()

	marginAcc, err := tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)
	marginAccBalanceBeforeMarketMove := marginAcc.Balance.Clone()

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceBeforeMarketMove := bondAcc.Balance.Clone()
	require.Equal(t, lp.CommitmentAmount, bondAccBalanceBeforeMarketMove)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeMarketMove := insurancePool.Balance.Clone()
	require.Equal(t, num.UintZero(), insurancePoolBalanceBeforeMarketMove)

	orderBuyAux1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party2-buy-order-1", types.SideBuy, auxParty1, orderSell1.Size+1, orderSell1.Price.Uint64())
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	genAccBalanceAfterMarketMove := genAcc.Balance.Clone()
	require.True(t, genAcc.Balance.IsZero())

	marginAcc, err = tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)
	marginAccBalanceAfterMarketMove := marginAcc.Balance.Clone()

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceAfterMarketMove := bondAcc.Balance.Clone()
	require.Less(t, bondAccBalanceAfterMarketMove, bondAccBalanceBeforeMarketMove)
	require.False(t, bondAccBalanceAfterMarketMove.IsZero())

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	insurancePoolBalanceAfterMarketMove := insurancePool.Balance.Clone()

	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.True(t, insurancePoolBalanceAfterMarketMove.GT(insurancePoolBalanceBeforeMarketMove))

	genAccBalanceChange, gNeg := num.UintZero().Delta(genAccBalanceAfterMarketMove, genAccBalanceBeforeMarketMove)
	marginAccBalanceChange, mNeg := num.UintZero().Delta(marginAccBalanceAfterMarketMove, marginAccBalanceBeforeMarketMove)
	insurancePoolBalanceChange, iNeg := num.UintZero().Delta(insurancePoolBalanceAfterMarketMove, insurancePoolBalanceBeforeMarketMove)
	// assume all positive
	expBB := num.Sum(bondAccBalanceBeforeMarketMove, genAccBalanceChange, marginAccBalanceChange, insurancePoolBalanceChange)
	if gNeg {
		// we've added, so subtract twice
		expBB.Sub(expBB, num.Sum(genAccBalanceChange, genAccBalanceChange))
	}
	if mNeg {
		expBB.Sub(expBB, num.Sum(marginAccBalanceChange, marginAccBalanceChange))
	}
	if iNeg {
		expBB.Sub(expBB, num.Sum(insurancePoolBalanceChange, insurancePoolBalanceChange))
	}

	require.Equal(t, expBB, bondAccBalanceAfterMarketMove)
}

func TestBondAccountUsedForMarginShortagePenaltyPaidFromMarginAccount_NoCloseout(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	initialMarkPrice := uint64(99)
	ctx := context.Background()
	openingAuction := &types.AuctionDuration{
		Duration: 1,
	}
	tm := getTestMarket(t, now, nil, openingAuction)

	setMarkPrice(t, tm, openingAuction, now, initialMarkPrice)

	asset := tm.asset

	bondPenaltyParameter := 0.1
	tm.market.OnMarketLiquidityV2BondPenaltyFactorUpdate(num.DecimalFromFloat(bondPenaltyParameter))

	var mainPartyInitialDeposit uint64 = 800
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Entries[0].ToAccount.ID()
	mainPartyMarginAccID := fmt.Sprintf("%smainParty%s3", tm.market.GetID(), tm.asset)
	addAccount(t, tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-1", types.SideSell, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderSell2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-2", types.SideSell, mainParty, 1, initialMarkPrice+5)
	confirmationSell2, err := tm.market.SubmitOrder(ctx, orderSell2)
	require.NotNil(t, confirmationSell2)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-buy-order-1", types.SideBuy, mainParty, 4, initialMarkPrice-2)

	confirmationBuy, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy)
	assert.NoError(t, err)

	require.Equal(t, 0, len(confirmationBuy.Trades))

	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200),
		Fee:              num.DecimalFromFloat(0.05),
	}

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, vgcrypto.RandomHash())
	require.NoError(t, err)

	marginAcc, err := tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.NotNil(t, marginAcc)
	require.False(t, marginAcc.Balance.IsZero())

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceBeforeMarketMove := bondAcc.Balance.Clone()
	require.Equal(t, lp.CommitmentAmount, bondAccBalanceBeforeMarketMove)

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset)
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolBalanceBeforeMarketMove := insurancePool.Balance.Clone()

	// Add sell order so LP can be closed out
	orderSellAux1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party2-buy-order-1", types.SideSell, auxParty1, 10, orderSell1.Price.Uint64()+1)
	confirmationSellAux1, err := tm.market.SubmitOrder(ctx, orderSellAux1)
	require.NotNil(t, confirmationSellAux1)
	require.NoError(t, err)
	require.Equal(t, 0, len(confirmationSellAux1.Trades))

	orderBuyAux1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party2-buy-order-1", types.SideBuy, auxParty1, orderSell1.Size+1, orderSell1.Price.Uint64())
	confirmationBuyAux1, err := tm.market.SubmitOrder(ctx, orderBuyAux1)
	require.NotNil(t, confirmationBuyAux1)
	require.NoError(t, err)
	require.Equal(t, 2, len(confirmationBuyAux1.Trades))

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.True(t, genAcc.Balance.IsZero())

	marginAccount, err := tm.collateralEngine.GetAccountByID(mainPartyMarginAccID)
	require.NoError(t, err)
	require.False(t, marginAccount.Balance.IsZero())

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset)
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.True(t, bondAcc.Balance.LT(bondAccBalanceBeforeMarketMove))
	require.True(t, bondAcc.Balance.IsZero())

	require.Equal(t, 1, tm.market.GetLPSCount())

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)

	require.NoError(t, err)
	require.NotNil(t, insurancePool)
	require.True(t, insurancePool.Balance.GT(insurancePoolBalanceBeforeMarketMove))
	require.True(t, bondAcc.Balance.IsZero())
}

func TestBondAccountUsedForMarginShortagePenaltyNotPaidOnTransitionFromAuction(t *testing.T) {
	t.Skip("to be fixed @witold")
	mainParty := "mainParty"
	auxParty1 := "auxParty1"
	now := time.Unix(10, 0)
	ctx := context.Background()
	openingAuctionDuration := &types.AuctionDuration{Duration: 10}
	tm := getTestMarket2(t, now, nil, openingAuctionDuration, true, 0.99)

	mktData := tm.market.GetMarketData()
	require.NotNil(t, mktData)
	require.Equal(t, types.MarketTradingModeOpeningAuction, mktData.MarketTradingMode)

	initialMarkPrice := uint64(99)

	asset, err := tm.mktCfg.GetAssets()
	require.NoError(t, err)

	var mainPartyInitialDeposit uint64 = 784 // 794 is the minimum required amount to cover margin without dipping into the bond account
	transferResp := addAccountWithAmount(tm, mainParty, mainPartyInitialDeposit)
	mainPartyGenAccID := transferResp.Entries[0].ToAccount.ID()
	addAccount(t, tm, auxParty1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-sell-order-1", types.SideSell, mainParty, 5, initialMarkPrice+2)
	confirmationSell1, err := tm.market.SubmitOrder(ctx, orderSell1)
	require.NotNil(t, confirmationSell1)
	require.NoError(t, err)

	orderBuy1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "party1-buy-order-1", types.SideBuy, mainParty, 4, initialMarkPrice-2)
	confirmationBuy1, err := tm.market.SubmitOrder(ctx, orderBuy1)
	assert.NotNil(t, confirmationBuy1)
	assert.NoError(t, err)
	require.Equal(t, 0, len(confirmationBuy1.Trades))

	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(200),
		Fee:              num.DecimalFromFloat(0.05),
	}

	genAcc, err := tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	genAccBalanceBeforeLPSubmission := genAcc.Balance.Clone()
	require.False(t, genAcc.Balance.IsZero())

	err = tm.market.SubmitLiquidityProvision(ctx, lp, mainParty, vgcrypto.RandomHash())
	require.NoError(t, err)

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.False(t, genAcc.Balance.IsZero())
	require.Equal(t, genAcc.Balance, num.UintZero().Sub(genAccBalanceBeforeLPSubmission, lp.CommitmentAmount))

	bondAcc, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset[0])
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	bondAccBalanceDuringAuction := bondAcc.Balance.Clone()
	require.True(t, lp.CommitmentAmount.EQ(bondAcc.Balance))

	insurancePoolAccID := fmt.Sprintf("%s*%s1", tm.market.GetID(), asset[0])
	insurancePool, err := tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NoError(t, err)
	insurancePoolDuringAuction := insurancePool.Balance.Clone()
	require.True(t, insurancePool.Balance.IsZero())

	// End auction
	setMarkPrice(t, tm, openingAuctionDuration, now, initialMarkPrice)

	mktData = tm.market.GetMarketData()
	require.NotNil(t, mktData)
	require.Equal(t, types.MarketTradingModeContinuous, mktData.MarketTradingMode)

	genAcc, err = tm.collateralEngine.GetAccountByID(mainPartyGenAccID)
	require.NoError(t, err)
	require.NotNil(t, genAcc)
	require.True(t, genAcc.Balance.IsZero())

	bondAcc, err = tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, mainParty, tm.mktCfg.ID, asset[0])
	require.NoError(t, err)
	require.NotNil(t, bondAcc)
	require.True(t, bondAcc.Balance.LT(bondAccBalanceDuringAuction))
	require.False(t, bondAcc.Balance.IsZero())
	require.True(t, bondAcc.Balance.LT(lp.CommitmentAmount))

	insurancePool, err = tm.collateralEngine.GetAccountByID(insurancePoolAccID)
	require.NotNil(t, insurancePool)
	require.NoError(t, err)
	require.True(t, insurancePool.Balance.EQ(insurancePoolDuringAuction))
	require.True(t, insurancePool.Balance.IsZero())
}
