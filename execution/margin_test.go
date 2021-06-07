package execution_test

import (
	"context"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMargins(t *testing.T) {
	party1, party2, party3 := "party1", "party2", "party3"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	price := uint64(100)
	size := uint64(100)

	addAccount(tm, party1)
	addAccount(tm, party2)
	addAccount(tm, party3)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	// set auction durations to 1 second
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 100000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_BUY, auxParty, 1, price),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_SELL, auxParty2, 1, price),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}

	now = now.Add(2 * time.Second)
	// leave opening auction
	tm.market.OnChainTimeUpdate(context.Background(), now)
	data := tm.market.GetMarketData()
	require.Equal(t, types.Market_TRADING_MODE_CONTINUOUS, data.MarketTradingMode)

	order1 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid12",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        size,
		Price:       price,
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-buy-order",
	}
	order2 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid123",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party3,
		MarketId:    tm.market.GetID(),
		Size:        size,
		Price:       price,
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
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        size,
		Price:       price,
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

	orderID := confirmation.Order.Id

	// Amend size up
	amend := &commandspb.OrderAmendment{
		OrderId:   orderID,
		MarketId:  tm.market.GetID(),
		SizeDelta: int64(10000),
	}
	amendment, err := tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend price and size up to breach margin
	amend.SizeDelta = 1000000000
	amend.Price = &types.Price{Value: 1000000000}
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.Nil(t, amendment)
	assert.Error(t, err)
}

/* Check that a failed new order margin check cannot be got around by amending
 * an existing order to the same values as the failed new order */
func TestPartialFillMargins(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	party3 := "party3"
	auxParty, auxParty2 := "auxParty", "auxParty2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})

	addAccount(tm, party1)
	addAccount(tm, party2)
	addAccount(tm, party3)
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	// ensure auction durations are 1 second
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)
	// create orders so we can leave opening auction
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_BUY, auxParty, 1, 10000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_SELL, auxParty2, 1, 10000000),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}
	now = now.Add(time.Second * 2) // opening auction is 1 second, move time ahead by 2 seconds so we leave auction
	tm.market.OnChainTimeUpdate(context.Background(), now)

	// use party 2+3 to set super high mark price
	orderSell1 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       10000000,
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
		Type:        types.Order_TYPE_MARKET,
		TimeInForce: types.Order_TIME_IN_FORCE_IOC,
		Side:        types.Side_SIDE_BUY,
		PartyId:     party3,
		MarketId:    tm.market.GetID(),
		Size:        1,
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

	// Attempt to create a new order for party1 that will be margin blocked
	orderBuy2 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        1000,
		Remaining:   1000,
		CreatedAt:   now.UnixNano(),
		ExpiresAt:   now.UnixNano() + 10000000000,
		Reference:   "party1-buy-order",
	}

	confirmation, err = tm.market.SubmitOrder(context.TODO(), orderBuy2)
	assert.Error(t, err)
	assert.Nil(t, confirmation)

	// Create a valid smaller order
	orderBuy3 := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        1,
		Price:       2,
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
	orderID := confirmation.Order.Id

	// Attempt to amend it to the same size as the failed new order
	amend := &commandspb.OrderAmendment{
		OrderId:   orderID,
		MarketId:  tm.market.GetID(),
		SizeDelta: int64(999),
	}
	amendment, err := tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.Nil(t, amendment)
	assert.Error(t, err)
}
