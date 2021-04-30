package execution_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/golang/mock/gomock"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderBufferOutputCount(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, party1)

	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
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
	orderAmend.Id = "amendingorder"
	orderAmend.Reference = "amendingorderreference"
	confirmation, err = tm.market.SubmitOrder(context.TODO(), &orderAmend)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	amend := &commandspb.OrderAmendment{
		MarketId: tm.market.GetID(),
		PartyId:  party1,
		OrderId:  orderAmend.Id,
	}

	// Amend price down (generates one order message)
	amend.Price = &types.Price{Value: orderBuy.Price - 1}
	amendConf, err := tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend price up (generates one order message)
	amend.Price = &types.Price{Value: orderBuy.Price + 1}
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend size down (generates one order message)
	amend.Price = nil
	amend.SizeDelta = -1
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend size up (generates one order message)
	amend.SizeDelta = +1
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend TIME_IN_FORCE -> GTT (generates one order message)
	amend.SizeDelta = 0
	amend.TimeInForce = types.Order_TIME_IN_FORCE_GTT
	amend.ExpiresAt = &types.Timestamp{Value: now.UnixNano() + 100000000000}
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend TIME_IN_FORCE -> GTC (generates one order message)
	amend.TimeInForce = types.Order_TIME_IN_FORCE_GTC
	amend.ExpiresAt = nil
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend ExpiresAt (generates two order messages)
	amend.TimeInForce = types.Order_TIME_IN_FORCE_GTT
	amend.ExpiresAt = &types.Timestamp{Value: now.UnixNano() + 100000000000}
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amend.ExpiresAt = &types.Timestamp{Value: now.UnixNano() + 200000000000}
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)
}

func TestAmendCancelResubmit(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, party1)

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	orderID := confirmation.GetOrder().Id

	// Amend the price to force a cancel+resubmit to the order book

	amend := &commandspb.OrderAmendment{
		OrderId:  orderID,
		PartyId:  confirmation.GetOrder().GetPartyId(),
		MarketId: confirmation.GetOrder().GetMarketId(),
		Price:    &types.Price{Value: 101},
	}
	amended, err := tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	amend = &commandspb.OrderAmendment{
		OrderId:   orderID,
		PartyId:   confirmation.GetOrder().GetPartyId(),
		MarketId:  confirmation.GetOrder().GetMarketId(),
		Price:     &types.Price{Value: 101},
		SizeDelta: 1,
	}
	amended, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)
}

func TestCancelWithWrongPartyID(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Now attempt to cancel it with the wrong partyID
	cancelOrder := &commandspb.OrderCancellation{
		OrderId:  confirmation.GetOrder().Id,
		MarketId: confirmation.GetOrder().MarketId,
	}
	cancelconf, err := tm.market.CancelOrder(context.TODO(), party2, cancelOrder.OrderId)
	assert.Nil(t, cancelconf)
	assert.Error(t, err, types.ErrInvalidPartyID)
}

func TestMarkPriceUpdateAfterPartialFill(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})

	addAccount(tm, party1)
	addAccount(tm, party2)
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	//Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       10,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
		Type:        types.Order_TYPE_LIMIT,
	}
	// Submit the original order
	buyConfirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	orderSell := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIME_IN_FORCE_IOC,
		Id:          "someid",
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        50,
		Price:       10,
		Remaining:   50,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
		Type:        types.Order_TYPE_MARKET,
	}
	// Submit an opposite order to partially fill
	sellConfirmation, err := tm.market.SubmitOrder(context.TODO(), orderSell)
	assert.NotNil(t, sellConfirmation)
	assert.NoError(t, err)

	// Validate that the mark price has been updated
	assert.EqualValues(t, tm.market.GetMarketData().MarkPrice, 10)
}

func TestExpireCancelGTCOrder(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, party1)

	orderBuy := &types.Order{
		CreatedAt:   10000000000,
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       10,
		Remaining:   100,
		Reference:   "party1-buy-order",
		Type:        types.Order_TYPE_LIMIT,
	}
	// Submit the original order
	buyConfirmation, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	// Move the current time forward
	tm.market.OnChainTimeUpdate(context.Background(), time.Unix(10, 100))

	amend := &commandspb.OrderAmendment{
		OrderId:     buyConfirmation.GetOrder().GetId(),
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		ExpiresAt:   &types.Timestamp{Value: 10000000010},
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
	}
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	// Validate that the mark price has been updated
	assert.EqualValues(t, amended.Order.TimeInForce, types.Order_TIME_IN_FORCE_GTT)
	assert.EqualValues(t, amended.Order.Status, types.Order_STATUS_EXPIRED)
	assert.EqualValues(t, amended.Order.CreatedAt, 10000000000)
	assert.EqualValues(t, amended.Order.ExpiresAt, 10000000010)
	assert.EqualValues(t, amended.Order.UpdatedAt, 10000000100)
}

func TestAmendPartialFillCancelReplace(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})

	addAccount(tm, party1)
	addAccount(tm, party2)
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	//Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 5),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 5),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        20,
		Price:       5,
		Remaining:   20,
		Reference:   "party1-buy-order",
		Type:        types.Order_TYPE_LIMIT,
	}
	// Place an order
	buyConfirmation, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	orderSell := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIME_IN_FORCE_IOC,
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        10,
		Price:       5,
		Remaining:   10,
		Reference:   "party2-sell-order",
		Type:        types.Order_TYPE_MARKET,
	}
	// Partially fill the original order
	sellConfirmation, err := tm.market.SubmitOrder(context.Background(), orderSell)
	assert.NotNil(t, sellConfirmation)
	assert.NoError(t, err)

	amend := &commandspb.OrderAmendment{
		OrderId:  buyConfirmation.GetOrder().GetId(),
		PartyId:  party1,
		MarketId: tm.market.GetID(),
		Price:    &types.Price{Value: 20},
	}
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	// Check the values are correct
	assert.EqualValues(t, amended.Order.Price, 20)
	assert.EqualValues(t, amended.Order.Remaining, 10)
	assert.EqualValues(t, amended.Order.Size, 20)
}

func TestAmendWrongPartyID(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(context.Background(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Send an amend but use the wrong partyID
	amend := &commandspb.OrderAmendment{
		OrderId:  confirmation.GetOrder().GetId(),
		PartyId:  party2,
		MarketId: confirmation.GetOrder().GetMarketId(),
		Price:    &types.Price{Value: 101},
	}
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	assert.Nil(t, amended)
	assert.Error(t, err, types.ErrInvalidPartyID)
}

func TestPartialFilledWashTrade(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	addAccount(tm, party1)
	addAccount(tm, party2)
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	//Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	alwaysOnBid := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 55),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 55),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderSell1 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Side:        types.Side_SIDE_SELL,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        15,
		Price:       55,
		Remaining:   15,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-sell-order",
	}
	confirmation, err := tm.market.SubmitOrder(context.Background(), orderSell1)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	orderSell2 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Side:        types.Side_SIDE_SELL,
		PartyId:     party2,
		MarketId:    tm.market.GetID(),
		Size:        15,
		Price:       53,
		Remaining:   15,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.Background(), orderSell2)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// This order should partially fill and then be rejected
	orderBuy1 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        30,
		Price:       60,
		Remaining:   30,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARTIALLY_FILLED)
	assert.Equal(t, confirmation.Order.Remaining, uint64(15))
}

func getAmend(market string, party string, orderID string, sizeDelta int64, price uint64,
	tif types.Order_TimeInForce, expiresAt int64) *commandspb.OrderAmendment {

	amend := &commandspb.OrderAmendment{
		OrderId:     orderID,
		PartyId:     party,
		MarketId:    market,
		SizeDelta:   sizeDelta,
		TimeInForce: tif,
	}

	if price > 0 {
		amend.Price = &types.Price{Value: price}
	}

	if expiresAt > 0 {
		amend.ExpiresAt = &types.Timestamp{Value: expiresAt}
	}

	return amend
}

func amendOrder(t *testing.T, tm *testMarket, party string, orderID string, sizeDelta int64, price uint64,
	tif types.Order_TimeInForce, expiresAt int64, pass bool) {
	amend := getAmend(tm.market.GetID(), party, orderID, sizeDelta, price, tif, expiresAt)

	amended, err := tm.market.AmendOrder(context.Background(), amend)
	if pass {
		assert.NotNil(t, amended)
		assert.NoError(t, err)
	}
}

func getOrder(_ *testing.T, tm *testMarket, now *time.Time, orderType types.Order_Type, tif types.Order_TimeInForce,
	expiresAt int64, side types.Side, party string, size uint64, price uint64) types.Order {
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        orderType,
		TimeInForce: tif,
		Side:        side,
		PartyId:     party,
		MarketId:    tm.market.GetID(),
		Size:        size,
		Price:       price,
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "",
	}

	if expiresAt > 0 {
		order.ExpiresAt = expiresAt
	}
	return order
}

func sendOrder(t *testing.T, tm *testMarket, now *time.Time, orderType types.Order_Type, tif types.Order_TimeInForce, expiresAt int64, side types.Side, party string,
	size uint64, price uint64) string {
	order := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        orderType,
		TimeInForce: tif,
		Side:        side,
		PartyId:     party,
		MarketId:    tm.market.GetID(),
		Size:        size,
		Price:       price,
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "",
	}

	if expiresAt > 0 {
		order.ExpiresAt = expiresAt
	}

	confirmation, err := tm.market.SubmitOrder(context.Background(), order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Move time forward one second
	//	*now = now.Add(time.Second)
	//	tm.market.OnChainTimeUpdate(*now)

	return confirmation.GetOrder().Id
}

func TestAmendToFill(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")
	addAccount(tm, "party2")

	// test_AmendMarketOrderFail
	_ = sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 100)      // 1 - a8
	_ = sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 110)      // 1 - a8
	_ = sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 120)      // 1 - a8
	orderID := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party2", 40, 50) // 1 - a8
	amendOrder(t, tm, "party2", orderID, 0, 500, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0, true)
}

func TestAmendToLosePriorityThenCancel(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")
	addAccount(tm, "party2")

	// Create 2 orders at the same level
	order1 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)
	_ = sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)

	// Amend the first order to make it lose time priority
	amendOrder(t, tm, "party1", order1, 1, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0, true)

	// Check we can cancel it
	cancelconf, _ := tm.market.CancelOrder(context.TODO(), "party1", order1)
	assert.NotNil(t, cancelconf)
	assert.Equal(t, types.Order_STATUS_CANCELLED, cancelconf.Order.Status)

}

func TestUnableToAmendGFAGFN(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	mainParty := "party1"
	auxParty := "party2"
	auxParty2 := "party22"
	addAccount(tm, mainParty)
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	//Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	tm.market.OnMarketAuctionMinimumDurationUpdate(context.Background(), time.Second)
	alwaysOnBid := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(context.Background(), o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(context.Background(), now)

	// test_AmendMarketOrderFail
	orderID := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, mainParty, 10, 100)
	amendOrder(t, tm, mainParty, orderID, 0, 0, types.Order_TIME_IN_FORCE_GFA, 0, false)
	amendOrder(t, tm, mainParty, orderID, 0, 0, types.Order_TIME_IN_FORCE_GFN, 0, false)

	orderID2 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFN, 0, types.Side_SIDE_SELL, mainParty, 10, 100)
	amendOrder(t, tm, mainParty, orderID2, 0, 0, types.Order_TIME_IN_FORCE_GTC, 0, false)
	amendOrder(t, tm, mainParty, orderID2, 0, 0, types.Order_TIME_IN_FORCE_GFA, 0, false)

	// EnterAuction should actually trigger an auction here...
	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(context.Background())
	orderID3 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GFA, 0, types.Side_SIDE_SELL, "party1", 10, 100)
	amendOrder(t, tm, "party1", orderID3, 0, 0, types.Order_TIME_IN_FORCE_GTC, 0, false)
	amendOrder(t, tm, "party1", orderID3, 0, 0, types.Order_TIME_IN_FORCE_GFN, 0, false)
}

func TestPeggedOrders(t *testing.T) {
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
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	addAccount(tm, "party2")

	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party2", 5, 9000)

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party2", 10, 0)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: +10}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 5, 9000)
}

func testPeggedOrderUnpark(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, "party2")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}

	// Create a single buy order to give this party a valid position
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 5, 11)

	// Add a pegged order which will park due to missing reference price
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())

	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)
	// Send a new order to set the BEST_ASK price and force the parked order to unpark
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party2", 5, 15)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrderAmendToMoveReference(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Place 2 orders to create valid reference prices
	bestBidOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 110)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Amend best bid price
	amendOrder(t, tm, "party1", bestBidOrder, 0, 88, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0, true)
	amendOrder(t, tm, "party1", bestBidOrder, 0, 86, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0, true)
}

func testPeggedOrderFilledOrder(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, "party2")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 110)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Place a sell MARKET order to fill the buy orders
	sendOrder(t, tm, &now, types.Order_TYPE_MARKET, types.Order_TIME_IN_FORCE_IOC, 0, types.Side_SIDE_SELL, "party2", 2, 0)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
}

func testParkedOrdersAreUnparkedWhenPossible(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, "party2")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 5)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)

	// Place a valid pegged order which will be parked because it cannot be repriced
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 1)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())

	// Send a higher buy price order to move the BEST BID price up
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 50)

	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testPeggedOrdersLeavingAuction(t *testing.T) {
	now := time.Unix(10, 0)
	auctionClose := now.Add(101 * time.Second)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 100,
	})
	ctx := context.Background()

	addAccount(tm, "party1")
	addAccount(tm, "party2")
	addAccount(tm, "party3")

	// Move into auction
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 100*time.Second)

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)
	// place 2 more orders that will result in a mark price being set
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party2", 1, 95)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party3", 1, 95)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -10}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARKED)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	// During an auction all pegged orders are parked so we don't add them to the list
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())

	// Update the time to force the auction to end
	tm.market.OnChainTimeUpdate(ctx, auctionClose)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrdersEnteringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 100,
	})
	ctx := context.Background()

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, "party2")
	addAccount(tm, "party3")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, 100*time.Second)
	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)
	// place 2 more orders that will result in a mark price being set
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party2", 1, 95)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party3", 1, 95)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -10}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARKED)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())

	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
}

func testPeggedOrderAddWithNoMarketPrice(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	// Place a valid pegged order which will be parked
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARKED)
	assert.NoError(t, err)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testPeggedOrderAdd(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 100)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 102)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	assert.NotNil(t, confirmation)
	assert.Equal(t, types.Order_STATUS_ACTIVE, confirmation.Order.Status)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())

	assert.Equal(t, uint64(98), order.Price)
}

func testPeggedOrderWithReprice(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 110)

	md := tm.market.GetMarketData()
	assert.Equal(t, uint64(100), md.MidPrice)
	// Place a valid pegged order which will be added to the order book
	// This order will cause the MID price to move and thus a reprice multiple times until it settles
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Check to make sure the existing pegged order is repriced correctly
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())

	// TODO need to find a way to validate details of the amended order
}

func testPeggedOrderParkWhenInAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARKED)
	assert.NoError(t, err)
}

func testPeggedOrderUnparkAfterLeavingAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARKED)
	assert.NoError(t, err)

	buy := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 90)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &buy)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	require.NotNil(t, buy)
	sell := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 110)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &sell)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	tm.market.LeaveAuction(ctx, closingAt)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrderTypes(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Not MARKET
	order.Type = types.Order_TYPE_MARKET
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)
}

func testPeggedOrderCancelParked(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")

	// Pegged order will be parked as no reference prices
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)
}

func testPeggedOrderTIFs(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}

	// Only allowed GTC
	order.Type = types.Order_TYPE_LIMIT
	order.TimeInForce = types.Order_TIME_IN_FORCE_GTC
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// and GTT
	order.TimeInForce = types.Order_TIME_IN_FORCE_GTT
	order.ExpiresAt = now.UnixNano() + 1000000000
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// but not IOC
	order.ExpiresAt = 0
	order.TimeInForce = types.Order_TIME_IN_FORCE_IOC
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	// or FOK
	order.TimeInForce = types.Order_TIME_IN_FORCE_FOK
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)
}

func testPeggedOrderBuys(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)

	// BEST BID peg must be <= 0
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: 3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: 0}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// MID peg must be < 0
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 0}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// BEST ASK peg not allowed
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: -3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 0}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)
}

func testPeggedOrderSells(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 100)

	// BEST BID peg not allowed
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: 3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: 0}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	// MID peg must be > 0
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 0}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	// BEST ASK peg must be >= 0
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: -3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 3}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 0}
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
}

func testPeggedOrderParkWhenPriceBelowZero(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	for _, acc := range []string{"buyer", "seller", "pegged"} {
		addAccount(tm, acc)
	}

	buy := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "buyer", 10, 4)
	_, err := tm.market.SubmitOrder(ctx, &buy)
	require.NoError(t, err)

	sell := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "seller", 10, 8)
	_, err = tm.market.SubmitOrder(ctx, &sell)
	require.NoError(t, err)

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "pegged", 10, 4)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -10}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.Equal(t,
		types.Order_STATUS_PARKED.String(),
		confirmation.Order.Status.String(), "When pegged price below zero (MIDPRICE - OFFSET) <= 0")
}

func testPeggedOrderParkWhenPriceRepricesBelowZero(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	for _, acc := range []string{"buyer", "seller", "pegged"} {
		addAccount(tm, acc)
	}

	buy := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "buyer", 10, 4)
	_, err := tm.market.SubmitOrder(ctx, &buy)
	require.NoError(t, err)

	sell := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "seller", 10, 8)
	_, err = tm.market.SubmitOrder(ctx, &sell)
	require.NoError(t, err)

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "pegged", 10, 4)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -5}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	amendOrder(t, tm, "buyer", buy.Id, 0, 1, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0, true)

	assert.Equal(t, types.Order_STATUS_PARKED.String(), confirmation.Order.Status.String())
}

/*func TestPeggedOrderCrash(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	for _, acc := range []string{"user1", "user2", "user3", "user4", "user5", "user6", "user7"} {
		addAccount(tm, acc)
	}

	// Set up the best bid/ask values
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user1", 5, 10500)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user2", 20, 11000)

	// Pegged order buy 35 MID -500
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user3", 35, 0)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -500}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Pegged order buy 16 BEST_BID -2000
	order2 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user4", 16, 0)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -2000}
	_, err = tm.market.SubmitOrder(ctx, &order2)
	require.NoError(t, err)

	// Pegged order sell 19 BEST_ASK 3000
	order3 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user5", 19, 0)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 3000}
	_, err = tm.market.SubmitOrder(ctx, &order3)
	require.NoError(t, err)

	// Buy 25 @ 10000
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user6", 25, 10000)

	// Sell 25 @ 10250
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user7", 25, 10250)
}*/

func testPeggedOrderParkCancelAll(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "user")

	// Send one normal order
	limitOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user", 10, 100)
	require.NotEmpty(t, limitOrder)

	// Send one pegged order that is live
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user", 10, 0)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -5}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)

	// Send one pegged order that is parked
	order2 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user", 10, 0)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -5}
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
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "user")

	// Send one normal expiring order
	limitOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, expire.UnixNano(), types.Side_SIDE_BUY, "user", 10, 100)
	require.NotEmpty(t, limitOrder)

	// Amend the expiry time
	amendOrder(t, tm, "user", limitOrder, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, now.UnixNano(), true)

	// Send one pegged order that will be parked
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, expire.UnixNano(), types.Side_SIDE_BUY, "user", 10, 0)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -5}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)

	// Send one pegged order that will also be parked (after additing liquidity monitoring to market all orders will be parked unless both best_bid and best_offer exist)
	order2 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, expire.UnixNano(), types.Side_SIDE_BUY, "user", 10, 0)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -5}
	confirmation, err = tm.market.SubmitOrder(ctx, &order2)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)

	assert.Equal(t, 2, tm.market.GetParkedOrderCount())
	assert.Equal(t, 2, tm.market.GetPeggedOrderCount())

	// Move the time forward
	orders, err := tm.market.RemoveExpiredOrders(context.Background(), afterexpire.UnixNano())
	require.NotNil(t, orders)
	assert.NoError(t, err)

	// Check that we have no pegged orders
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
}

func testPeggedOrderOutputMessages(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	addAccount(tm, "user1")
	addAccount(tm, "user2")
	addAccount(tm, "user3")
	addAccount(tm, "user4")
	addAccount(tm, "user5")
	addAccount(tm, "user6")
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)
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
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user1", 10, 0)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: 10}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, uint64(7), tm.orderEventCount)

	order2 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user2", 10, 0)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: 15}
	confirmation2, err := tm.market.SubmitOrder(ctx, &order2)
	require.NoError(t, err)
	assert.NotNil(t, confirmation2)
	assert.Equal(t, uint64(8), tm.orderEventCount)

	order3 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user3", 10, 0)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	confirmation3, err := tm.market.SubmitOrder(ctx, &order3)
	require.NoError(t, err)
	assert.NotNil(t, confirmation3)
	assert.Equal(t, uint64(9), tm.orderEventCount)

	order4 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user4", 10, 0)
	order4.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -10}
	confirmation4, err := tm.market.SubmitOrder(ctx, &order4)
	require.NoError(t, err)
	assert.NotNil(t, confirmation4)
	assert.Equal(t, uint64(10), tm.orderEventCount)

	limitOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user5", 1000, 120)
	require.NotEmpty(t, limitOrder)
	assert.Equal(t, uint64(14), tm.orderEventCount)

	limitOrder2 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user6", 1000, 80)
	require.NotEmpty(t, limitOrder2)
	assert.Equal(t, uint64(17), tm.orderEventCount)
}

func testPeggedOrderOutputMessages2(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()

	addAccount(tm, "user1")
	addAccount(tm, "user2")
	auxParty := "auxParty"
	auxParty2 := "auxParty2"
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	// Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 100000)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Create a pegged parked order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user1", 10, 0)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_PARKED, confirmation.Order.Status)
	assert.NotNil(t, confirmation)
	assert.Equal(t, uint64(7), tm.orderEventCount)

	// Send normal order to unpark the pegged order
	limitOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user2", 1000, 120)
	require.NotEmpty(t, limitOrder)
	assert.Equal(t, uint64(9), tm.orderEventCount)
	assert.Equal(t, types.Order_STATUS_ACTIVE, confirmation.Order.Status)

	// Cancel the normal order to park the pegged order
	tm.market.CancelOrder(ctx, "user2", limitOrder)
	require.Equal(t, types.Order_STATUS_PARKED, confirmation.Order.Status)
	assert.Equal(t, uint64(11), tm.orderEventCount)

	// Send a new normal order to unpark the pegged order
	limitOrder2 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "user2", 1000, 80)
	require.NotEmpty(t, limitOrder2)
	require.Equal(t, types.Order_STATUS_ACTIVE, confirmation.Order.Status)
	assert.Equal(t, uint64(13), tm.orderEventCount)

	// Fill that order to park the pegged order
	limitOrder3 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "user1", 1000, 80)
	require.NotEmpty(t, limitOrder3)
	require.Equal(t, types.Order_STATUS_PARKED, confirmation.Order.Status)
	assert.Equal(t, uint64(16), tm.orderEventCount)
}

func testPeggedOrderRepricing(t *testing.T) {
	// Create the market
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)

	var (
		buyPrice  uint64 = 90
		sellPrice uint64 = 110
		midPrice         = (sellPrice + buyPrice) / 2
	)

	tests := []struct {
		reference      types.PeggedReference
		side           types.Side
		offset         int64
		expectedPrice  uint64
		expectingError string
	}{
		{
			reference:     types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
			side:          types.Side_SIDE_BUY,
			offset:        -3,
			expectedPrice: buyPrice - 3,
		},
		{
			reference:      types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
			side:           types.Side_SIDE_BUY,
			offset:         3,
			expectingError: "can't have a positive offset on Buy orders",
		},
		{
			reference:     types.PeggedReference_PEGGED_REFERENCE_MID,
			side:          types.Side_SIDE_BUY,
			offset:        -5,
			expectedPrice: midPrice - 5,
		},
		{
			reference:     types.PeggedReference_PEGGED_REFERENCE_MID,
			side:          types.Side_SIDE_SELL,
			offset:        5,
			expectedPrice: midPrice + 5,
		},
		{
			reference:     types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
			side:          types.Side_SIDE_SELL,
			offset:        5,
			expectedPrice: sellPrice + 5,
		},
		{
			reference:      types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
			side:           types.Side_SIDE_SELL,
			offset:         -5,
			expectingError: "can't have a negative offset on Sell orders",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			// Create market
			tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
				Duration: 1,
			})
			ctx := context.Background()
			tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

			auxParty, auxParty2 := "auxParty", "auxParty2"
			addAccount(tm, "party1")
			addAccount(tm, auxParty)
			addAccount(tm, auxParty2)

			auxOrders := []*types.Order{
				getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
				getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
				getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
				getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
			}
			for _, o := range auxOrders {
				conf, err := tm.market.SubmitOrder(ctx, o)
				require.NoError(t, err)
				require.NotNil(t, conf)
			}
			// leave auction
			now := now.Add(2 * time.Second)
			tm.market.OnChainTimeUpdate(ctx, now)

			// Create buy and sell orders
			sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, buyPrice)
			sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, sellPrice)

			// Create pegged order
			order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, test.side, "party1", 10, 0)
			order.PeggedOrder = &types.PeggedOrder{Reference: test.reference, Offset: test.offset}
			conf, err := tm.market.SubmitOrder(context.Background(), &order)
			if msg := test.expectingError; msg != "" {
				require.Error(t, err, msg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedPrice, conf.Order.Price)
			}
		})
	}
}

func testPeggedOrderExpiring(t *testing.T) {
	// Create the market
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)

	tm := getTestMarket(t, now, closingAt, nil, nil)
	addAccount(tm, "party")

	// Create buy and sell orders
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party", 1, 100)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party", 1, 200)

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
		addAccount(tm, test.party)

		order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, 0, types.Side_SIDE_BUY, test.party, 10, 150)
		order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
		order.ExpiresAt = test.expiration.UnixNano()
		_, err := tm.market.SubmitOrder(context.Background(), &order)
		require.NoError(t, err)
	}
	assert.Equal(t, len(expirations), tm.market.GetPeggedOrderCount())

	orders, err := tm.market.RemoveExpiredOrders(context.Background(), now.Add(25*time.Minute).UnixNano())
	require.NoError(t, err)
	assert.Equal(t, 2, len(orders))
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount(), "1 order should still be in the market")
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
// in createdAt order. Test this by creating a pegged order and repricing to make it lose it's time order
func testPeggedOrderCanDeleteAfterLostPriority(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)

	addAccount(tm, "party1")

	// Place trades so we have a valid BEST_BID
	buyOrder1 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 100)
	require.NotNil(t, buyOrder1)

	// Place the pegged order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Place a normal limit order behind the pegged order
	buyOrder2 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	require.NotNil(t, buyOrder2)

	// Amend first order to move pegged
	amendOrder(t, tm, "party1", buyOrder1, 0, 101, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0, true)
	// Amend again to make the pegged order reprice behind the second limit order
	amendOrder(t, tm, "party1", buyOrder1, 0, 100, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0, true)

	// Try to delete the pegged order
	cancelconf, _ := tm.market.CancelOrder(context.TODO(), "party1", order.Id)
	assert.NotNil(t, cancelconf)
	assert.Equal(t, types.Order_STATUS_CANCELLED, cancelconf.Order.Status)
}

// If we amend an order that is parked and not in auction we need to see if the amendment has caused the
// order to be unparkable. If so we will have to put it back on the live book.
func testPeggedOrderAmendParkedToLive(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 10),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 10),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		assert.NoError(t, err)
		assert.NotNil(t, conf)
	}

	// Place the pegged order which will be parked
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we can reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedOffset = &wrapperspb.Int64Value{Value: -5}
	amended, err := tm.market.AmendOrder(ctx, amend)
	require.NotNil(t, amended)
	assert.Equal(t, int64(-5), amended.Order.PeggedOrder.Offset)
	assert.NoError(t, err)

	// Check we should have no parked orders
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

// Amend a parked order but the order remains parked
func testPeggedOrderAmendParkedStayParked(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will be parked
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we can reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedOffset = &wrapperspb.Int64Value{Value: -15}
	amended, err := tm.market.AmendOrder(ctx, amend)
	require.NotNil(t, amended)
	assert.Equal(t, int64(-15), amended.Order.PeggedOrder.Offset)
	assert.NoError(t, err)

	// Check we should have no parked orders
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

// Take a valid live order and force it to be parked by amending it
func testPeggedOrderAmendForcesPark(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedOffset = &wrapperspb.Int64Value{Value: -15}
	amended, err := tm.market.AmendOrder(ctx, amend)
	require.NotNil(t, amended)
	assert.NoError(t, err)

	// Order should be parked
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.Order_STATUS_PARKED, amended.Order.Status)
}

func testPeggedOrderAmendDuringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(ctx)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedOffset = &wrapperspb.Int64Value{Value: -5}
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.Order_STATUS_PARKED, amended.Order.Status)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testPeggedOrderAmendReference(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 10),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 10),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)
	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedReference = types.PeggedReference_PEGGED_REFERENCE_MID
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.Order_STATUS_ACTIVE, amended.Order.Status)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReference_PEGGED_REFERENCE_MID, amended.Order.PeggedOrder.Reference)
}

func testPeggedOrderAmendReferenceInAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(ctx)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedReference = types.PeggedReference_PEGGED_REFERENCE_MID
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.Order_STATUS_PARKED, amended.Order.Status)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReference_PEGGED_REFERENCE_MID, amended.Order.PeggedOrder.Reference)
}

func testPeggedOrderAmendMultipleInAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")

	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(ctx)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedReference = types.PeggedReference_PEGGED_REFERENCE_MID
	amend.TimeInForce = types.Order_TIME_IN_FORCE_GTT
	amend.ExpiresAt = &types.Timestamp{Value: 20000000000}
	amended, err := tm.market.AmendOrder(ctx, amend)
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.Order_STATUS_PARKED, amended.Order.Status)
	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReference_PEGGED_REFERENCE_MID, amended.Order.PeggedOrder.Reference)
	assert.Equal(t, types.Order_TIME_IN_FORCE_GTT, amended.Order.TimeInForce)
}

func testPeggedOrderAmendMultiple(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 10),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 10),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
	require.NotNil(t, sellOrder)

	// leave opening auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Place the pegged order which will park it
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Amend offset so we cannot reprice
	amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIME_IN_FORCE_UNSPECIFIED, 0)
	amend.PeggedReference = types.PeggedReference_PEGGED_REFERENCE_MID
	amend.TimeInForce = types.Order_TIME_IN_FORCE_GTT
	amend.ExpiresAt = &types.Timestamp{Value: 20000000000}
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	require.NotNil(t, amended)
	assert.NoError(t, err)

	assert.Equal(t, types.Order_STATUS_ACTIVE, amended.Order.Status)
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, types.PeggedReference_PEGGED_REFERENCE_MID, amended.Order.PeggedOrder.Reference)
	assert.Equal(t, types.Order_TIME_IN_FORCE_GTT, amended.Order.TimeInForce)
}

func testPeggedOrderMidPriceCalc(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, "party1")
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
	buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	require.NotNil(t, buyOrder)
	sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 110)
	require.NotNil(t, sellOrder)
	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)

	// Place the pegged orders
	order1 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -20}
	confirmation1, err := tm.market.SubmitOrder(context.Background(), &order1)
	require.NotNil(t, confirmation1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(80), confirmation1.Order.Price)

	order2 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 10)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: +20}
	confirmation2, err := tm.market.SubmitOrder(context.Background(), &order2)
	require.NotNil(t, confirmation2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(120), confirmation2.Order.Price)

	// Make the mid price wonky (needs rounding)
	buyOrder2 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 91)
	require.NotNil(t, buyOrder2)

	// Check the pegged orders have reprices properly
	assert.Equal(t, uint64(81), confirmation1.Order.Price)  // Buy price gets rounded up
	assert.Equal(t, uint64(120), confirmation2.Order.Price) // Sell price gets rounded down
}

func TestPeggedOrderUnparkAfterLeavingAuctionWithNoFunds2772(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	addAccount(tm, "party2")
	addAccount(tm, "party3")
	addAccount(tm, "party4")
	auxParty := "auxParty"
	addAccount(tm, auxParty)

	//Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
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

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	buyPeggedOrder := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party1", 1000000000000, 0)
	buyPeggedOrder.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	confirmation1, err := tm.market.SubmitOrder(ctx, &buyPeggedOrder)
	assert.NotNil(t, confirmation1)
	assert.Equal(t, confirmation1.Order.Status, types.Order_STATUS_PARKED)
	assert.NoError(t, err)

	sellPeggedOrder := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party1", 1000000000000, 0)
	sellPeggedOrder.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: +10}
	confirmation2, err := tm.market.SubmitOrder(ctx, &sellPeggedOrder)
	assert.NotNil(t, confirmation2)
	assert.Equal(t, confirmation2.Order.Status, types.Order_STATUS_PARKED)
	assert.NoError(t, err)

	sellOrder1 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party2", 4, 2000)
	confirmation3, err := tm.market.SubmitOrder(ctx, &sellOrder1)
	assert.NotNil(t, confirmation3)
	assert.NoError(t, err)

	tm.market.LeaveAuction(ctx, closingAt)

	buyOrder1 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_BUY, "party3", 100, 6500)
	confirmation4, err := tm.market.SubmitOrder(ctx, &buyOrder1)
	assert.NotNil(t, confirmation4)
	assert.NoError(t, err)

	sellOrder2 := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, 0, types.Side_SIDE_SELL, "party4", 20, 7000)
	confirmation5, err := tm.market.SubmitOrder(ctx, &sellOrder2)
	assert.NotNil(t, confirmation5)
	assert.NoError(t, err)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

// test for issue 787,
// segv when an GTT order is cancelled, then expires
func TestOrderBookSimple_CancelGTTOrderThenRunExpiration(t *testing.T) {
	now := time.Unix(5, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()
	defer tm.ctrl.Finish()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order01", types.Side_SIDE_BUY, "aaa", 10, 100)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	cncl, err := tm.market.CancelOrder(ctx, o1.PartyId, o1.Id)
	require.NoError(t, err)
	require.NotNil(t, cncl)
	assert.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())

	orders, err := tm.market.RemoveExpiredOrders(context.Background(), now.Add(10*time.Second).UnixNano())
	require.NoError(t, err)
	require.Len(t, orders, 0)
	assert.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())
}

func TestGTTExpiredNotFilled(t *testing.T) {
	now := time.Unix(5, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()
	defer tm.ctrl.Finish()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order01", types.Side_SIDE_SELL, "aaa", 10, 100)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	// then remove expired, set 1 sec after order exp time.
	orders, err := tm.market.RemoveExpiredOrders(context.Background(), now.Add(10*time.Second).UnixNano())
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, types.Order_STATUS_EXPIRED, orders[0].Status)
}

func TestGTTExpiredPartiallyFilled(t *testing.T) {
	now := time.Unix(5, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, &types.AuctionDuration{
		Duration: 1,
	})
	defer tm.ctrl.Finish()
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx := context.Background()
	tm.market.OnMarketAuctionMinimumDurationUpdate(ctx, time.Second)

	auxParty, auxParty2 := "auxParty", "auxParty2"
	addAccount(tm, auxParty)
	addAccount(tm, auxParty2)

	auxOrders := []*types.Order{
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 1000000),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux1", types.Side_SIDE_SELL, auxParty, 1, 100),
		getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "aux2", types.Side_SIDE_BUY, auxParty2, 1, 100),
	}
	for _, o := range auxOrders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NoError(t, err)
		require.NotNil(t, conf)
	}
	// leave auction
	now = now.Add(2 * time.Second)
	tm.market.OnChainTimeUpdate(ctx, now)
	addAccount(tm, "aaa")
	addAccount(tm, "bbb")

	// We probably don't need these orders anymore, but they don't do any harm
	//Assure liquidity auction won't be triggered
	tm.market.OnMarketLiquidityTargetStakeTriggeringRatio(context.Background(), 0)
	alwaysOnBid := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnBid", types.Side_SIDE_BUY, auxParty, 1, 1)
	conf, err := tm.market.SubmitOrder(context.Background(), alwaysOnBid)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	alwaysOnAsk := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "alwaysOnAsk", types.Side_SIDE_SELL, auxParty, 1, 10000)
	conf, err = tm.market.SubmitOrder(context.Background(), alwaysOnAsk)
	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, types.Order_STATUS_ACTIVE, conf.Order.Status)

	// place expiring order
	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order01", types.Side_SIDE_SELL, "aaa", 10, 100)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	// add matching order
	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order02", types.Side_SIDE_BUY, "bbb", 1, 100)
	o2.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NoError(t, err)
	require.NotNil(t, o2conf)

	// then remove expired, set 1 sec after order exp time.
	orders, err := tm.market.RemoveExpiredOrders(context.Background(), now.Add(10*time.Second).UnixNano())
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, types.Order_STATUS_EXPIRED, orders[0].Status)
	assert.Equal(t, o1.Id, orders[0].Id)
}

func TestOrderBook_RemoveExpiredOrders(t *testing.T) {
	now := time.Unix(5, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()
	defer tm.ctrl.Finish()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	someTimeLater := now.Add(100 * time.Second)

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order01", types.Side_SIDE_SELL, "aaa", 1, 1)
	o1.ExpiresAt = someTimeLater.UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	o2 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order02", types.Side_SIDE_SELL, "aaa", 99, 3298)
	o2.ExpiresAt = someTimeLater.UnixNano() + 1
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NoError(t, err)
	require.NotNil(t, o2conf)

	o3 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order03", types.Side_SIDE_SELL, "aaa", 19, 771)
	o3.ExpiresAt = someTimeLater.UnixNano()
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NoError(t, err)
	require.NotNil(t, o3conf)

	o4 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order04", types.Side_SIDE_SELL, "aaa", 7, 1000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NoError(t, err)
	require.NotNil(t, o4conf)

	o5 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order05", types.Side_SIDE_SELL, "aaa", 99999, 199)
	o5.ExpiresAt = someTimeLater.UnixNano()
	o5conf, err := tm.market.SubmitOrder(ctx, o5)
	require.NoError(t, err)
	require.NotNil(t, o5conf)

	o6 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order06", types.Side_SIDE_SELL, "aaa", 100, 100)
	o6conf, err := tm.market.SubmitOrder(ctx, o6)
	require.NoError(t, err)
	require.NotNil(t, o6conf)

	o7 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order07", types.Side_SIDE_SELL, "aaa", 9999, 41)
	o7.ExpiresAt = someTimeLater.UnixNano() + 9999
	o7conf, err := tm.market.SubmitOrder(ctx, o7)
	require.NoError(t, err)
	require.NotNil(t, o7conf)

	o8 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order08", types.Side_SIDE_SELL, "aaa", 1, 1)
	o8.ExpiresAt = someTimeLater.UnixNano() - 9999
	o8conf, err := tm.market.SubmitOrder(ctx, o8)
	require.NoError(t, err)
	require.NotNil(t, o8conf)

	o9 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order09", types.Side_SIDE_SELL, "aaa", 12, 65)
	o9conf, err := tm.market.SubmitOrder(ctx, o9)
	require.NoError(t, err)
	require.NotNil(t, o9conf)

	o10 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order10", types.Side_SIDE_SELL, "aaa", 1, 1)
	o10.ExpiresAt = someTimeLater.UnixNano() - 1
	o10conf, err := tm.market.SubmitOrder(ctx, o10)
	require.NoError(t, err)
	require.NotNil(t, o10conf)

	expired, err := tm.market.RemoveExpiredOrders(context.Background(), someTimeLater.UnixNano())
	assert.NoError(t, err)
	assert.Len(t, expired, 5)
}

func Test2965EnsureLPOrdersAreNotCancelleableWithCancelAll(t *testing.T) {
	now := time.Unix(10, 0)
	closingAt := time.Unix(1000000000, 0)
	ctx := context.Background()

	mktCfg := getMarket(closingAt, defaultPriceMonitorSettings, &types.AuctionDuration{
		Duration: 10000,
	})
	mktCfg.Fees = &types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      "0.001",
			InfrastructureFee: "0.0005",
			MakerFee:          "0.00025",
		},
	}
	mktCfg.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: 0.001,
			Tau:                   0.00011407711613050422,
			Params: &types.LogNormalModelParams{
				Mu:    0,
				R:     0.016,
				Sigma: 20,
			},
		},
	}

	tm := newTestMarket(t, now).Run(ctx, mktCfg)
	tm.StartOpeningAuction().
		WithAccountAndAmount("trader-0", 1000000).
		WithAccountAndAmount("trader-1", 1000000).
		WithAccountAndAmount("trader-2", 10000000000).
		// provide stake as well but will cancel
		WithAccountAndAmount("trader-2-bis", 10000000000).
		WithAccountAndAmount("trader-3", 1000000).
		WithAccountAndAmount("trader-4", 1000000)

	tm.market.OnSuppliedStakeToObligationFactorUpdate(1.0)
	tm.market.OnChainTimeUpdate(ctx, now)

	orderParams := []struct {
		id        string
		size      uint64
		side      types.Side
		tif       types.Order_TimeInForce
		pegRef    types.PeggedReference
		pegOffset int64
	}{
		{"trader-4", 1, types.Side_SIDE_BUY, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_BID, -2000},
		{"trader-3", 1, types.Side_SIDE_SELL, types.Order_TIME_IN_FORCE_GTC, types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, 1000},
	}
	traderA, traderB := orderParams[0], orderParams[1]

	tpl := OrderTemplate{
		Type: types.Order_TYPE_LIMIT,
	}
	var orders = []*types.Order{
		// Limit Orders
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5500 + traderA.pegOffset), // 3500
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       uint64(5000 - traderB.pegOffset), // 4000
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-1",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        10,
			Remaining:   10,
			Price:       5500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GFA,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       5000,
			Side:        types.Side_SIDE_SELL,
			PartyId:     "trader-2",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        100,
			Remaining:   100,
			Price:       3500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),
		tpl.New(types.Order{
			Size:        20,
			Remaining:   20,
			Price:       8500,
			Side:        types.Side_SIDE_BUY,
			PartyId:     "trader-0",
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		}),

		// Pegged Orders
		tpl.New(types.Order{
			PartyId:     traderA.id,
			Side:        traderA.side,
			Size:        traderA.size,
			Remaining:   traderA.size,
			TimeInForce: traderA.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderA.pegRef,
				Offset:    traderA.pegOffset,
			},
		}),
		tpl.New(types.Order{
			PartyId:     traderB.id,
			Side:        traderB.side,
			Size:        traderB.size,
			Remaining:   traderB.size,
			TimeInForce: traderB.tif,
			PeggedOrder: &types.PeggedOrder{
				Reference: traderB.pegRef,
				Offset:    traderB.pegOffset,
			},
		}),
	}

	tm.WithSubmittedOrders(t, orders...)

	// Add a LPSubmission
	// this is a log of stake, enough to cover all
	// the required stake for the market
	lp := &types.LiquidityProvisionSubmission{
		MarketId:         tm.market.GetID(),
		CommitmentAmount: 2000000,
		Fee:              "0.01",
		Reference:        "THIS-IS-LP",
		Sells: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 10, Offset: 2},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 13, Offset: 1},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 10, Offset: -1},
			{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Proportion: 13, Offset: -15},
		},
	}

	// Leave the auction
	tm.market.OnChainTimeUpdate(ctx, now.Add(10001*time.Second))

	require.NoError(t, tm.market.SubmitLiquidityProvision(ctx, lp, "trader-2", "id-lp"))
	assert.Equal(t, 1, tm.market.GetLPSCount())

	tm.market.OnChainTimeUpdate(ctx, now.Add(10011*time.Second))

	newOrder := tpl.New(types.Order{
		MarketId:    tm.market.GetID(),
		Size:        20,
		Remaining:   20,
		Price:       10250,
		Side:        types.Side_SIDE_SELL,
		PartyId:     "trader-2",
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	})

	tm.events = nil
	cnf, err := tm.market.SubmitOrder(ctx, newOrder)
	assert.NoError(t, err)
	assert.Len(t, cnf.Trades, 0)

	// now we cancel all orders, but should get only 1 cancellation
	// and the ID should be newOrder
	tm.events = nil
	cancelCnf, err := tm.market.CancelAllOrders(ctx, "trader-2")
	assert.NoError(t, err)
	assert.Len(t, cancelCnf, 2)

	t.Run("ExpectedOrderCancelled", func(t *testing.T) {
		// one event is sent, this is a rejected event from
		// the first order we try to place, the party does
		// not have enough funds
		expectedIds := map[string]bool{
			newOrder.Id:  false,
			orders[3].Id: false,
		}

		require.Len(t, cancelCnf, len(expectedIds))

		for _, o := range cancelCnf {
			_, ok := expectedIds[o.Order.Id]
			if !ok {
				t.Errorf("unexpected cancelled order: %v", o.Order.Id)
			}
			expectedIds[o.Order.Id] = true
		}

		for id, ok := range expectedIds {
			if !ok {
				t.Errorf("expected order to be cancelled was not cancelled: %v", id)
			}
		}
	})

}
