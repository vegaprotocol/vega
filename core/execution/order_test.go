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

package execution_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAmendMarginCheckFails(t *testing.T) {
	lpparty := "lp-party-1"
	lpparty2 := "lp-party-2"
	lpparty3 := "lp-party-3"

	p1 := "p1"
	p2 := "p2"

	party1 := "party1"

	now := time.Unix(10, 0)
	auctionEnd := now.Add(10001 * time.Second)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		// the liquidity provider
		WithAccountAndAmount(lpparty, 500000000000).
		WithAccountAndAmount(lpparty2, 500000000000).
		WithAccountAndAmount(lpparty3, 500000000000).
		WithAccountAndAmount(p1, 500000000000).
		WithAccountAndAmount(p2, 500000000000)
	addAccountWithAmount(tm, "lpprov", 10000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(.2))
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(55000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 50),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 53),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 50),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 53),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	tm.EndOpeningAuction(t, auctionEnd, false)

	addAccountWithAmount(tm, party1, 18001)

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}

	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	orderID := confirmation.Order.ID

	// Amend the price to force a cancel+resubmit to the order book

	amend := &types.OrderAmendment{
		OrderID:   orderID,
		MarketID:  confirmation.Order.MarketID,
		SizeDelta: 100000,
	}

	tm.events = nil
	amended, err := tm.market.AmendOrder(context.TODO(), amend, confirmation.Order.Party, vgcrypto.RandomHash())
	assert.Nil(t, amended)
	assert.EqualError(t, err, "margin check failed")
	// ensure no events for the update order were sent
	assert.Len(t, tm.events, 2)
	// first event was to update the positions
	assert.Equal(t, int64(100100), tm.events[0].(*events.PositionState).PotentialBuys())
	// second to restore to the initial size as margin check failed
	assert.Equal(t, int64(100), tm.events[1].(*events.PositionState).PotentialBuys())

	// now we just cancel it and see if all is fine.
	tm.events = nil
	cancelled, err := tm.market.CancelOrder(context.TODO(), confirmation.Order.Party, orderID, vgcrypto.RandomHash())
	assert.NotNil(t, cancelled)
	assert.NoError(t, err)
	assert.Equal(t, cancelled.Order.Status, types.OrderStatusCancelled)

	found := false
	for _, v := range tm.events {
		if o, ok := v.(*events.Order); ok && o.Order().Id == orderID {
			assert.Equal(t, o.Order().Status, types.OrderStatusCancelled)
			found = true
		}
	}
	assert.True(t, found)
}

func TestOrderBufferOutputCount(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, party1)

	orderBuy := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Status:      types.OrderStatusActive,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   0,
		Reference:   "party1-buy-order",
	}
	orderAmend := *orderBuy

	// Create an order (generates one order message)
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Create a new order (generates one order message)
	orderAmend.ID = "amendingorder"
	orderAmend.Reference = "amendingorderreference"
	confirmation, err = tm.market.SubmitOrder(context.TODO(), &orderAmend)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	amend := &types.OrderAmendment{
		MarketID: tm.market.GetID(),
		OrderID:  orderAmend.ID,
	}

	one := num.NewUint(1)
	// Amend price down (generates one order message)
	amend.Price = num.UintZero().Sub(orderBuy.Price, one)
	amendConf, err := tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend price up (generates one order message)
	amend.Price.AddSum(one, one) // we subtracted one, add 1 to get == to orderBuy.Price, + 1 again
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend size down (generates one order message)
	amend.Price = nil
	amend.SizeDelta = -1
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend size up (generates one order message)
	amend.SizeDelta = +1
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend TIME_IN_FORCE -> GTT (generates one order message)
	amend.SizeDelta = 0
	amend.TimeInForce = types.OrderTimeInForceGTT
	exp := now.UnixNano() + 100000000000
	amend.ExpiresAt = &exp
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend TIME_IN_FORCE -> GTC (generates one order message)
	amend.TimeInForce = types.OrderTimeInForceGTC
	amend.ExpiresAt = nil
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend ExpiresAt (generates two order messages)
	amend.TimeInForce = types.OrderTimeInForceGTT
	exp = now.UnixNano() + 100000000000
	amend.ExpiresAt = &exp
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	exp = now.UnixNano() + 200000000000
	amend.ExpiresAt = &exp
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)
}

func TestAmendCancelResubmit(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, party1)

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	orderID := confirmation.Order.ID

	// Amend the price to force a cancel+resubmit to the order book

	amend := &types.OrderAmendment{
		OrderID:  orderID,
		MarketID: confirmation.Order.MarketID,
		Price:    num.NewUint(101),
	}
	amended, err := tm.market.AmendOrder(context.TODO(), amend, confirmation.Order.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	amend = &types.OrderAmendment{
		OrderID:   orderID,
		MarketID:  confirmation.Order.MarketID,
		Price:     num.NewUint(101),
		SizeDelta: 1,
	}
	amended, err = tm.market.AmendOrder(context.TODO(), amend, confirmation.Order.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amended)
	assert.NoError(t, err)
}

func TestCancelWithWrongPartyID(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Now attempt to cancel it with the wrong partyID
	cancelOrder := &types.OrderCancellation{
		OrderID:  confirmation.Order.ID,
		MarketID: confirmation.Order.MarketID,
	}
	cancelconf, err := tm.market.CancelOrder(context.TODO(), party2, cancelOrder.OrderID, vgcrypto.RandomHash())
	assert.Nil(t, cancelconf)
	assert.Error(t, err, types.ErrInvalidPartyID)
}

func TestMarkPriceUpdateAfterPartialFill(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(25000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), now)

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(10),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
		Type:        types.OrderTypeLimit,
	}
	// Submit the original order
	buyConfirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	orderSell := &types.Order{
		Status:      types.OrderStatusActive,
		TimeInForce: types.OrderTimeInForceIOC,
		ID:          "someid",
		Side:        types.SideSell,
		Party:       party2,
		MarketID:    tm.market.GetID(),
		Size:        50,
		Price:       num.NewUint(10),
		Remaining:   50,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
		Type:        types.OrderTypeMarket,
	}
	// Submit an opposite order to partially fill
	sellConfirmation, err := tm.market.SubmitOrder(context.TODO(), orderSell)
	assert.NotNil(t, sellConfirmation)
	assert.NoError(t, err)

	// Validate that the mark price has been updated
	assert.True(t, tm.market.GetMarketData().MarkPrice.EQ(num.NewUint(10)))
}

func TestExpireCancelGTCOrder(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, party1)

	orderBuy := &types.Order{
		CreatedAt:   int64(now.Second()),
		Status:      types.OrderStatusActive,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(10),
		Remaining:   100,
		Reference:   "party1-buy-order",
		Type:        types.OrderTypeLimit,
	}
	// Submit the original order
	buyConfirmation, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	tm.now = time.Unix(10, 100)
	tm.market.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), time.Unix(10, 100))
	exp := int64(10000000010)
	amend := &types.OrderAmendment{
		OrderID:     buyConfirmation.Order.ID,
		MarketID:    tm.market.GetID(),
		ExpiresAt:   &exp,
		TimeInForce: types.OrderTimeInForceGTT,
	}

	amended, err := tm.market.AmendOrder(context.Background(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	// Validate that the mark price has been updated
	assert.EqualValues(t, amended.Order.TimeInForce, types.OrderTimeInForceGTT)
	assert.EqualValues(t, amended.Order.Status, types.OrderStatusExpired)
	assert.EqualValues(t, amended.Order.CreatedAt, 10000000000)
	assert.EqualValues(t, amended.Order.ExpiresAt, exp)
	assert.EqualValues(t, 10000000100, amended.Order.UpdatedAt)
}

func TestAmendPartialFillCancelReplace(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 5),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 5),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(25000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), now)

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		TimeInForce: types.OrderTimeInForceGTC,
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        20,
		Price:       num.NewUint(5),
		Remaining:   20,
		Reference:   "party1-buy-order",
		Type:        types.OrderTypeLimit,
	}
	// Place an order
	buyConfirmation, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	orderSell := &types.Order{
		Status:      types.OrderStatusActive,
		TimeInForce: types.OrderTimeInForceIOC,
		Side:        types.SideSell,
		Party:       party2,
		MarketID:    tm.market.GetID(),
		Size:        10,
		Price:       num.NewUint(5),
		Remaining:   10,
		Reference:   "party2-sell-order",
		Type:        types.OrderTypeMarket,
	}
	// Partially fill the original order
	sellConfirmation, err := tm.market.SubmitOrder(context.Background(), orderSell)
	assert.NotNil(t, sellConfirmation)
	assert.NoError(t, err)

	amend := &types.OrderAmendment{
		OrderID:  buyConfirmation.Order.ID,
		MarketID: tm.market.GetID(),
		Price:    num.NewUint(20),
	}
	amended, err := tm.market.AmendOrder(context.Background(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	// Check the values are correct
	assert.True(t, amended.Order.Price.EQ(amend.Price))
	assert.EqualValues(t, amended.Order.Remaining, 16)
	assert.EqualValues(t, amended.Order.Size, 20)
}

func TestAmendWrongPartyID(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Send an amend but use the wrong partyID
	amend := &types.OrderAmendment{
		OrderID:  confirmation.Order.ID,
		MarketID: confirmation.Order.MarketID,
		Price:    num.NewUint(101),
	}
	amended, err := tm.market.AmendOrder(context.Background(), amend, party2, vgcrypto.RandomHash())
	assert.Nil(t, amended)
	assert.Error(t, err, types.ErrInvalidPartyID)
}

func TestPartialFilledWashTrade(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 55),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 55),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(25000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	orderSell1 := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Side:        types.SideSell,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        15,
		Price:       num.NewUint(55),
		Remaining:   15,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-sell-order",
	}
	confirmation, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	orderSell2 := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Side:        types.SideSell,
		Party:       party2,
		MarketID:    tm.market.GetID(),
		Size:        15,
		Price:       num.NewUint(53),
		Remaining:   15,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// This order should partially fill and then be rejected
	orderBuy1 := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        30,
		Price:       num.NewUint(60),
		Remaining:   30,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, confirmation.Order.Status, types.OrderStatusPartiallyFilled)
	assert.Equal(t, confirmation.Order.Remaining, uint64(15))
}

func getAmend(market string, orderID string, sizeDelta int64, price uint64, tif types.OrderTimeInForce, expiresAt int64) *types.OrderAmendment {
	amend := &types.OrderAmendment{
		OrderID:     orderID,
		MarketID:    market,
		SizeDelta:   sizeDelta,
		TimeInForce: tif,
	}

	if price > 0 {
		amend.Price = num.NewUint(price)
	}

	if expiresAt > 0 {
		amend.ExpiresAt = &expiresAt
	}

	return amend
}

func amendOrder(t *testing.T, tm *testMarket, party string, orderID string, sizeDelta int64, price uint64,
	tif types.OrderTimeInForce, expiresAt int64, pass bool,
) {
	t.Helper()
	amend := getAmend(tm.market.GetID(), orderID, sizeDelta, price, tif, expiresAt)

	amended, err := tm.market.AmendOrder(context.Background(), amend, party, vgcrypto.RandomHash())
	if pass {
		assert.NotNil(t, amended)
		assert.NoError(t, err)
	}
}

//nolint:unparam
func getOrder(t *testing.T, tm *testMarket, now *time.Time, orderType types.OrderType, tif types.OrderTimeInForce,
	expiresAt int64, side types.Side, party string, size uint64, price uint64,
) types.Order {
	t.Helper()
	order := types.Order{
		Status:      types.OrderStatusActive,
		Type:        orderType,
		TimeInForce: tif,
		Side:        side,
		Party:       party,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       num.NewUint(price),
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "",
	}

	if expiresAt > 0 {
		order.ExpiresAt = expiresAt
	}
	return order
}

func sendOrder(t *testing.T, tm *testMarket, now *time.Time, orderType types.OrderType, tif types.OrderTimeInForce, expiresAt int64, side types.Side, party string,
	size uint64, price uint64,
) string {
	t.Helper()
	order := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        orderType,
		TimeInForce: tif,
		Side:        side,
		Party:       party,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       num.NewUint(price),
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "",
	}

	if expiresAt > 0 {
		order.ExpiresAt = expiresAt
	}

	confirmation, err := tm.market.SubmitOrder(context.Background(), order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)
	return confirmation.Order.ID
}

func TestAmendToFill(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")

	// test_AmendMarketOrderFail
	_ = sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 100)      // 1 - a8
	_ = sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 110)      // 1 - a8
	_ = sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 120)      // 1 - a8
	orderID := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party2", 40, 50) // 1 - a8
	amendOrder(t, tm, "party2", orderID, 0, 500, types.OrderTimeInForceUnspecified, 0, true)
}

func TestAmendToLosePriorityThenCancel(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")

	// Create 2 orders at the same level
	order1 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 100)
	_ = sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 100)

	// Amend the first order to make it lose time priority
	amendOrder(t, tm, "party1", order1, 1, 0, types.OrderTimeInForceUnspecified, 0, true)

	// Check we can cancel it
	cancelconf, _ := tm.market.CancelOrder(context.TODO(), "party1", order1, vgcrypto.RandomHash())
	assert.NotNil(t, cancelconf)
	assert.Equal(t, types.OrderStatusCancelled, cancelconf.Order.Status)
}

func TestUnableToAmendGFAGFN(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{Duration: 1})
	mainParty := "party1"
	auxParty := "party2"
	auxParty2 := "party22"
	addAccount(t, tm, mainParty)
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(25000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), now)

	// test_AmendMarketOrderFail
	orderID := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, mainParty, 10, 100)
	amendOrder(t, tm, mainParty, orderID, 0, 0, types.OrderTimeInForceGFA, 0, false)
	amendOrder(t, tm, mainParty, orderID, 0, 0, types.OrderTimeInForceGFN, 0, false)

	orderID2 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGFN, 0, types.SideSell, mainParty, 10, 100)
	amendOrder(t, tm, mainParty, orderID2, 0, 0, types.OrderTimeInForceGTC, 0, false)
	amendOrder(t, tm, mainParty, orderID2, 0, 0, types.OrderTimeInForceGFA, 0, false)

	// EnterAuction should actually trigger an auction here...
	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(context.Background())
	orderID3 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGFA, 0, types.SideSell, "party1", 10, 100)
	amendOrder(t, tm, "party1", orderID3, 0, 0, types.OrderTimeInForceGTC, 0, false)
	amendOrder(t, tm, "party1", orderID3, 0, 0, types.OrderTimeInForceGFN, 0, false)
}

func TestMarketPeggedOrders(t *testing.T) {
	t.Run("pegged orders must be LIMIT orders ", testPeggedOrderTypes)
	t.Run("pegged orders must be either GTT or GTC ", testPeggedOrderTIFs)
	t.Run("pegged orders buy side validation", testPeggedOrderBuys)
	t.Run("pegged orders sell side validation", testPeggedOrderSells)
	t.Run("pegged orders are parked when price below 0", testPeggedOrderParkWhenPriceBelowZero)
	t.Run("pegged orders are parked when price reprices below 0", testPeggedOrderParkWhenPriceRepricesBelowZero)
	t.Run("pegged order when there is no market prices", testPeggedOrderAddWithNoMarketPrice)
	t.Run("pegged order add to order book", testPeggedOrderAdd)
	t.Run("pegged order test when placing a pegged order forces a reprice", testPeggedOrderWithReprice)
	t.Run("pegged order entry during an auction", testPeggedOrderParkWhenInAuction)
	t.Run("Pegged orders unpark order after leaving auction", testPeggedOrderUnparkAfterLeavingAuction)
	t.Run("pegged order repricing", testPeggedOrderRepricing)
	t.Run("pegged order check that a filled pegged order is handled correctly", testPeggedOrderFilledOrder)
	t.Run("parked orders during normal trading are unparked when possible", testParkedOrdersAreUnparkedWhenPossible)
	t.Run("pegged orders are handled correctly when moving into auction", testPeggedOrdersEnteringAuction)
	t.Run("pegged orders are handled correctly when moving out of auction", testPeggedOrdersLeavingAuction)
	t.Run("pegged orders amend to move reference", testPeggedOrderAmendToMoveReference)
	t.Run("pegged orders are removed when expired", testPeggedOrderExpiring)
	t.Run("pegged orders unpark order due to reference becoming valid", testPeggedOrderUnpark)
	t.Run("pegged order cancel a parked order", testPeggedOrderCancelParked)
	t.Run("pegged order reprice when no limit orders", testPeggedOrderRepriceCrashWhenNoLimitOrders)
	t.Run("pegged orders cancelall", testPeggedOrderParkCancelAll)
	t.Run("pegged orders expiring 2", testPeggedOrderExpiring2)
	t.Run("pegged orders test for events produced", testPeggedOrderOutputMessages)
	t.Run("pegged orders test for events produced 2", testPeggedOrderOutputMessages2)
}

func testPeggedOrderRepriceCrashWhenNoLimitOrders(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")

	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party2", 5, 9000)

	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party2", 10, 0)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 10)
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 5, 9000)
}

func testPeggedOrderUnpark(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}

	// Create a single buy order to give this party a valid position
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 5, 11)

	// Add a pegged order which will park due to missing reference price
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 10)
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))

	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)
	// Send a new order to set the BEST_ASK price and force the parked order to unpark
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party2", 5, 15)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrderAmendToMoveReference(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	// Place 2 orders to create valid reference prices
	bestBidOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 90)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 110)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 10)
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Amend best bid price
	amendOrder(t, tm, "party1", bestBidOrder, 0, 88, types.OrderTimeInForceUnspecified, 0, true)
	amendOrder(t, tm, "party1", bestBidOrder, 0, 86, types.OrderTimeInForceUnspecified, 0, true)
}

func testPeggedOrderFilledOrder(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 80),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 120),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 90)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 110)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Place a sell MARKET order to fill the buy orders
	sendOrder(t, tm, &now, types.OrderTypeMarket, types.OrderTimeInForceIOC, 0, types.SideSell, "party2", 2, 0)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testParkedOrdersAreUnparkedWhenPossible(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 5)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 100)

	// Place a valid pegged order which will be parked because it cannot be repriced
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 1)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 10)
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())

	// Send a higher buy price order to move the BEST BID price up
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 50)

	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testPeggedOrdersLeavingAuction(t *testing.T) {
	now := time.Unix(10, 0)
	auctionClose := now.Add(101 * time.Second)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 100,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")
	addAccount(t, tm, "party3")
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Move into auction
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 100*time.Second)

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 90)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 100)
	// place 2 more orders that will result in a mark price being set
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party2", 1, 95)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party3", 1, 95)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 10)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.OrderStatusParked)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	// During an auction all pegged orders are parked so we don't add them to the list
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())

	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// Update the time to force the auction to end
	tm.now = auctionClose
	tm.market.OnTick(ctx, auctionClose)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrdersEnteringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 100,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")
	addAccount(t, tm, "party3")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 100*time.Second)
	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 90)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 100)
	// place 2 more orders that will result in a mark price being set
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party2", 1, 95)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party3", 1, 95)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 10)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.OrderStatusParked)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())

	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
}

func testPeggedOrderAddWithNoMarketPrice(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	// Place a valid pegged order which will be parked
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.OrderStatusParked)
	assert.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testPeggedOrderAdd(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(25000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 100)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 102)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	assert.NotNil(t, confirmation)
	assert.Equal(t, types.OrderStatusActive, confirmation.Order.Status)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())

	assert.True(t, order.Price.EQ(num.NewUint(500047)), order.Price.String())
}

func testPeggedOrderWithReprice(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(25000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 90)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 110)

	md := tm.market.GetMarketData()
	assert.True(t, md.MidPrice.EQ(num.NewUint(500045)), md.MidPrice.String())
	// Place a valid pegged order which will be added to the order book
	// This order will cause the MID price to move and thus a reprice multiple times until it settles
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Check to make sure the existing pegged order is repriced correctly
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())

	// TODO need to find a way to validate details of the amended order
}

func testPeggedOrderParkWhenInAuction(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.OrderStatusParked)
	assert.NoError(t, err)
}

func testPeggedOrderUnparkAfterLeavingAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.OrderStatusParked)
	assert.NoError(t, err)

	buy := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 90)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &buy)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	require.NotNil(t, buy)
	sell := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 110)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &sell)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	tm.market.LeaveAuctionWithIDGen(ctx, closingAt, newTestIDGenerator())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrderTypes(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Not MARKET
	order.Type = types.OrderTypeMarket
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)
}

func testPeggedOrderCancelParked(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")

	// Pegged order will be parked as no reference prices
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)
}

func testPeggedOrderTIFs(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)

	// Only allowed GTC
	order.Type = types.OrderTypeLimit
	order.TimeInForce = types.OrderTimeInForceGTC
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// and GTT
	order.TimeInForce = types.OrderTimeInForceGTT
	order.ExpiresAt = now.UnixNano() + 1000000000
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// but not IOC
	order.ExpiresAt = 0
	order.TimeInForce = types.OrderTimeInForceIOC
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	// or FOK
	order.TimeInForce = types.OrderTimeInForceFOK
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)
}

func testPeggedOrderBuys(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")

	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 100)

	// BEST BID peg must be >= 0
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 0)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// MID peg must be > 0
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 0)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// BEST ASK peg not allowed
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 3)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 3)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 0)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)
}

func testPeggedOrderSells(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")

	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 100)

	// BEST BID peg not allowed
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 0)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	// MID peg must be > 0
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 0)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 3)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// BEST ASK peg must be >= 0
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 3)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 0)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
}

func testPeggedOrderParkWhenPriceBelowZero(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	for _, acc := range []string{"buyer", "seller", "pegged"} {
		addAccount(t, tm, acc)
	}

	buy := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "buyer", 10, 4)
	_, err := tm.market.SubmitOrder(ctx, &buy)
	require.NoError(t, err)

	sell := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "seller", 10, 8)
	_, err = tm.market.SubmitOrder(ctx, &sell)
	require.NoError(t, err)

	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "pegged", 10, 4)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 10)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.Equal(t,
		types.OrderStatusParked.String(),
		confirmation.Order.Status.String(), "When pegged price below zero (MIDPRICE - OFFSET) <= 0")
}

func testPeggedOrderParkWhenPriceRepricesBelowZero(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	for _, acc := range []string{"buyer", "seller", "pegged"} {
		addAccount(t, tm, acc)
	}

	buy := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "buyer", 10, 4)
	_, err := tm.market.SubmitOrder(ctx, &buy)
	require.NoError(t, err)

	sell := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "seller", 10, 8)
	_, err = tm.market.SubmitOrder(ctx, &sell)
	require.NoError(t, err)

	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "pegged", 10, 4)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 5)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	amendOrder(t, tm, "buyer", buy.ID, 0, 1, types.OrderTimeInForceUnspecified, 0, true)

	assert.Equal(t, types.OrderStatusParked.String(), confirmation.Order.Status.String())
}

/*func TestPeggedOrderCrash(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	for _, acc := range []string{"user1", "user2", "user3", "user4", "user5", "user6", "user7"} {
		addAccount(tm, acc)
	}

	// Set up the best bid/ask values
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user1", 5, 10500)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user2", 20, 11000)

	// Pegged order buy 35 MID -500
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user3", 35, 0)
	order.PeggedOrder = getPeggedOrder(types.PeggedReference_PEGGED_REFERENCE_MID,500)
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Pegged order buy 16 BEST_BID -2000
	order2 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user4", 16, 0)
	order2.PeggedOrder = getPeggedOrder(types.PeggedReference_PEGGED_REFERENCE_BEST_BID,2000)
	_, err = tm.market.SubmitOrder(ctx, &order2)
	require.NoError(t, err)

	// Pegged order sell 19 BEST_ASK 3000
	order3 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user5", 19, 0)
	order3.PeggedOrder = getPeggedOrder(types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,3000)
	_, err = tm.market.SubmitOrder(ctx, &order3)
	require.NoError(t, err)

	// Buy 25 @ 10000
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user6", 25, 10000)

	// Sell 25 @ 10250
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user7", 25, 10250)
}*/

func testPeggedOrderParkCancelAll(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "user")

	// Send one normal order
	limitOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user", 10, 100)
	require.NotEmpty(t, limitOrder)

	// Send one pegged order that is live
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user", 10, 0)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 5)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)

	// Send one pegged order that is parked
	order2 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user", 10, 0)
	order2.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 5)
	confirmation2, err := tm.market.SubmitOrder(ctx, &order2)
	require.NoError(t, err)
	assert.NotNil(t, confirmation2)

	cancelConf, err := tm.market.CancelAllOrders(ctx, "user")
	require.NoError(t, err)
	require.NotNil(t, cancelConf)
	assert.Equal(t, 3, len(cancelConf))
}

func testPeggedOrderExpiring2(t *testing.T) {
	now := time.Unix(10, 0)
	expire := now.Add(time.Second * 100)
	afterexpire := now.Add(time.Second * 200)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "user")

	// Send one normal expiring order
	limitOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTT, expire.UnixNano(), types.SideBuy, "user", 10, 100)
	require.NotEmpty(t, limitOrder)

	// Amend the expiry time
	amendOrder(t, tm, "user", limitOrder, 0, 0, types.OrderTimeInForceUnspecified, now.UnixNano(), true)

	// Send one pegged order that will be parked
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTT, expire.UnixNano(), types.SideBuy, "user", 10, 0)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 5)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)

	// Send one pegged order that will also be parked (after additing liquidity monitoring to market all orders will be parked unless both best_bid and best_offer exist)
	order2 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTT, expire.UnixNano(), types.SideBuy, "user", 10, 0)
	order2.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 5)
	confirmation, err = tm.market.SubmitOrder(ctx, &order2)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	tm.now = tm.now.Add(time.Second)
	tm.market.OnTick(ctx, tm.now)

	assert.Equal(t, 2, tm.market.GetParkedOrderCount())
	assert.Equal(t, 2, tm.market.GetPeggedOrderCount())

	// Move the time forward
	tm.events = nil
	tm.market.OnTick(ctx, afterexpire)
	t.Run("3 orders expired", func(t *testing.T) {
		// First collect all the orders events
		orders := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().Status == types.OrderStatusExpired {
					orders = append(orders, mustOrderFromProto(evt.Order()))
				}
			}
		}
		require.Len(t, orders, 2)
		// Check that we have no pegged orders
		assert.Equal(t, 0, tm.market.GetParkedOrderCount())
		assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	})
}

func testPeggedOrderOutputMessages(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "user1")
	addAccount(t, tm, "user2")
	addAccount(t, tm, "user3")
	addAccount(t, tm, "user4")
	addAccount(t, tm, "user5")
	addAccount(t, tm, "user6")
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 100000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "user1", 10, 0)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 10)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, int(11), int(tm.orderEventCount))

	order2 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "user2", 10, 0)
	order2.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 15)
	confirmation2, err := tm.market.SubmitOrder(ctx, &order2)
	require.NoError(t, err)
	assert.NotNil(t, confirmation2)
	assert.Equal(t, int(14), int(tm.orderEventCount))

	order3 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user3", 10, 0)
	order3.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 10)
	confirmation3, err := tm.market.SubmitOrder(ctx, &order3)
	require.NoError(t, err)
	assert.NotNil(t, confirmation3)
	assert.Equal(t, int(17), int(tm.orderEventCount))

	order4 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user4", 10, 0)
	order4.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 10)
	confirmation4, err := tm.market.SubmitOrder(ctx, &order4)
	require.NoError(t, err)
	assert.NotNil(t, confirmation4)
	assert.Equal(t, int(20), int(tm.orderEventCount))

	limitOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "user5", 1000, 120)
	require.NotEmpty(t, limitOrder)
	// force reference price  checks result in more events
	// assert.Equal(t, int(28), int(tm.orderEventCount))
	assert.Equal(t, int(30), int(tm.orderEventCount))

	limitOrder2 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user6", 1000, 80)
	require.NotEmpty(t, limitOrder2)
	// assert.Equal(t, int(35), int(tm.orderEventCount))
	assert.Equal(t, int(39), int(tm.orderEventCount))
}

func testPeggedOrderOutputMessages2(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "user1")
	addAccount(t, tm, "user2")
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 100000)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	// Create a pegged parked order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user1", 10, 0)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusParked, confirmation.Order.Status)
	assert.NotNil(t, confirmation)
	assert.Equal(t, int(11), int(tm.orderEventCount))

	// Send normal order to unpark the pegged order
	limitOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user2", 1000, 120)
	require.NotEmpty(t, limitOrder)
	assert.Equal(t, int(17), int(tm.orderEventCount))
	assert.Equal(t, types.OrderStatusActive, confirmation.Order.Status)

	// Cancel the normal order to park the pegged order
	tm.market.CancelOrder(ctx, "user2", limitOrder, vgcrypto.RandomHash())
	require.Equal(t, types.OrderStatusParked, confirmation.Order.Status)
	assert.Equal(t, int(23), int(tm.orderEventCount))

	// Send a new normal order to unpark the pegged order
	limitOrder2 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "user2", 1000, 80)
	require.NotEmpty(t, limitOrder2)
	require.Equal(t, types.OrderStatusActive, confirmation.Order.Status)
	assert.Equal(t, int(29), int(tm.orderEventCount))

	// Fill that order to park the pegged order
	limitOrder3 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "user1", 1000, 80)
	require.NotEmpty(t, limitOrder3)
	require.Equal(t, types.OrderStatusActive, confirmation.Order.Status)
	assert.Equal(t, int(36), int(tm.orderEventCount))
	// assert.Equal(t, int(34), int(tm.orderEventCount))
}

func testPeggedOrderRepricing(t *testing.T) {
	// Create the market
	now := time.Unix(10, 0)

	var (
		buyPrice  uint64 = 90
		sellPrice uint64 = 110
		midPrice         = (sellPrice + buyPrice) / 2
	)

	tests := []struct {
		reference      types.PeggedReference
		side           types.Side
		offset         uint64
		expectedPrice  *num.Uint
		expectingError string
	}{
		{
			reference:     types.PeggedReferenceBestBid,
			side:          types.SideBuy,
			offset:        3,
			expectedPrice: num.NewUint(buyPrice - 3),
		},
		{
			reference:      types.PeggedReferenceBestAsk,
			side:           types.SideBuy,
			offset:         0,
			expectedPrice:  num.UintZero(),
			expectingError: "offset must be greater than zero",
		},
		{
			reference:     types.PeggedReferenceMid,
			side:          types.SideBuy,
			offset:        5,
			expectedPrice: num.NewUint(midPrice - 5),
		},
		{
			reference:     types.PeggedReferenceMid,
			side:          types.SideSell,
			offset:        5,
			expectedPrice: num.NewUint(midPrice + 5),
		},
		{
			reference:     types.PeggedReferenceBestAsk,
			side:          types.SideSell,
			offset:        5,
			expectedPrice: num.NewUint(sellPrice + 5),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			// Create market
			tm := getTestMarket(t, now, nil, &types.AuctionDuration{
				Duration: 1,
			})
			ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
			tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

			auxParty, auxParty2 := "auxParty", "auxParty2"
			addAccount(t, tm, "party1")
			addAccount(t, tm, auxParty)
			addAccount(t, tm, auxParty2)
			addAccountWithAmount(tm, "lpprov", 10000000)

			auxOrders := []*types.Order{
				getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 80),
				getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 120),
				getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
				getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
			}
			for _, o := range auxOrders {
				conf, err := tm.market.SubmitOrder(ctx, o)
				require.NoError(t, err)
				require.NotNil(t, conf)
			}
			lp := &types.LiquidityProvisionSubmission{
				MarketID:         tm.market.GetID(),
				CommitmentAmount: num.NewUint(25000),
				Fee:              num.DecimalFromFloat(0.01),
				Sells: []*types.LiquidityOrder{
					newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
				},
				Buys: []*types.LiquidityOrder{
					newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
				},
			}
			require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
			// leave auction
			now := now.Add(2 * time.Second)
			tm.now = now
			tm.market.OnTick(ctx, now)

			// Create buy and sell orders
			sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, buyPrice)
			sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, sellPrice)

			// Create pegged order
			order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, test.side, "party1", 10, 0)
			order.PeggedOrder = newPeggedOrder(test.reference, test.offset)
			conf, err := tm.market.SubmitOrder(context.Background(), &order)
			if msg := test.expectingError; msg != "" {
				require.Error(t, err, msg)
			} else {
				require.NoError(t, err)
				assert.True(t, test.expectedPrice.EQ(conf.Order.Price), conf.Order.Price)
			}
		})
	}
}

func testPeggedOrderExpiring(t *testing.T) {
	// Create the market
	now := time.Unix(10, 0)

	tm := getTestMarket(t, now, nil, nil)
	addAccount(t, tm, "party")

	// Create buy and sell orders
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party", 1, 100)
	sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party", 1, 200)

	// let's create N orders with different expiration time
	expirations := []struct {
		party      string
		expiration time.Time
	}{
		{"party-10", now.Add(10 * time.Minute)},
		{"party-20", now.Add(20 * time.Minute)},
		{"party-30", now.Add(30 * time.Minute)},
	}
	for _, test := range expirations {
		addAccount(t, tm, test.party)

		order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTT, 0, types.SideBuy, test.party, 10, 150)
		order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 10)
		order.ExpiresAt = test.expiration.UnixNano()
		_, err := tm.market.SubmitOrder(context.Background(), &order)
		require.NoError(t, err)
	}
	assert.Equal(t, len(expirations), tm.market.GetPeggedOrderCount())

	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.events = nil
	tm.market.OnTick(ctx, now.Add(25*time.Minute))
	t.Run("2 orders expired", func(t *testing.T) {
		// First collect all the orders events
		orders := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().Status == types.OrderStatusExpired {
					orders = append(orders, mustOrderFromProto(evt.Order()))
				}
			}
		}

		assert.Equal(t, 2, len(orders))
		assert.Equal(t, 1, tm.market.GetPeggedOrderCount(), "1 order should still be in the market")
	})
}

func TestPeggedOrdersAmends(t *testing.T) {
	t.Run("pegged orders amend an order that is parked but becomes live ", testPeggedOrderAmendParkedToLive)
	t.Run("pegged orders amend an order that is parked and remains parked", testPeggedOrderAmendParkedStayParked)
	t.Run("pegged orders amend an order that is live but becomes parked", testPeggedOrderAmendForcesPark)
	t.Run("pegged orders amend an order while in auction", testPeggedOrderAmendDuringAuction)
	t.Run("pegged orders amend an orders pegged reference", testPeggedOrderAmendReference)
	t.Run("pegged orders amend an orders pegged reference during an auction", testPeggedOrderAmendReferenceInAuction)
	t.Run("pegged orders amend multiple fields at once", testPeggedOrderAmendMultiple)
	t.Run("pegged orders amend multiple fields at once in an auction", testPeggedOrderAmendMultipleInAuction)
	t.Run("pegged orders delete an order that has lost time priority", testPeggedOrderCanDeleteAfterLostPriority)
	t.Run("pegged orders validate mid price values", testPeggedOrderMidPriceCalc)
}

// We had a case where things crashed when the orders on the same price level were not sorted
// in createdAt order. Test this by creating a pegged order and repricing to make it lose it's time order.
func testPeggedOrderCanDeleteAfterLostPriority(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)

	addAccount(t, tm, "party1")

	// Place trades so we have a valid BEST_BID
	buyOrder1 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 100)
	require.NotNil(t, buyOrder1)

	// Place the pegged order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 10)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Place a normal limit order behind the pegged order
	buyOrder2 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 90)
	require.NotNil(t, buyOrder2)

	// Amend first order to move pegged
	amendOrder(t, tm, "party1", buyOrder1, 0, 101, types.OrderTimeInForceUnspecified, 0, true)
	// Amend again to make the pegged order reprice behind the second limit order
	amendOrder(t, tm, "party1", buyOrder1, 0, 100, types.OrderTimeInForceUnspecified, 0, true)

	// Try to delete the pegged order
	cancelconf, _ := tm.market.CancelOrder(context.TODO(), "party1", order.ID, vgcrypto.RandomHash())
	assert.NotNil(t, cancelconf)
	assert.Equal(t, types.OrderStatusCancelled, cancelconf.Order.Status)
}

// If we amend an order that is parked and not in auction we need to see if the amendment has caused the
// order to be unparkable. If so we will have to put it back on the live book.
func testPeggedOrderAmendParkedToLive(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 10),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 10),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
		assert.NotNil(t, conf)
	}

	// Place the pegged order which will be parked
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 20)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we can reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	off := num.NewUint(5)
	amend.PeggedOffset = off
	amended, err := tm.market.AmendOrder(ctx, amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.Equal(t, off, amended.Order.PeggedOrder.Offset)
	assert.NoError(t, err)

	// Check we should have no parked orders
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

// Amend a parked order but the order remains parked.
func testPeggedOrderAmendParkedStayParked(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will be parked
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 20)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we can reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	off := num.NewUint(15)
	amend.PeggedOffset = off
	amended, err := tm.market.AmendOrder(ctx, amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.Equal(t, off, amended.Order.PeggedOrder.Offset)
	assert.NoError(t, err)

	// Check we should have no parked orders
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

// Take a valid live order and force it to be parked by amending it.
func testPeggedOrderAmendForcesPark(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	amend.PeggedOffset = num.NewUint(15)
	amended, err := tm.market.AmendOrder(ctx, amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.NoError(t, err)

	// Order should be parked
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.OrderStatusParked, amended.Order.Status)
}

func testPeggedOrderAmendDuringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(ctx)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	amend.PeggedOffset = num.NewUint(5)
	amended, err := tm.market.AmendOrder(context.Background(), amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.OrderStatusParked, amended.Order.Status)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testPeggedOrderAmendReference(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 10),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 10),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	amend.PeggedReference = types.PeggedReferenceMid
	amended, err := tm.market.AmendOrder(context.Background(), amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.OrderStatusActive, amended.Order.Status)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReferenceMid, amended.Order.PeggedOrder.Reference)
}

func testPeggedOrderAmendReferenceInAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(ctx)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	amend.PeggedReference = types.PeggedReferenceMid
	amended, err := tm.market.AmendOrder(context.Background(), amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.OrderStatusParked, amended.Order.Status)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReferenceMid, amended.Order.PeggedOrder.Reference)
}

func testPeggedOrderAmendMultipleInAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")

	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(ctx)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	amend.PeggedReference = types.PeggedReferenceMid
	amend.TimeInForce = types.OrderTimeInForceGTT
	exp := int64(20000000000)
	amend.ExpiresAt = &exp
	amended, err := tm.market.AmendOrder(ctx, amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.OrderStatusParked, amended.Order.Status)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReferenceMid, amended.Order.PeggedOrder.Reference)
	assert.Equal(t, types.OrderTimeInForceGTT, amended.Order.TimeInForce)
}

func testPeggedOrderAmendMultiple(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 10),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 10),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 11)
	require.NotNil(t, sellOrder)
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))

	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 3)
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), confirmation.Order.ID, 0, 0, types.OrderTimeInForceUnspecified, 0)
	amend.PeggedReference = types.PeggedReferenceMid
	amend.TimeInForce = types.OrderTimeInForceGTT
	exp := int64(20000000000)
	amend.ExpiresAt = &exp
	amended, err := tm.market.AmendOrder(context.Background(), amend, "party1", vgcrypto.RandomHash())
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.OrderStatusActive, amended.Order.Status)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReferenceMid, amended.Order.PeggedOrder.Reference)
	assert.Equal(t, types.OrderTimeInForceGTT, amended.Order.TimeInForce)
}

func testPeggedOrderMidPriceCalc(t *testing.T) {
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, "party1")
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 90)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1, 110)
	require.NotNil(t, sellOrder)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	now = now.Add(2 * time.Second)
	tm.now = now
	tm.market.OnTick(ctx, now)

	// Place the pegged orders
	order1 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 10, 10)
	order1.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 20)
	confirmation1, err := tm.market.SubmitOrder(context.Background(), &order1)
	require.NotNil(t, confirmation1)
	assert.NoError(t, err)
	assert.True(t, confirmation1.Order.Price.EQ(num.NewUint(80)))

	order2 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 10, 10)
	order2.PeggedOrder = newPeggedOrder(types.PeggedReferenceMid, 20)
	confirmation2, err := tm.market.SubmitOrder(context.Background(), &order2)
	require.NotNil(t, confirmation2)
	assert.NoError(t, err)
	assert.True(t, confirmation2.Order.Price.EQ(num.NewUint(120)))

	// Make the mid price wonky (needs rounding)
	buyOrder2 := sendOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1, 91)
	require.NotNil(t, buyOrder2)

	// Check the pegged orders have reprices properly
	assert.True(t, confirmation1.Order.Price.EQ(num.NewUint(81)))  // Buy price gets rounded up
	assert.True(t, confirmation2.Order.Price.EQ(num.NewUint(120))) // Sell price gets rounded down
}

func TestPeggedOrderUnparkAfterLeavingAuctionWithNoFunds2772(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, "party1")
	addAccount(t, tm, "party2")
	addAccount(t, tm, "party3")
	addAccount(t, tm, "party4")
	auxParty := "auxParty"
	addAccount(t, tm, auxParty)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 100000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	buyPeggedOrder := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party1", 1000000000000, 0)
	buyPeggedOrder.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 10)
	confirmation1, err := tm.market.SubmitOrder(ctx, &buyPeggedOrder)
	assert.NotNil(t, confirmation1)
	assert.Equal(t, confirmation1.Order.Status, types.OrderStatusParked)
	assert.NoError(t, err)

	sellPeggedOrder := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party1", 1000000000000, 0)
	sellPeggedOrder.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestAsk, 10)
	confirmation2, err := tm.market.SubmitOrder(ctx, &sellPeggedOrder)
	assert.NotNil(t, confirmation2)
	assert.Equal(t, confirmation2.Order.Status, types.OrderStatusParked)
	assert.NoError(t, err)

	sellOrder1 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party2", 4, 2000)
	confirmation3, err := tm.market.SubmitOrder(ctx, &sellOrder1)
	assert.NotNil(t, confirmation3)
	assert.NoError(t, err)

	tm.market.LeaveAuctionWithIDGen(ctx, closingAt, newTestIDGenerator())

	buyOrder1 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideBuy, "party3", 100, 6500)
	confirmation4, err := tm.market.SubmitOrder(ctx, &buyOrder1)
	assert.NotNil(t, confirmation4)
	assert.NoError(t, err)

	sellOrder2 := getOrder(t, tm, &now, types.OrderTypeLimit, types.OrderTimeInForceGTC, 0, types.SideSell, "party4", 20, 7000)
	confirmation5, err := tm.market.SubmitOrder(ctx, &sellOrder2)
	assert.NotNil(t, confirmation5)
	assert.NoError(t, err)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

// test for issue 787,
// segv when an GTT order is cancelled, then expires.
func TestOrderBookSimple_CancelGTTOrderThenRunExpiration(t *testing.T) {
	now := time.Unix(5, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	defer tm.ctrl.Finish()

	addAccount(t, tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideBuy, "aaa", 10, 100)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	cncl, err := tm.market.CancelOrder(ctx, o1.Party, o1.ID, vgcrypto.RandomHash())
	require.NoError(t, err)
	require.NotNil(t, cncl)
	assert.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())

	tm.market.OnTick(ctx, now.Add(10*time.Second))
	t.Run("no orders expired", func(t *testing.T) {
		// First collect all the orders events
		orders := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().Status == types.OrderStatusExpired {
					orders = append(orders, mustOrderFromProto(evt.Order()))
				}
			}
		}
		require.Len(t, orders, 0)
	})
	assert.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())
}

func TestGTTExpiredNotFilled(t *testing.T) {
	now := time.Unix(5, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	defer tm.ctrl.Finish()

	addAccount(t, tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideSell, "aaa", 10, 100)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	// then remove expired, set 1 sec after order exp time.
	tm.events = nil
	tm.market.OnTick(ctx, now.Add(10*time.Second))
	t.Run("1 order expired", func(t *testing.T) {
		// First collect all the orders events
		orders := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().Status == types.OrderStatusExpired {
					orders = append(orders, mustOrderFromProto(evt.Order()))
				}
			}
		}
		assert.Len(t, orders, 1)
	})
}

func TestGTTExpiredPartiallyFilled(t *testing.T) {
	now := time.Unix(5, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	defer tm.ctrl.Finish()
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 10000000)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideSell, auxParty, 1, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideBuy, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(5000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 1, 10),
		},
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	// leave auction
	tm.now = tm.now.Add(2 * time.Second)
	tm.market.OnTick(ctx, tm.now)
	addAccount(t, tm, "aaa")
	addAccount(t, tm, "bbb")

	// We probably don't need these orders anymore, but they don't do any harm
	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))

	// place expiring order
	o1 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideSell, "aaa", 10, 100)
	o1.ExpiresAt = tm.now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	// add matching order
	o2 := getMarketOrder(tm, tm.now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order02", types.SideBuy, "bbb", 1, 100)
	o2.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NoError(t, err)
	require.NotNil(t, o2conf)

	// then remove expired, set 1 sec after order exp time.
	tm.events = nil
	tm.market.OnTick(ctx, tm.now.Add(10*time.Second))
	t.Run("1 order expired - the other matched", func(t *testing.T) {
		// First collect all the orders events
		orders := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().Status == types.OrderStatusExpired {
					orders = append(orders, mustOrderFromProto(evt.Order()))
				} else {
					fmt.Printf("%s\n", evt.Order().Status)
				}
			}
		}
		assert.Len(t, orders, 1)
	})
}

func TestOrderBook_RemoveExpiredOrders(t *testing.T) {
	now := time.Unix(5, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	defer tm.ctrl.Finish()

	addAccount(t, tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	someTimeLater := now.Add(100 * time.Second)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideSell, "aaa", 1, 1)
	o1.ExpiresAt = someTimeLater.UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order02", types.SideSell, "aaa", 99, 3298)
	o2.ExpiresAt = someTimeLater.UnixNano() + 1
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NoError(t, err)
	require.NotNil(t, o2conf)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order03", types.SideSell, "aaa", 19, 771)
	o3.ExpiresAt = someTimeLater.UnixNano()
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NoError(t, err)
	require.NotNil(t, o3conf)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "aaa", 7, 1000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NoError(t, err)
	require.NotNil(t, o4conf)

	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order05", types.SideSell, "aaa", 99999, 199)
	o5.ExpiresAt = someTimeLater.UnixNano()
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NoError(t, err)
	require.NotNil(t, o5conf)

	o6 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order06", types.SideSell, "aaa", 100, 100)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NoError(t, err)
	require.NotNil(t, o6conf)

	o7 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order07", types.SideSell, "aaa", 9999, 41)
	o7.ExpiresAt = someTimeLater.UnixNano() + 9999
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NoError(t, err)
	require.NotNil(t, o7conf)

	o8 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order08", types.SideSell, "aaa", 1, 1)
	o8.ExpiresAt = someTimeLater.UnixNano() - 9999
	o8conf, err := tm.market.SubmitOrder(ctx, o8)
	require.NoError(t, err)
	require.NotNil(t, o8conf)

	o9 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order09", types.SideSell, "aaa", 12, 65)
	o9conf, err := tm.market.SubmitOrder(ctx, o9)
	require.NoError(t, err)
	require.NotNil(t, o9conf)

	o10 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order10", types.SideSell, "aaa", 1, 1)
	o10.ExpiresAt = someTimeLater.UnixNano() - 1
	o10conf, err := tm.market.SubmitOrder(ctx, o10)
	require.NoError(t, err)
	require.NotNil(t, o10conf)

	tm.events = nil
	tm.market.OnTick(ctx, someTimeLater)
	t.Run("5 orders expired", func(t *testing.T) {
		// First collect all the orders events
		orders := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().Status == types.OrderStatusExpired {
					orders = append(orders, mustOrderFromProto(evt.Order()))
				}
			}
		}
		require.Len(t, orders, 5)
	})
}

func Test2965EnsureLPOrdersAreNotCancelleableWithCancelAll(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	mktCfg := getMarket(defaultPriceMonitorSettings, &types.AuctionDuration{
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

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("party-0", 1000000).
		WithAccountAndAmount("party-1", 1000000).
		WithAccountAndAmount("party-2", 10000000000).
		// provide stake as well but will cancel
		WithAccountAndAmount("party-2-bis", 10000000000).
		WithAccountAndAmount("party-3", 1000000).
		WithAccountAndAmount("party-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(1.0))
	tm.now = now
	tm.market.OnTick(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.OrderTimeInForce
		pegRef    types.PeggedReference
		pegOffset *num.Uint
	}{
		{"party-4", 1, types.SideBuy, types.OrderTimeInForceGTC, types.PeggedReferenceBestBid, num.NewUint(2000)},
		{"party-3", 1, types.SideSell, types.OrderTimeInForceGTC, types.PeggedReferenceBestAsk, num.NewUint(1000)},
	}
	partyA, partyB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.OrderTypeLimit,
	}
	orders := []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       num.UintZero().Sub(num.NewUint(5500), partyA.pegOffset), // 3500
			Side:        types.SideBuy,
			Party:       "party-0",
			TimeInForce: types.OrderTimeInForceGFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       num.UintZero().Sub(num.NewUint(5000), partyB.pegOffset), // 4000
			Side:        types.SideSell,
			Party:       "party-1",
			TimeInForce: types.OrderTimeInForceGFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       num.NewUint(5500),
			Side:        types.SideBuy,
			Party:       "party-2",
			TimeInForce: types.OrderTimeInForceGFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       num.NewUint(5000),
			Side:        types.SideSell,
			Party:       "party-2",
			TimeInForce: types.OrderTimeInForceGTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       num.NewUint(3500),
			Side:        types.SideBuy,
			Party:       "party-0",
			TimeInForce: types.OrderTimeInForceGTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       num.NewUint(8500),
			Side:        types.SideBuy,
			Party:       "party-0",
			TimeInForce: types.OrderTimeInForceGTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			Party:       partyA.id,
			Side:        partyA.side,
			Size:        partyA.size,
			Remaining:   partyA.size,
			TimeInForce: partyA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: partyA.pegRef,
				Offset:    partyA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			Party:       partyB.id,
			Side:        partyB.side,
			Size:        partyB.size,
			Remaining:   partyB.size,
			TimeInForce: partyB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: partyB.pegRef,
				Offset:    partyB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(2000000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "THIS-IS-LP",
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestAsk, 2, 10),
			newLiquidityOrder(types.PeggedReferenceBestAsk, 1, 13),
		},
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceBestBid, 1, 10),
			newLiquidityOrder(types.PeggedReferenceMid, 15, 13),
		},
	}

	// Leave the auction
	tm.now = now.Add(10001 * time.Second)
	tm.market.OnTick(ctx, tm.now)

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "party-2", vgcrypto.RandomHash()))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	tm.now = now.Add(10011 * time.Second)
	tm.market.OnTick(ctx, tm.now)

	newOrder := tpl.New(types.Order{
		MarketID:    tm.market.GetID(),
		Size:        20,
		Remaining:   20,
		Price:       num.NewUint(10250),
		Side:        types.SideSell,
		Party:       "party-2",
		TimeInForce: types.OrderTimeInForceGTC,
	})

	tm.events = nil
	cnf, err := tm.market.SubmitOrder(ctx, newOrder)
	assert.NoError(t, err)
	assert.Len(t, cnf.Trades, 0)

	// now we cancel all orders, but should get only 1 cancellation
	// and the ID should be newOrder
	tm.events = nil
	cancelCnf, err := tm.market.CancelAllOrders(ctx, "party-2")
	assert.NoError(t, err)
	assert.Len(t, cancelCnf, 2)

	t.Run("ExpectedOrderCancelled", func(t *testing.T) {
		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedIds := map[string]bool{
			newOrder.ID:  false,
			orders[3].ID: false,
		}

		require.Len(t, cancelCnf, len(expectedIds))

		for _, o := range cancelCnf {
			_, ok := expectedIds[o.Order.ID]
			if !ok {
				t.Errorf("unexpected cancelled order: %v", o.Order.ID)
			}
			expectedIds[o.Order.ID] = true
		}

		for id, ok := range expectedIds {
			if !ok {
				t.Errorf("expected order to be cancelled was not cancelled: %v", id)
			}
		}
	})
}

func TestMissingLP(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	party3 := "party3"
	auxParty, auxParty2 := "auxParty", "auxParty2"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)
	addAccount(t, tm, party3)
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))
	// ensure auction durations are 1 second
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, vgcrypto.RandomHash(), types.SideBuy, auxParty, 1, 800000)
	conf, err := tm.market.SubmitOrder(ctx, alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, vgcrypto.RandomHash(), types.SideSell, auxParty, 1, 8200000)
	conf, err = tm.market.SubmitOrder(ctx, alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)
	// create orders so we can leave opening auction
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, vgcrypto.RandomHash(), types.SideBuy, auxParty, 1, 3500000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, vgcrypto.RandomHash(), types.SideSell, auxParty2, 1, 3500000),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}
	now = now.Add(time.Second * 2) // opening auction is 1 second, move time ahead by 2 seconds so we leave auction
	tm.now = now
	tm.market.OnTick(ctx, now)

	// Here we are in auction
	assert.True(t, tm.mas.InAuction())

	// Send LP order

	lps := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(10000),
		Fee:              num.DecimalFromFloat(0.01),
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: num.NewUint(10000)},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReferenceMid, Proportion: 1, Offset: num.NewUint(10000)},
		},
	}

	tm.market.SubmitLiquidityProvision(ctx, lps, party1, vgcrypto.RandomHash())

	// Check we have 2 orders on each side of the book (4 in total)
	assert.EqualValues(t, 4, tm.market.GetOrdersOnBookCount())

	// Send in a limit order to move the BEST_BID and MID price
	newBestBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, vgcrypto.RandomHash(), types.SideBuy, auxParty, 1, 810000)
	conf, err = tm.market.SubmitOrder(ctx, newBestBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	// Check we have 5 orders in total
	assert.EqualValues(t, 5, tm.market.GetOrdersOnBookCount())
	now = now.Add(time.Second * 2) // opening auction is 1 second, move time ahead by 2 seconds so we leave auction
	tm.now = now
	tm.market.OnTick(ctx, now)
}
