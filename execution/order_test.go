package execution_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestOrderBufferOutputCount(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)
	tm.broker.EXPECT().Send(gomock.Any()).MinTimes(11)

	orderBuy := &types.Order{
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTC,
		Status:      types.Order_STATUS_ACTIVE,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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

	// Cancel it (generates one order message)
	cancelled, err := tm.market.CancelOrderByID(confirmation.Order.Id)
	assert.NotNil(t, cancelled, "cancelled freshly submitted order")
	assert.NoError(t, err)
	assert.EqualValues(t, confirmation.Order.Id, cancelled.Order.Id)

	// Create a new order (generates one order message)
	orderAmend.Id = "amendingorder"
	orderAmend.Reference = "amendingorderreference"
	confirmation, err = tm.market.SubmitOrder(context.TODO(), &orderAmend)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	amend := &types.OrderAmendment{
		MarketID: tm.market.GetID(),
		PartyID:  party1,
		OrderID:  orderAmend.Id,
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

	// Amend TIF -> GTT (generates one order message)
	amend.SizeDelta = 0
	amend.TimeInForce = types.Order_TIF_GTT
	amend.ExpiresAt = &types.Timestamp{Value: now.UnixNano() + 100000000000}
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend TIF -> GTC (generates one order message)
	amend.TimeInForce = types.Order_TIF_GTC
	amend.ExpiresAt = nil
	amendConf, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend ExpiresAt (generates two order messages)
	amend.TimeInForce = types.Order_TIF_GTT
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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

	amend := &types.OrderAmendment{
		OrderID:  orderID,
		PartyID:  confirmation.GetOrder().GetPartyID(),
		MarketID: confirmation.GetOrder().GetMarketID(),
		Price:    &types.Price{Value: 101},
	}
	amended, err := tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	amend = &types.OrderAmendment{
		OrderID:   orderID,
		PartyID:   confirmation.GetOrder().GetPartyID(),
		MarketID:  confirmation.GetOrder().GetMarketID(),
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
	cancelOrder := &types.OrderCancellation{
		OrderID:  confirmation.GetOrder().Id,
		MarketID: confirmation.GetOrder().MarketID,
		PartyID:  party2,
	}
	cancelconf, err := tm.market.CancelOrder(context.TODO(), cancelOrder.PartyID, cancelOrder.OrderID)
	assert.Nil(t, cancelconf)
	assert.Error(t, err, types.ErrInvalidPartyID)
}

func TestMarkPriceUpdateAfterPartialFill(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIF_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_IOC,
		Id:          "someid",
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)

	orderBuy := &types.Order{
		CreatedAt:   10000000000,
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIF_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
	tm.market.OnChainTimeUpdate(time.Unix(10, 100))

	amend := &types.OrderAmendment{
		OrderID:     buyConfirmation.GetOrder().GetId(),
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		ExpiresAt:   &types.Timestamp{Value: 10000000010},
		TimeInForce: types.Order_TIF_GTT,
	}
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	// Validate that the mark price has been updated
	assert.EqualValues(t, amended.Order.TimeInForce, types.Order_TIF_GTT)
	assert.EqualValues(t, amended.Order.Status, types.Order_STATUS_EXPIRED)
	assert.EqualValues(t, amended.Order.CreatedAt, 10000000000)
	assert.EqualValues(t, amended.Order.ExpiresAt, 10000000010)
	assert.EqualValues(t, amended.Order.UpdatedAt, 10000000100)
}

func TestAmendPartialFillCancelReplace(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		TimeInForce: types.Order_TIF_GTC,
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_IOC,
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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

	amend := &types.OrderAmendment{
		OrderID:  buyConfirmation.GetOrder().GetId(),
		PartyID:  party1,
		MarketID: tm.market.GetID(),
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTC,
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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

	// Send an aend but use the wrong partyID
	amend := &types.OrderAmendment{
		OrderID:  confirmation.GetOrder().GetId(),
		PartyID:  party2,
		MarketID: confirmation.GetOrder().GetMarketID(),
		Price:    &types.Price{Value: 101},
	}
	amended, err := tm.market.AmendOrder(context.Background(), amend)
	assert.Nil(t, amended)
	assert.Error(t, err, types.ErrInvalidPartyID)
}

func TestPartialFilledWashTrade(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderSell1 := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIF_GTC,
		Side:        types.Side_SIDE_SELL,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GTC,
		Side:        types.Side_SIDE_SELL,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
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
		TimeInForce: types.Order_TIF_GTC,
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        30,
		Price:       60,
		Remaining:   30,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	confirmation, err = tm.market.SubmitOrder(context.Background(), orderBuy1)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_REJECTED)
	assert.Equal(t, confirmation.Order.Remaining, uint64(15))
}

func amendOrder(t *testing.T, tm *testMarket, party string, orderID string, sizeDelta int64, price uint64,
	tif types.Order_TimeInForce, expiresAt int64, pass bool) {
	amend := &types.OrderAmendment{
		OrderID:     orderID,
		PartyID:     party,
		MarketID:    tm.market.GetID(),
		SizeDelta:   sizeDelta,
		TimeInForce: tif,
	}

	if price > 0 {
		amend.Price = &types.Price{Value: price}
	}

	if expiresAt > 0 {
		amend.ExpiresAt = &types.Timestamp{Value: expiresAt}
	}

	amended, err := tm.market.AmendOrder(context.Background(), amend)
	if pass {
		assert.NotNil(t, amended)
		assert.NoError(t, err)
	}
}

func sendOrder(t *testing.T, tm *testMarket, now *time.Time, orderType types.Order_Type, tif types.Order_TimeInForce, expiresAt int64, side types.Side, party string,
	size uint64, price uint64) string {
	order := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        orderType,
		TimeInForce: tif,
		Side:        side,
		PartyID:     party,
		MarketID:    tm.market.GetID(),
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	addAccount(tm, "party2")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// test_AmendMarketOrderFail
	orderId := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 100) // 1 - a8
	orderId = sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 110)  // 1 - a8
	orderId = sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 120)  // 1 - a8
	orderId = sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party2", 40, 50)    // 1 - a8
	amendOrder(t, tm, "party2", orderId, 0, 500, types.Order_TIF_UNSPECIFIED, 0, true)
}
