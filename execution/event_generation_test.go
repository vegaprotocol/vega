package execution_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

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

	// Start the opening auction
	tm.mas.StartOpeningAuction(*now, &types.AuctionDuration{Duration: 10})
	tm.mas.AuctionStarted(ctx)
	tm.market.EnterAuction(ctx)

	// Reset the event counter
	tm.eventCount = 0
	tm.orderEventCount = 0
	tm.events = nil

	return tm
}

func leaveAuction(tm *testMarket, ctx context.Context, now *time.Time) {
	// Leave auction to force the order to be removed
	*now = now.Add(time.Second * 20)
	tm.market.LeaveAuction(ctx, *now)
}

func processEvents(t *testing.T, tm *testMarket, ctx context.Context) *subscribers.MarketDepthBuilder {
	mdb := subscribers.NewMarketDepthBuilder(ctx, nil, true)

	for _, event := range tm.orderEvents {
		mdb.Push(event)
	}
	return mdb
}

func TestEvents_LeavingAuctionCancelsGFAOrders(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
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

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 0, mdb.GetOrderCount(tm.market.GetID()))
}

func TestEvents_EnteringAuctionCancelsGFNOrders(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

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

	// Check we have 6 events
	assert.Equal(t, uint64(6), tm.orderEventCount)
	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 2, mdb.GetOrderCount(tm.market.GetID()))
}

func TestEvents_CloseOutTrader(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

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

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "trader-A", 1, 10)
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	// Move price high to force a close out
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order02", types.Side_SIDE_BUY, "trader-B", 1, 100)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 100, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we have 6 events
	assert.Equal(t, uint64(12), tm.orderEventCount)
	assert.Equal(t, int64(1), tm.market.GetOrdersOnBookCount())

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 1, mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, uint64(69), mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
}

func TestEvents_CloseOutTraderWithPeggedOrder(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

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
	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order05", types.Side_SIDE_BUY, "trader-A", 1, 10)
	o5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NotNil(t, o5conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order03", types.Side_SIDE_SELL, "trader-C", 100, 100)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	// Check we have 6 events
	assert.Equal(t, uint64(13), tm.orderEventCount)
	assert.Equal(t, int64(2), tm.market.GetOrdersOnBookCount())

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 2, mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, uint64(69), mdb.GetVolumeAtPrice(tm.market.GetID(), types.Side_SIDE_SELL, 100))
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func TestEvents_EnteringAuctionParksAllPegs(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

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

	// Check we have 6 events
	assert.Equal(t, uint64(8), tm.orderEventCount)
	assert.Equal(t, int64(3), tm.market.GetOrdersOnBookCount())

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 3, mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 3, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_SelfTrading(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	tm := startMarketInAuction(t, ctx, &now)
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
	assert.Equal(t, uint64(4), tm.orderEventCount)
	assert.Equal(t, int64(1), tm.market.GetOrdersOnBookCount())

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 1, mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 1, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_Amending(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	tm := startMarketInAuction(t, ctx, &now)
	leaveAuction(tm, ctx, &now)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "trader-C", 1, 10)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	amendment := &types.OrderAmendment{
		OrderId:  o1.Id,
		PartyId:  o1.PartyId,
		MarketId: o1.MarketId,
		Price:    &types.Price{Value: 11},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = &types.Price{Value: 9}
	amendConf, err = tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = nil
	amendment.SizeDelta = 3
	amendConf, err = tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.SizeDelta = -2
	amendConf, err = tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.SizeDelta = 1
	amendment.Price = &types.Price{Value: 10}
	amendConf, err = tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(6), tm.orderEventCount)
	assert.Equal(t, int64(1), tm.market.GetOrdersOnBookCount())

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 1, mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 1, mdb.GetPriceLevels(tm.market.GetID()))
}

func TestEvents_MovingPegsAround(t *testing.T) {
	now := time.Unix(10, 0)
	ctx := context.Background()
	tm := startMarketInAuction(t, ctx, &now)
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

	amendment := &types.OrderAmendment{
		OrderId:  o2.Id,
		PartyId:  o2.PartyId,
		MarketId: o2.MarketId,
		Price:    &types.Price{Value: 8},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = &types.Price{Value: 18}
	amendConf, err = tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amendment.Price = &types.Price{Value: 22}
	amendConf, err = tm.market.AmendOrder(ctx, amendment)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Check we have the right amount of events
	assert.Equal(t, uint64(10), tm.orderEventCount)
	assert.Equal(t, int64(0), tm.market.GetOrdersOnBookCount())

	mdb := processEvents(t, tm, ctx)
	assert.Equal(t, 0, mdb.GetOrderCount(tm.market.GetID()))
	assert.Equal(t, 0, mdb.GetPriceLevels(tm.market.GetID()))
}
