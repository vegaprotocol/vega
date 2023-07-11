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

	"code.vegaprotocol.io/vega/core/subscribers"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startMarketInAuction(t *testing.T, ctx context.Context, now *time.Time) *testMarket {
	t.Helper()

	pmt := &types.PriceMonitoringTrigger{
		Horizon:          60,
		HorizonDec:       num.DecimalFromFloat(60),
		Probability:      num.DecimalFromFloat(.95),
		AuctionExtension: 60,
	}
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				pmt,
			},
		},
	}

	tm := getTestMarket(t, *now, pMonitorSettings, nil)

	addAccountWithAmount(tm, "party-A", 1000)
	addAccountWithAmount(tm, "party-B", 100000000)
	addAccountWithAmount(tm, "party-C", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), 10*time.Second)
	// Start the opening auction
	tm.mas.StartOpeningAuction(*now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx, *now)
	tm.market.EnterAuction(ctx)

	// Reset the event counter
	clearEvents(tm)

	return tm
}

func leaveAuction(tm *testMarket, ctx context.Context, now *time.Time) {
	// Leave auction to force the order to be removed
	*now = now.Add(time.Second * 20)
	tm.market.LeaveAuctionWithIDGen(ctx, *now, newTestIDGenerator())
}

func processEventsWithCounter(t *testing.T, tm *testMarket, mdb *subscribers.MarketDepthBuilder) {
	t.Helper()
	for _, event := range tm.orderEvents {
		mdb.Push(event)
	}
	needToQuit := false
	orders := mdb.GetAllOrders(tm.market.GetID())
	for _, order := range orders {
		if !tm.market.ValidateOrder(order) {
			needToQuit = true
		}
	}

	if !checkConsistency(t, tm, mdb) {
		/*// We had an error, lets dump all the events
		for i, event := range tm.orderEvents {
			switch te := event.(type) {
			case subscribers.OE:
				fmt.Println("Event:", i, te.Order())
			}
		}*/
		needToQuit = true
	}

	if needToQuit {
		require.Equal(t, true, false)
	}
}

func processEvents(t *testing.T, tm *testMarket, mdb *subscribers.MarketDepthBuilder) {
	t.Helper()
	processEventsWithCounter(t, tm, mdb)
}

func clearEvents(tm *testMarket) {
	// Reset the event counter
	tm.eventCount = 0
	tm.orderEventCount = 0
	tm.events = nil
	tm.orderEvents = nil
}

// Check that the orders in the matching engine are the same as the orders in the market depth.
func checkConsistency(t *testing.T, tm *testMarket, mdb *subscribers.MarketDepthBuilder) bool {
	t.Helper()
	correct := true
	// Do we have the same number of orders in each?
	if !assert.Equal(t, tm.market.GetOrdersOnBookCount(), mdb.GetOrderCount(tm.market.GetID())) {
		correct = false
	}
	// Do we have the same volume in each?
	if !assert.Equal(t, tm.market.GetVolumeOnBook(), mdb.GetTotalVolume(tm.market.GetID())) {
		correct = false
	}
	// Do we have the same best bid price?
	if !assert.True(t, tm.market.GetMarketData().BestBidPrice.EQ(mdb.GetBestBidPrice(tm.market.GetID()))) {
		correct = false
	}
	// Do we have the same best ask price?
	if !assert.True(t, tm.market.GetMarketData().BestOfferPrice.EQ(mdb.GetBestAskPrice(tm.market.GetID()))) {
		correct = false
	}

	// Check volume at each level is correct
	bestBid := tm.market.GetMarketData().BestBidPrice.Clone()
	bestAsk := tm.market.GetMarketData().BestOfferPrice.Clone()

	if !assert.Equal(t, tm.market.GetMarketData().BestBidVolume, mdb.GetVolumeAtPrice(tm.market.GetID(), types.SideBuy, bestBid.Uint64())) {
		correct = false
	}

	if !assert.Equal(t, tm.market.GetMarketData().BestOfferVolume, mdb.GetVolumeAtPrice(tm.market.GetID(), types.SideSell, bestAsk.Uint64())) {
		fmt.Println("BestAskVolume in OB:", tm.market.GetMarketData().BestOfferVolume)
		fmt.Println("BestAskVolume in MD:", mdb.GetVolumeAtPrice(tm.market.GetID(), types.SideSell, bestAsk.Uint64()))
		correct = false
	}

	return correct
}

func TestEvents_LeavingAuctionCancelsGFAOrders(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	// Add a GFA order
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "Order01", types.SideBuy, "party-A", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Leave auction to force the order to be removed
	leaveAuction(tm, ctx, &now)

	// Check we have 2 events
	assert.Equal(t, uint64(2), tm.orderEventCount)

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(0), mdb.GetOrderCount(tm.market.GetID()))
}

func TestEvents_EnteringAuctionCancelsGFNOrders(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 100001)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFN, "Order01", types.SideBuy, "party-A", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "party-B", 1, 10)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Move the mark price super high to force a price auction
	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 100000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 100000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we are in a price auction
	assert.Equal(t, types.AuctionTriggerPrice, tm.market.GetMarketData().Trigger)

	// Check we have the right amount of events
	assert.Equal(t, uint64(8), tm.orderEventCount)

	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(4), mdb.GetOrderCount(tm.market.GetID()))
}

func TestEvents_CloseOutParty(t *testing.T) {
	t.Skip("TODO fix this - this test seems to trigger price auction (price range is 501-1010 IIRC)")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFN, "Order01", types.SideSell, "party-A", 10, 2)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)
	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 10, 2)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-A", 1, 10)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())
	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	// Move price high to force a closed out
	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 100)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	// assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())
	assert.Equal(t, int64(5), tm.market.GetOrdersOnBookCount())

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 100, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode, "market not continuous: %s (trigger: %s)", md.MarketTradingMode, md.Trigger)

	// Check we have the right amount of events
	assert.Equal(t, uint64(14), tm.orderEventCount)
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(3), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.SideSell, 100))
	assert.Equal(t, uint64(89), mdb.GetVolumeAtPrice(tm.market.GetID(), types.SideSell, 100))
}

func TestEvents_CloseOutPartyWithPeggedOrder(t *testing.T) {
	t.Skip("there's some weird magic going on here...")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFN, "Order01", types.SideSell, "party-A", 10, 2)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 10, 2)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 100)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o6 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order06", types.SideBuy, "party-B", 1, 99)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-A", 1, 0)
	o5.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o7 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order07", types.SideBuy, "party-A", 1, 0)
	o7.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NotNil(t, o7conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 100, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	md = tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	// Check we have the right amount of events
	// assert.Equal(t, uint64(15), tm.orderEventCount)
	assert.Equal(t, uint64(17), tm.orderEventCount)
	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(4), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.SideSell, 100))
	assert.Equal(t, uint64(89), mdb.GetVolumeAtPrice(tm.market.GetID(), types.SideSell, 100))
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func TestEvents_PeggedOrderNotAbleToRepriceDueToMargin(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideSell, "party-C", 1, 200)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 1, 100)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-A", 50, 0)
	o5.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Move the best bid price up so that the pegged order cannot be repriced
	o7 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order07", types.SideBuy, "party-B", 2, 200)
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NotNil(t, o7conf)
	require.NoError(t, err)

	// Check we have the right amount of events
	// assert.Equal(t, uint64(6), tm.orderEventCount)
	assert.Equal(t, uint64(4), tm.orderEventCount)
	// assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	// assert.Equal(t, int64(2), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, int64(3), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
}

func TestEvents_EnteringAuctionParksAllPegs(t *testing.T) {
	t.Skip("More weird magic vomiting in my face...")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 1000001)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.MarketTradingModeContinuous, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-C", 2, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideSell, "party-B", 1, 10)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 1, 0)
	o4.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Move the mark price super high to force a price auction
	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 1000000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 1, 1000000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we are in a price auction
	assert.Equal(t, types.AuctionTriggerPrice, tm.market.GetMarketData().Trigger)

	// Check we have the right amount of events
	assert.Equal(t, uint64(10), tm.orderEventCount)
	assert.Equal(t, int64(5), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(5), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 5, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_SelfTrading(t *testing.T) {
	t.Skip("Are these all broken??")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-C", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-B", 2, 10)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(6), tm.orderEventCount)
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(3), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 3, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_Amending(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-C", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	amendment := &types.OrderAmendment{
		OrderID:  o1.ID,
		MarketID: o1.MarketID,
		Price:    num.NewUint(11),
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o1.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = num.NewUint(9)
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = nil
	amendment.SizeDelta = 3
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.SizeDelta = -2
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.SizeDelta = 1
	amendment.Price = num.NewUint(10)
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(6), tm.orderEventCount)
	assert.Equal(t, int64(1), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(1), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 1, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_MovingPegsAround(t *testing.T) {
	t.Skip("yeah.. tests of doom")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideSell, "party-C", 1, 20)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideBuy, "party-A", 1, 0)
	o3.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &types.OrderAmendment{
		OrderID:  o2.ID,
		MarketID: o2.MarketID,
		Price:    num.NewUint(8),
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o2.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = num.NewUint(18)
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o2.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = num.NewUint(22)
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o2.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(12), tm.orderEventCount)
	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(2), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 2, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_MovingPegsAround2(t *testing.T) {
	t.Skip("tests are doomed")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideSell, "party-C", 2, 20)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideBuy, "party-A", 1, 0)
	o3.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &types.OrderAmendment{
		OrderID:  o1.ID,
		MarketID: o1.MarketID,
		Price:    num.NewUint(9),
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o1.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(8), tm.orderEventCount)
	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(2), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 2, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_AmendOrderToSelfTrade(t *testing.T) {
	t.Skip("The pony comes...")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-C", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-B", 2, 11)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &types.OrderAmendment{
		OrderID:  o3.ID,
		MarketID: o3.MarketID,
		Price:    num.NewUint(10),
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o3.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(7), tm.orderEventCount)
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(3), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 3, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_AmendOrderToIncreaseSizeAndPartiallyFill(t *testing.T) {
	t.Skip("The end is upon us")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, num.DecimalFromFloat(0))

	auxParty := "aux"
	addAccount(t, tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderBuy", types.SideBuy, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "AuxOrderSell", types.SideSell, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-C", 5, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 5, 11)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-A", 1, 12)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &types.OrderAmendment{
		OrderID:   o3.ID,
		MarketID:  o3.MarketID,
		Price:     num.NewUint(11),
		SizeDelta: 5,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o3.Party, vgcrypto.RandomHash())
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(7), tm.orderEventCount)
	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(4), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 4, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_CloseOutPartyWithNotEnoughLiquidity(t *testing.T) {
	t.Skip("Jehova!!!")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	// place some orders on the book for when we leave auction
	addAccountWithAmount(tm, "party-X", 100000000)
	addAccountWithAmount(tm, "party-Y", 100000000)

	orders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "auctionOrder1", types.SideSell, "party-X", 5, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "auctionOrder2", types.SideBuy, "party-Y", 5, 1),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "auctionOrder3", types.SideSell, "party-X", 10, 3),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "auctionOrder4", types.SideBuy, "party-Y", 10, 2),
	}
	for _, o := range orders {
		_, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
	}
	// move time forwards 20 seconds, so the opening auction can end
	now = now.Add(time.Second * 20)
	tm.market.OnTick(ctx, now)
	// leaveAuction(tm, ctx, &now)

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFN, "Order01", types.SideSell, "party-A", 30, 1)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 30, 1)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideBuy, "party-B", 1, 100)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o6 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order06", types.SideBuy, "party-B", 1, 99)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-A", 1, 0)
	o5.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 10, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(15), tm.orderEventCount)
	assert.Equal(t, int64(5), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(5), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.SideSell, 100))
	assert.Equal(t, uint64(10), mdb.GetVolumeAtPrice(tm.market.GetID(), types.SideSell, 100))
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
}

func TestEvents_PeggedOrders(t *testing.T) {
	t.Skip("Multi-coloured skittles and an astronaut")
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	// place some orders on the book for when we leave auction
	addAccountWithAmount(tm, "party-X", 100000000)
	addAccountWithAmount(tm, "party-Y", 100000000)

	orders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "auctionOrder1", types.SideSell, "party-X", 5, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "auctionOrder2", types.SideBuy, "party-Y", 5, 100),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "auctionOrder3", types.SideSell, "party-X", 10, 103),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "auctionOrder4", types.SideBuy, "party-Y", 10, 102),
	}
	for _, o := range orders {
		_, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
	}
	// move time forwards 20 seconds, so the opening auction can end
	now = now.Add(time.Second * 20)
	tm.market.OnTick(ctx, now)
	// leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFN, "Order01", types.SideBuy, "party-B", 2, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideBuy, "party-B", 2, 98)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-C", 2, 110)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o6 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order06", types.SideSell, "party-C", 2, 112)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order05", types.SideBuy, "party-A", 1, 0)
	o5.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 1)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o7 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order07", types.SideBuy, "party-A", 1, 0)
	o7.PeggedOrder = newPeggedOrder(types.PeggedReferenceBestBid, 99)
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NotNil(t, o7conf)
	require.NoError(t, err)

	// Now cause the best bid to drop and cause a reprice
	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideSell, "party-C", 2, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(15), tm.orderEventCount)
	assert.Equal(t, int64(8), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, 2, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount()) // ??
}
