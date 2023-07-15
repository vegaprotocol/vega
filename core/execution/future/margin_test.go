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

func TestMargins(t *testing.T) {
	party1, party2, party3 := "party1", "party2", "party3"
	now := time.Unix(10, 0)
	tm := getTestMarket2(t, now, nil, &types.AuctionDuration{
		Duration: 1,
		// increase lpRange so that LP orders don't get pushed too close to MID and test can behave as expected
	}, true, 1)
	price := num.NewUint(100)
	size := uint64(100)

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)
	addAccount(t, tm, party3)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 100000)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	// set auction durations to 1 second
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
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
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideBuy, auxParty, 1, price.Uint64()),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideSell, auxParty2, 1, price.Uint64()),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(500),
		Fee:              num.DecimalFromFloat(0.01),
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))

	now = now.Add(2 * time.Second)
	// leave opening auction
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm.now = now
	tm.market.OnTick(ctx, now)
	data := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, data.MarketTradingMode)

	order1 := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid12",
		Side:        types.SideBuy,
		Party:       party2,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       price.Clone(),
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-buy-order",
	}
	order2 := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid123",
		Side:        types.SideSell,
		Party:       party3,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       price.Clone(),
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "party3-buy-order",
	}
	_, err = tm.market.SubmitOrder(context.TODO(), order1)
	assert.NoError(t, err)
	confirmation, err := tm.market.SubmitOrder(context.TODO(), order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirmation.Trades))

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       price.Clone(),
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Create an order to amend
	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderBuy)
	if !assert.NoError(t, err) {
		t.Fatalf("Error: %v", err)
	}
	if !assert.NotNil(t, confirmation) {
		t.Fatal("SubmitOrder confirmation was nil, but no error.")
	}

	orderID := confirmation.Order.ID

	// Amend size up
	amend := &types.OrderAmendment{
		OrderID:   orderID,
		MarketID:  tm.market.GetID(),
		SizeDelta: 10000,
	}
	amendment, err := tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend price and size up to breach margin
	amend.SizeDelta = 1000000000
	amend.Price = num.NewUint(1000000000)
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.Nil(t, amendment)
	assert.Error(t, err)
}

/* Check that a failed new order margin check cannot be got around by amending
 * an existing order to the same values as the failed new order. */
func TestPartialFillMargins(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	party3 := "party3"
	auxParty, auxParty2 := "auxParty", "auxParty2"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
		Duration: 1,
	})

	addAccount(t, tm, party1)
	addAccount(t, tm, party2)
	addAccount(t, tm, party3)
	addAccount(t, tm, auxParty)
	addAccount(t, tm, auxParty2)
	addAccountWithAmount(tm, "lpprov", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), num.DecimalFromFloat(0))
	// ensure auction durations are 1 second
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1000000000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.OrderStatusActive, conf.Order.Status)
	// create orders so we can leave opening auction
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideBuy, auxParty, 1, 10000000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideSell, auxParty2, 1, 10000000),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}
	mktD := tm.market.GetMarketData()
	fmt.Printf("TS: %s\nSS: %s\n", mktD.TargetStake, mktD.SuppliedStake)
	lp := &types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(30000000),
		Fee:              num.DecimalFromFloat(0.01),
	}
	require.NoError(t, tm.market.SubmitLiquidityProvision(context.Background(), lp, "lpprov", vgcrypto.RandomHash()))
	now = now.Add(time.Second * 2) // opening auction is 1 second, move time ahead by 2 seconds so we leave auction
	tm.now = now
	tm.market.OnTick(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()), now)

	// use party 2+3 to set super high mark price
	orderSell1 := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Side:        types.SideSell,
		Party:       party2,
		MarketID:    tm.market.GetID(),
		Size:        1,
		Price:       num.NewUint(10000000),
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   now.UnixNano() + 10000000000,
		Reference:   "party2-sell-order",
	}
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderSell1)
	require.NoError(t, err)
	require.NotNil(t, confirmation)

	// other side of the instant match
	orderBuy1 := &types.Order{
		Type:        types.OrderTypeMarket,
		TimeInForce: types.OrderTimeInForceIOC,
		Side:        types.SideBuy,
		Party:       party3,
		MarketID:    tm.market.GetID(),
		Size:        1,
		Price:       num.UintZero(),
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		Reference:   "party3-buy-order",
	}

	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderBuy1)
	if !assert.NoError(t, err) {
		t.Fatalf("Error: %v", err)
	}
	if !assert.NotNil(t, confirmation) {
		t.Fatal("SubmitOrder confirmation was nil, but no error.")
	}

	// Create a valid smaller order
	orderBuy3 := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTT,
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        1,
		Price:       num.NewUint(2),
		Remaining:   1,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   now.UnixNano() + 10000000000,
		Reference:   "party1-buy-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderBuy3)
	if !assert.NoError(t, err) {
		t.Fatalf("Error: %v", err)
	}
	if !assert.NotNil(t, confirmation) {
		t.Fatal("SubmitOrder confirmation was nil, but no error.")
	}
	orderID := confirmation.Order.ID

	// Attempt to amend it to the same size as the failed new order
	amend := &types.OrderAmendment{
		OrderID:   orderID,
		MarketID:  tm.market.GetID(),
		SizeDelta: 999,
	}
	amendment, err := tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.Nil(t, amendment)
	assert.Error(t, err)
}

// TODO karel - fix this tests
// func TestMarginRequirementSkippedWhenReducingExposure(t *testing.T) {
// 	ctx := context.Background()
// 	party1 := "party1"
// 	party2 := "party2"
// 	party3 := "party3"
// 	party4 := "party4"
// 	auxParty, auxParty2 := "auxParty", "auxParty2"
// 	now := time.Unix(10, 0)
// 	tm := getTestMarket(t, now, nil, &types.AuctionDuration{
// 		Duration: 1,
// 	})

// 	addAccount(t, tm, party1)
// 	addAccountWithAmount(tm, party2, 3000)
// 	addAccountWithAmount(tm, party3, 1990)
// 	addAccountWithAmount(tm, party4, 80)
// 	addAccount(t, tm, auxParty)
// 	addAccount(t, tm, auxParty2)
// 	addAccountWithAmount(tm, "lpprov", 100000000)
// 	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

// 	// Assure liquidity auction won't be triggered
// 	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))
// 	// ensure auction durations are 1 second
// 	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
// 	alwaysOnBid := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnBid", types.SideBuy, auxParty, 1, 500)
// 	conf, err := tm.market.SubmitOrder(ctx, alwaysOnBid)
// 	require.NotNil(t, conf)
// 	require.NoError(t, err)
// 	require.Equal(t, types.OrderStatusActive, conf.Order.Status)

// 	alwaysOnAsk := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "alwaysOnAsk", types.SideSell, auxParty, 1, 1500)
// 	conf, err = tm.market.SubmitOrder(ctx, alwaysOnAsk)
// 	require.NotNil(t, conf)
// 	require.NoError(t, err)
// 	require.Equal(t, types.OrderStatusActive, conf.Order.Status)
// 	// create orders so we can leave opening auction
// 	auxOrders := []*types.Order{
// 		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux1", types.SideBuy, auxParty, 1, 1000),
// 		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "aux2", types.SideSell, auxParty2, 1, 1000),
// 	}
// 	for _, o := range auxOrders {
// 		conf, err := tm.market.SubmitOrder(ctx, o)
// 		require.NotNil(t, conf)
// 		require.NoError(t, err)
// 	}
// 	lp := &types.LiquidityProvisionSubmission{
// 		MarketID:         tm.market.GetID(),
// 		CommitmentAmount: num.NewUint(30000000),
// 		Fee:              num.DecimalFromFloat(0.01),
// 	}
// 	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "lpprov", vgcrypto.RandomHash()))

// 	party4Order := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, party4, types.SideBuy, party4, 1, uint64(500))
// 	_, err = tm.market.SubmitOrder(ctx, party4Order)
// 	require.ErrorContains(t, err, "margin")

// 	now = now.Add(time.Second * 2) // opening auction is 1 second, move time ahead by 2 seconds so we leave auction
// 	tm.now = now
// 	tm.market.OnTick(vegacontext.WithTraceID(ctx, vgcrypto.RandomHash()), now)

// 	posSize := uint64(10)
// 	matchingPrice := uint64(1000)
// 	party2Order := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, party2, types.SideSell, party2, posSize, matchingPrice)
// 	confirmation, err := tm.market.SubmitOrder(ctx, party2Order)
// 	require.NoError(t, err)
// 	require.NotNil(t, confirmation)

// 	party3Order := getMarketOrder(tm, now, types.OrderTypeMarket, types.OrderTimeInForceIOC, party3, types.SideBuy, party3, posSize, matchingPrice)

// 	confirmation, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.NoError(t, err)
// 	require.NotNil(t, confirmation)
// 	require.Equal(t, 1, len(confirmation.Trades))

// 	// both parties low on margin
// 	bal2 := tm.PartyGeneralAccount(t, party2).Balance
// 	bal3 := tm.PartyGeneralAccount(t, party3).Balance
// 	require.True(t, bal2.LT(num.NewUint(50)), bal2.String())
// 	require.True(t, bal3.LT(num.NewUint(50)), bal3.String())

// 	// parties try to place more limit orders in the same direction and fail
// 	changeSizeTo(party2Order, 1)
// 	party2Order.Type = types.OrderTypeMarket

// 	changeSizeTo(party3Order, 1)
// 	party3Order.Type = types.OrderTypeMarket

// 	_, err = tm.market.SubmitOrder(ctx, party2Order)
// 	require.ErrorContains(t, err, "margin")
// 	_, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.ErrorContains(t, err, "margin")

// 	// parties try to reduce position with market order of size greater than position and fail
// 	party2Order.Side = types.SideBuy
// 	changeSizeTo(party2Order, posSize+1)
// 	party2Order.Type = types.OrderTypeMarket

// 	party3Order.Side = types.SideSell
// 	changeSizeTo(party3Order, posSize+1)
// 	party3Order.Type = types.OrderTypeMarket

// 	_, err = tm.market.SubmitOrder(ctx, party2Order)
// 	require.ErrorContains(t, err, "margin")
// 	_, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.ErrorContains(t, err, "margin")

// 	// parties try to reduce position with passive limit order of size less than position and succeed
// 	changeSizeTo(party2Order, posSize-2)
// 	party2Order.Type = types.OrderTypeLimit
// 	party2Order.TimeInForce = types.OrderTimeInForceGTC
// 	party2Order.Price = num.UintZero().Sub(num.NewUint(matchingPrice), num.NewUint(501))

// 	changeSizeTo(party3Order, posSize-2)
// 	party3Order.Type = types.OrderTypeLimit
// 	party3Order.TimeInForce = types.OrderTimeInForceGTC
// 	party3Order.Price = num.UintZero().Add(num.NewUint(matchingPrice), num.NewUint(501))

// 	conf, err = tm.market.SubmitOrder(ctx, party2Order)
// 	require.NoError(t, err)
// 	require.Empty(t, conf.Trades)
// 	conf, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.NoError(t, err)
// 	require.Empty(t, conf.Trades)

// 	// parties try to place a pegged order in the same opposite direction and fail
// 	party2Order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceMid, Offset: num.NewUint(10)}
// 	changeSizeTo(party2Order, 1)
// 	_, err = tm.market.SubmitOrder(ctx, party2Order)
// 	require.ErrorContains(t, err, "margin")
// 	party2Order.PeggedOrder = nil

// 	party3Order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceMid, Offset: num.NewUint(10)}
// 	changeSizeTo(party2Order, 1)
// 	_, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.ErrorContains(t, err, "margin")
// 	party3Order.PeggedOrder = nil

// 	// parties place more passive limit orders so that the total order size is greater than position size and fail
// 	changeSizeTo(party2Order, 5)
// 	changeSizeTo(party3Order, 5)

// 	_, err = tm.market.SubmitOrder(ctx, party2Order)
// 	require.ErrorContains(t, err, "margin")
// 	_, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.ErrorContains(t, err, "margin")

// 	// parties successfully reduce their position with market order
// 	partialReduction := uint64(5)
// 	changeSizeTo(party2Order, partialReduction)
// 	party2Order.Type = types.OrderTypeMarket
// 	party2Order.TimeInForce = types.OrderTimeInForceFOK

// 	changeSizeTo(party3Order, partialReduction)
// 	party3Order.Type = types.OrderTypeMarket
// 	party3Order.TimeInForce = types.OrderTimeInForceFOK

// 	_, err = tm.market.SubmitOrder(ctx, party2Order)
// 	require.NoError(t, err)
// 	_, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.NoError(t, err)

// 	// parties try to close position with order with size equal to initial position size but fail as the position is now smaller
// 	changeSizeTo(party2Order, posSize-partialReduction)
// 	changeSizeTo(party3Order, posSize-partialReduction)

// 	_, err = tm.market.SubmitOrder(ctx, party3Order)
// 	require.NoError(t, err)
// 	_, err = tm.market.SubmitOrder(ctx, party2Order)
// 	require.NoError(t, err)
// }

func changeSizeTo(ord *types.Order, size uint64) {
	ord.Size = size
	ord.Remaining = size
}
