package execution_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startMarketInAuction(t *testing.T, ctx context.Context, now *time.Time) *testMarket {
	closingAt := time.Unix(1000000000, 0)

	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{
				{Horizon: 60, Probability: 0.95, AuctionExtension: 60},
			},
		},
		UpdateFrequency: 600,
	}

	tm := getTestMarket(t, *now, closingAt, pMonitorSettings, nil)

	addAccountWithAmount(tm, "trader-A", 1000)
	addAccountWithAmount(tm, "trader-B", 100000000)
	addAccountWithAmount(tm, "trader-C", 100000000)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), 10*time.Second)
	// Start the opening auction
	tm.mas.StartOpeningAuction(*now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Reset the event counter
	clearEvents(tm)

	return tm
}

func leaveAuction(tm *testMarket, ctx context.Context, now *time.Time) {
	// Leave auction to force the order to be removed
	*now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, *now)
}

func processEventsWithCounter(t *testing.T, tm *testMarket, mdb *subscribers.MarketDepthBuilder, i int) {
	for _, event := range tm.orderEvents {
		mdb.Push(event)
	}
	needToQuit := false
	orders := mdb.GetAllOrders(tm.market.GetID())
	for _, order := range orders {
		if !tm.market.ValidateOrder(types.OrderFromProto(order)) {
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
	processEventsWithCounter(t, tm, mdb, 0)
}

func clearEvents(tm *testMarket) {
	// Reset the event counter
	tm.eventCount = 0
	tm.orderEventCount = 0
	tm.events = nil
	tm.orderEvents = nil
}

// Check that the orders in the matching engine are the same as the orders in the market depth
func checkConsistency(t *testing.T, tm *testMarket, mdb *subscribers.MarketDepthBuilder) bool {
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
	if !assert.Equal(t, tm.market.GetMarketData().BestBidPrice, mdb.GetBestBidPrice(tm.market.GetID())) {
		correct = false
	}
	// Do we have the same best ask price?
	if !assert.Equal(t, tm.market.GetMarketData().BestOfferPrice, mdb.GetBestAskPrice(tm.market.GetID())) {
		correct = false
	}

	// Check volume at each level is correct
	bestBid := tm.market.GetMarketData().BestBidPrice
	bestAsk := tm.market.GetMarketData().BestOfferPrice

	if !assert.Equal(t, tm.market.GetMarketData().BestBidVolume, mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_BUY, bestBid)) {
		correct = false
	}

	if !assert.Equal(t, tm.market.GetMarketData().BestOfferVolume, mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, bestAsk)) {
		fmt.Println("BestAskVolume in OB:", tm.market.GetMarketData().BestOfferVolume)
		fmt.Println("BestAskVolume in MD:", mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, bestAsk))
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
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "Order01", types.Side_SIDE_BUY, "trader-A", 10, 10)
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
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 100001)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_BUY, "trader-A", 10, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_SELL, "trader-B", 1, 10)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Move the mark price super high to force a price auction
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 100000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 100000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we are in a price auction
	assert.Equal(t, types.AuctionTrigger_AUCTION_TRIGGER_PRICE, tm.market.GetMarketData().Trigger)

	// Check we have the right amount of events
	assert.Equal(t, uint64(8), tm.orderEventCount)
	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(4), mdb.GetOrderCount(tm.market.GetID()))
}

func TestEvents_CloseOutTrader(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_SELL, "trader-A", 10, 2)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_BUY, "trader-B", 10, 2)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "trader-A", 1, 10)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	// Move price high to force a close out
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 100)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 100, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	md = tm.market.GetMarketData()
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, md.MarketTradingMode)

	// Check we have the right amount of events
	assert.Equal(t, uint64(14), tm.orderEventCount)
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(3), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, uint64(89), mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
}

func TestEvents_CloseOutTraderWithPeggedOrder(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_SELL, "trader-A", 10, 2)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_BUY, "trader-B", 10, 2)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 100)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o6 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order06", types.Side_SIDE_BUY, "trader-B", 1, 99)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "trader-A", 1, 0)
	o5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o7 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order07", types.Side_SIDE_BUY, "trader-A", 1, 0)
	o7.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -110}
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NotNil(t, o7conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 100, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	md = tm.market.GetMarketData()
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, md.MarketTradingMode)

	// Check we have the right amount of events
	// assert.Equal(t, uint64(15), tm.orderEventCount)
	assert.Equal(t, uint64(17), tm.orderEventCount)
	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(4), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, uint64(89), mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func TestEvents_PeggedOrderNotAbleToRepriceDueToMargin(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-C", 1, 200)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_BUY, "trader-B", 1, 100)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "trader-A", 50, 0)
	o5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Move the best bid price up so that the pegged order cannot reprice
	o7 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order07", types.Side_SIDE_BUY, "trader-B", 2, 200)
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
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 1000001)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	md := tm.market.GetMarketData()
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, md.MarketTradingMode)

	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-C", 2, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_SELL, "trader-B", 1, 10)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_BUY, "trader-B", 1, 0)
	o4.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	// Move the mark price super high to force a price auction
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 1000000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 1, 1000000)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we are in a price auction
	assert.Equal(t, types.AuctionTrigger_AUCTION_TRIGGER_PRICE, tm.market.GetMarketData().Trigger)

	// Check we have the right amount of events
	assert.Equal(t, uint64(10), tm.orderEventCount)
	assert.Equal(t, int64(5), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(5), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 5, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_SelfTrading(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-C", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-B", 2, 10)
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

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-C", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	amendment := &commandspb.OrderAmendment{
		OrderId:  o1.Id,
		MarketId: o1.MarketId,
		Price:    &types.Price{Value: 11},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o1.PartyId)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = &types.Price{Value: 9}
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.PartyId)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = nil
	amendment.SizeDelta = 3
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.PartyId)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.SizeDelta = -2
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.PartyId)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.SizeDelta = 1
	amendment.Price = &types.Price{Value: 10}
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o1.PartyId)
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
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-C", 1, 20)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_BUY, "trader-A", 1, 0)
	o3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &commandspb.OrderAmendment{
		OrderId:  o2.Id,
		MarketId: o2.MarketId,
		Price:    &types.Price{Value: 8},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o2.PartyId)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = &types.Price{Value: 18}
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o2.PartyId)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = &types.Price{Value: 22}
	amendConf, err = tm.market.AmendOrder(ctx, amendment, o2.PartyId)
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
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_SELL, "trader-C", 2, 20)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_BUY, "trader-A", 1, 0)
	o3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &commandspb.OrderAmendment{
		OrderId:  o1.Id,
		MarketId: o1.MarketId,
		Price:    &types.Price{Value: 9},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o1.PartyId)
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
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-C", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 10)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-B", 2, 11)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &commandspb.OrderAmendment{
		OrderId:  o3.Id,
		MarketId: o3.MarketId,
		Price:    &types.Price{Value: 10},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o3.PartyId)
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
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(ctx, 0)

	auxParty := "aux"
	addAccount(tm, auxParty)
	auxOrder1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderBuy", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(ctx, auxOrder1)
	require.NotNil(t, conf)
	require.NoError(t, err)

	auxOrder2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "AuxOrderSell", types.Side_SIDE_SELL, auxParty, 1, 101)
	conf, err = tm.market.SubmitOrder(ctx, auxOrder2)
	require.NotNil(t, conf)
	require.NoError(t, err)

	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-C", 5, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 5, 11)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-A", 1, 12)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	amendment := &commandspb.OrderAmendment{
		OrderId:   o3.Id,
		MarketId:  o3.MarketId,
		Price:     &types.Price{Value: 11},
		SizeDelta: 5,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, o3.PartyId)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(7), tm.orderEventCount)
	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(4), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 4, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_CloseOutTraderWithNotEnoughLiquidity(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)

	// place some orders on the book for when we leave auction
	addAccountWithAmount(tm, "trader-X", 100000000)
	addAccountWithAmount(tm, "trader-Y", 100000000)

	orders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "auctionOrder1", types.Side_SIDE_SELL, "trader-X", 5, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "auctionOrder2", types.Side_SIDE_BUY, "trader-Y", 5, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auctionOrder3", types.Side_SIDE_SELL, "trader-X", 10, 3),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auctionOrder4", types.Side_SIDE_BUY, "trader-Y", 10, 2),
	}
	for _, o := range orders {
		_, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
	}
	// move time forwards 20 seconds, so the opening auction can end
	now = now.Add(time.Second * 20)
	tm.market.OnChainTimeUpdate(ctx, now)
	// leaveAuction(tm, ctx, &now)

	// Add a GFN order
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_SELL, "trader-A", 30, 1)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	// Fill some of it to set the mark price
	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_BUY, "trader-B", 30, 1)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 100)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o6 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order06", types.Side_SIDE_BUY, "trader-B", 1, 99)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "trader-A", 1, 0)
	o5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 10, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(15), tm.orderEventCount)
	assert.Equal(t, int64(5), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(5), mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, uint64(10), mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
}

func TestEvents_LPOrderRecalculationDueToFill(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 98)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_BUY, "trader-B", 1, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_SELL, "trader-B", 1, 110)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	buys := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1, Proportion: 50},
	}
	sells := []*types.LiquidityOrder{
		{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 1, Proportion: 50},
	}

	lps := &commandspb.LiquidityProvisionSubmission{
		Fee:              "0.05",
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 10,
		Buys:             buys,
		Sells:            sells}

	err = tm.market.SubmitLiquidityProvision(ctx, lps, "trader-A", "LPOrder01")
	require.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetLPSCount())

	o6 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order06", types.Side_SIDE_SELL, "trader-C", 2, 99)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	// Check we have the right amount of events
	// assert.Equal(t, uint64(11), tm.orderEventCount)
	assert.Equal(t, uint64(4), tm.orderEventCount)
	assert.Equal(t, int64(4), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, int64(4), mdb.GetOrderCount(tm.market.GetID()))
	// assert.Equal(t, 2, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func TestEvents_PeggedOrders(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)
	tm := startMarketInAuction(t, ctx, &now)
	// place some orders on the book for when we leave auction
	addAccountWithAmount(tm, "trader-X", 100000000)
	addAccountWithAmount(tm, "trader-Y", 100000000)

	orders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "auctionOrder1", types.Side_SIDE_SELL, "trader-X", 5, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, "auctionOrder2", types.Side_SIDE_BUY, "trader-Y", 5, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auctionOrder3", types.Side_SIDE_SELL, "trader-X", 10, 103),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "auctionOrder4", types.Side_SIDE_BUY, "trader-Y", 10, 102),
	}
	for _, o := range orders {
		_, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
	}
	// move time forwards 20 seconds, so the opening auction can end
	now = now.Add(time.Second * 20)
	tm.market.OnChainTimeUpdate(ctx, now)
	// leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, "Order01", types.Side_SIDE_BUY, "trader-B", 2, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_BUY, "trader-B", 2, 98)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_SELL, "trader-C", 2, 110)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o6 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order06", types.Side_SIDE_SELL, "trader-C", 2, 112)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NotNil(t, o6conf)
	require.NoError(t, err)

	// Place the pegged order
	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "trader-A", 1, 0)
	o5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o7 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order07", types.Side_SIDE_BUY, "trader-A", 1, 0)
	o7.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -99}
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NotNil(t, o7conf)
	require.NoError(t, err)

	// Now cause the best bid to drop and cause a reprice
	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 2, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(15), tm.orderEventCount)
	assert.Equal(t, int64(8), tm.market.GetOrdersOnBookCount())

	processEvents(t, tm, mdb)
	assert.Equal(t, 2, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount()) //??
}
