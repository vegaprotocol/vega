package execution_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	tm.market.OnChainTimeUpdate(context.Background(), time.Unix(10, 100))

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

func getAmend(market string, party string, orderID string, sizeDelta int64, price uint64,
	tif types.Order_TimeInForce, expiresAt int64) *types.OrderAmendment {

	amend := &types.OrderAmendment{
		OrderID:     orderID,
		PartyID:     party,
		MarketID:    market,
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

func getOrder(t *testing.T, tm *testMarket, now *time.Time, orderType types.Order_Type, tif types.Order_TimeInForce,
	expiresAt int64, side types.Side, party string, size uint64, price uint64) types.Order {
	order := types.Order{
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
	return order
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

func TestUnableToAmendGFAGFN(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// test_AmendMarketOrderFail
	orderId := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 100)
	amendOrder(t, tm, "party1", orderId, 0, 0, types.Order_TIF_GFA, 0, false)
	amendOrder(t, tm, "party1", orderId, 0, 0, types.Order_TIF_GFN, 0, false)

	orderId2 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GFN, 0, types.Side_SIDE_SELL, "party1", 10, 100)
	amendOrder(t, tm, "party1", orderId2, 0, 0, types.Order_TIF_GTC, 0, false)
	amendOrder(t, tm, "party1", orderId2, 0, 0, types.Order_TIF_GFA, 0, false)

	// EnterAuction should actually trigger an auction here...
	tm.mas.StartPriceAuction(now, &types.AuctionDuration{
		Duration: closeSec / 10, // some time in the future, before closing
	})
	tm.market.EnterAuction(context.Background())
	orderId3 := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GFA, 0, types.Side_SIDE_SELL, "party1", 10, 100)
	amendOrder(t, tm, "party1", orderId3, 0, 0, types.Order_TIF_GTC, 0, false)
	amendOrder(t, tm, "party1", orderId3, 0, 0, types.Order_TIF_GFN, 0, false)
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
}

func testPeggedOrderAmendToMoveReference(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Place 2 orders to create valid reference prices
	bestBidOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 110)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Amend best bid price
	amendOrder(t, tm, "party1", bestBidOrder, 0, 88, types.Order_TIF_UNSPECIFIED, 0, true)
	amendOrder(t, tm, "party1", bestBidOrder, 0, 86, types.Order_TIF_UNSPECIFIED, 0, true)
}

func testPeggedOrderFilledOrder(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	addAccount(tm, "party2")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 110)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	// Place a sell MARKET order to fill the buy orders
	sendOrder(t, tm, &now, types.Order_TYPE_MARKET, types.Order_TIF_IOC, 0, types.Side_SIDE_SELL, "party2", 2, 0)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 0, tm.market.GetPeggedOrderCount())
}

func testParkedOrdersAreUnparkedWhenPossible(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	addAccount(tm, "party2")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 5)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 1)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
	_, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	assert.Equal(t, 1, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())

	// Send a higher buy price order to move the BEST BID price up
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 50)

	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
}

func testPeggedOrdersLeavingAuction(t *testing.T) {
	now := time.Unix(10, 0)
	auctionClose := now.Add(101 * time.Second)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -10}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARKED)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	// During an auction all pegged orders are parked so we don't add them to the list
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())

	// Update the time to force the auction to end
	tm.market.OnChainTimeUpdate(ctx, auctionClose)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrdersEnteringAuction(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Place 2 orders to create valid reference prices
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 100)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -10}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_ACTIVE)
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())

	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	assert.Equal(t, 1, tm.market.GetPeggedOrderCount())
	assert.Equal(t, 0, tm.market.GetParkedOrderCount())
}

func testPeggedOrderAddWithNoMarketPrice(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Place a valid pegged order which will be parked
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
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
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 100)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 102)

	// Place a valid pegged order which will be added to the order book
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
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
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 1, 90)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 1, 110)

	md := tm.market.GetMarketData()
	assert.Equal(t, uint64(100), md.MidPrice)
	// Place a valid pegged order which will be added to the order book
	// This order will cause the MID price to move and thus a reprice multiple times until it settles
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
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
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
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
	tm := getTestMarket(t, now, closingAt, nil)
	ctx := context.Background()

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Move into auction
	tm.mas.StartOpeningAuction(now, &types.AuctionDuration{Duration: 100})
	tm.market.EnterAuction(ctx)

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	assert.NotNil(t, confirmation)
	assert.Equal(t, confirmation.Order.Status, types.Order_STATUS_PARKED)
	assert.NoError(t, err)

	buy := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 90)
	confirmation, err = tm.market.SubmitOrder(context.Background(), &buy)
	require.NotNil(t, confirmation)
	assert.NoError(t, err)

	require.NotNil(t, buy)
	sell := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 110)
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
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

func testPeggedOrderTIFs(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Pegged order must be a LIMIT order
	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -3}

	// Only allowed GTC
	order.Type = types.Order_TYPE_LIMIT
	order.TimeInForce = types.Order_TIF_GTC
	confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// and GTT
	order.TimeInForce = types.Order_TIF_GTT
	order.ExpiresAt = now.UnixNano() + 1000000000
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// but not IOC
	order.ExpiresAt = 0
	order.TimeInForce = types.Order_TIF_IOC
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)

	// or FOK
	order.TimeInForce = types.Order_TIF_FOK
	confirmation, err = tm.market.SubmitOrder(context.Background(), &order)
	assert.Nil(t, confirmation)
	assert.Error(t, err)
}

func testPeggedOrderBuys(t *testing.T) {
	now := time.Unix(10, 0)
	closeSec := int64(10000000000)
	closingAt := time.Unix(closeSec, 0)
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 100)

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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 100)

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

	// BEST ASK peg mudst be >= 0
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx := context.Background()

	for _, acc := range []string{"buyer", "seller", "pegged"} {
		addAccount(tm, acc)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	}

	buy := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "buyer", 10, 4)
	_, err := tm.market.SubmitOrder(ctx, &buy)
	require.NoError(t, err)

	sell := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "seller", 10, 8)
	_, err = tm.market.SubmitOrder(ctx, &sell)
	require.NoError(t, err)

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "pegged", 10, 4)
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
	tm := getTestMarket(t, now, closingAt, nil)

	addAccount(tm, "party1")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	ctx := context.Background()

	for _, acc := range []string{"buyer", "seller", "pegged"} {
		addAccount(tm, acc)
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	}

	buy := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "buyer", 10, 4)
	_, err := tm.market.SubmitOrder(ctx, &buy)
	require.NoError(t, err)

	sell := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "seller", 10, 8)
	_, err = tm.market.SubmitOrder(ctx, &sell)
	require.NoError(t, err)

	order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "pegged", 10, 4)
	order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_MID, Offset: -5}
	confirmation, err := tm.market.SubmitOrder(ctx, &order)
	require.NoError(t, err)

	amendOrder(t, tm, "buyer", buy.Id, 0, 1, types.Order_TIF_UNSPECIFIED, 0, true)

	assert.Equal(t, types.Order_STATUS_PARKED.String(), confirmation.Order.Status.String())
	//	assert.Equal(t,
	//		types.Order_STATUS_PARKED.String(),
	//		confirmation.Order.Status.String(), "When pegged price below zero (MIDPRICE - OFFSET) <= 0")
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
			expectedPrice: midPrice - 1,
		},
		{
			reference:     types.PeggedReference_PEGGED_REFERENCE_MID,
			side:          types.Side_SIDE_SELL,
			offset:        5,
			expectedPrice: midPrice,
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
			tm := getTestMarket(t, now, closingAt, nil)

			// Create the party
			addAccount(tm, "party")
			tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

			// Create buy and sell orders
			sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party", 1, buyPrice)
			sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party", 1, sellPrice)

			// Create pegged order
			order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, test.side, "party", 10, 0)
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

	tm := getTestMarket(t, now, closingAt, nil)
	addAccount(tm, "party")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// Create buy and sell orders
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party", 1, 100)
	sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party", 1, 200)

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

		order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, test.party, 10, 150)
		order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -10}
		order.ExpiresAt = test.expiration.UnixNano()
		_, err := tm.market.SubmitOrder(context.Background(), &order)
		require.NoError(t, err)
	}
	assert.Equal(t, len(expirations), tm.market.GetPeggedOrderCount())

	orders, err := tm.market.RemoveExpiredOrders(now.Add(25 * time.Minute).UnixNano())
	require.NoError(t, err)
	assert.Empty(t, orders, "expiring pegged orders shouldn't be in the books expiring list")
	assert.Equal(t, 1, tm.market.GetPeggedOrderCount(), "1 order should still be in the market")
}

func TestPeggedOrderAmendForcesUnPark(t *testing.T) {
	/*	now := time.Unix(10, 0)
		closeSec := int64(10000000000)
		closingAt := time.Unix(closeSec, 0)
		tm := getTestMarket(t, now, closingAt, nil)

		addAccount(tm, "party1")
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
		buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
		require.NotNil(t, buyOrder)
		sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
		require.NotNil(t, sellOrder)

		// Place the pegged order which will be parked
		order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 10)
		order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -20}
		confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
		require.Nil(t, confirmation)
		assert.Error(t, err)

		// Amend offset so we can reprice
		amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIF_UNSPECIFIED, 0)
		amend.PeggedOffset = &types.PeggedOffset{Value: -5}
		amended, err := tm.market.AmendOrder(context.Background(), amend)
		require.Nil(t, amended)
		assert.Error(t, err)*/
}

func TestPeggedOrderAmendForcesPark(t *testing.T) {
	/*	now := time.Unix(10, 0)
		closeSec := int64(10000000000)
		closingAt := time.Unix(closeSec, 0)
		tm := getTestMarket(t, now, closingAt, nil)

		addAccount(tm, "party1")
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Place 2 trades so we have a valid BEST_BID+MID+BEST_ASK price
		buyOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_BUY, "party1", 10, 9)
		require.NotNil(t, buyOrder)
		sellOrder := sendOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 11)
		require.NotNil(t, sellOrder)

		// Place the pegged order
		order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 10)
		order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
		confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
		require.Nil(t, confirmation)
		assert.Error(t, err)

		// Amend offset so we cannot reprice
		amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIF_UNSPECIFIED, 0)
		amend.PeggedOffset = &types.PeggedOffset{Value: -15}
		amended, err := tm.market.AmendOrder(context.Background(), amend)
		require.Nil(t, amended)
		assert.Error(t, err)*/
}

func TestPeggedOrderAmendWhileParked(t *testing.T) {
	/*	now := time.Unix(10, 0)
		closeSec := int64(10000000000)
		closingAt := time.Unix(closeSec, 0)
		tm := getTestMarket(t, now, closingAt, nil)

		addAccount(tm, "party1")
		tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

		// Place a pegged order when there is no reference price so it will be parked
		order := getOrder(t, tm, &now, types.Order_TYPE_LIMIT, types.Order_TIF_GTC, 0, types.Side_SIDE_SELL, "party1", 10, 100)
		order.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -3}
		confirmation, err := tm.market.SubmitOrder(context.Background(), &order)
		require.Nil(t, confirmation)
		assert.Error(t, err)

		// Try to amend the offset value
		amend := getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIF_UNSPECIFIED, 0)
		amend.PeggedOffset = &types.PeggedOffset{Value: -5}
		amended, err := tm.market.AmendOrder(context.Background(), amend)
		require.NotNil(t, amended)
		assert.NoError(t, err)

		// Try to amend the reference price
		amend = getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 0, types.Order_TIF_UNSPECIFIED, 0)
		amend.PeggedReference = types.PeggedReference_PEGGED_REFERENCE_MID
		amended, err = tm.market.AmendOrder(context.Background(), amend)
		require.NotNil(t, amended)
		assert.NoError(t, err)

		// Try to amend the size of the order
		amend = getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 1, 0, types.Order_TIF_UNSPECIFIED, 0)
		amended, err = tm.market.AmendOrder(context.Background(), amend)
		require.NotNil(t, amended)
		assert.NoError(t, err)

		// Try to amend the price of the order
		amend = getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 101, types.Order_TIF_UNSPECIFIED, 0)
		amended, err = tm.market.AmendOrder(context.Background(), amend)
		require.Nil(t, amended)
		assert.Error(t, err)

		// Try to amend the time in force of the order
		amend = getAmend(tm.market.GetID(), "party1", confirmation.Order.Id, 0, 101, types.Order_TIF_GTT, 10000000000)
		amended, err = tm.market.AmendOrder(context.Background(), amend)
		require.NotNil(t, amended)
		assert.NoError(t, err)*/
}
