package execution_test

import (
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
	tm := getTestMarket(t, now, closingAt)

	addAccount(tm, party1)
	tm.orderStore.EXPECT().Add(gomock.Any()).Times(11)

	orderBuy := &types.Order{
		Type:        types.Order_LIMIT,
		TimeInForce: types.Order_GTC,
		Status:      types.Order_Active,
		Id:          "someid",
		Side:        types.Side_Buy,
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
	confirmation, err := tm.market.SubmitOrder(orderBuy)
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
	confirmation, err = tm.market.SubmitOrder(&orderAmend)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	amend := &types.OrderAmendment{
		MarketID: tm.market.GetID(),
		PartyID:  party1,
		OrderID:  orderAmend.Id,
	}

	// Amend price down (generates one order message)
	amend.Price = &types.Price{Value: orderBuy.Price - 1}
	amendConf, err := tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend price up (generates one order message)
	amend.Price = &types.Price{Value: orderBuy.Price + 1}
	amendConf, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend size down (generates one order message)
	amend.Price = nil
	amend.SizeDelta = -1
	amendConf, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend size up (generates one order message)
	amend.SizeDelta = +1
	amendConf, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend TIF -> GTT (generates one order message)
	amend.SizeDelta = 0
	amend.TimeInForce = types.Order_GTT
	amend.ExpiresAt = &types.Timestamp{Value: now.UnixNano() + 100000000000}
	amendConf, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend TIF -> GTC (generates one order message)
	amend.TimeInForce = types.Order_GTC
	amend.ExpiresAt = nil
	amendConf, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	// Amend ExpiresAt (generates two order messages)
	amend.TimeInForce = types.Order_GTT
	amend.ExpiresAt = &types.Timestamp{Value: now.UnixNano() + 100000000000}
	amendConf, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)

	amend.ExpiresAt = &types.Timestamp{Value: now.UnixNano() + 200000000000}
	amendConf, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amendConf)
	assert.NoError(t, err)
}

func TestAmendCancelResubmit(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt)

	addAccount(tm, party1)
	tm.orderStore.EXPECT().Add(gomock.Any()).Times(1)
	tm.accountBuf.EXPECT().Add(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Type:        types.Order_LIMIT,
		TimeInForce: types.Order_GTC,
		Id:          "someid",
		Side:        types.Side_Buy,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	orderID := confirmation.GetOrder().Id

	// Amend the price to force a cancel+resubmit to the order book
	tm.orderStore.EXPECT().Add(gomock.Any()).Times(1).Do(func(order types.Order) {
		// Validate that the orderID is the same as before
		assert.EqualValues(t, order.Id, orderID)
		assert.EqualValues(t, order.Price, uint64(101))
		assert.EqualValues(t, order.Remaining, uint64(100))
		assert.EqualValues(t, order.Version, uint64(2))
	})

	amend := &types.OrderAmendment{
		OrderID:  orderID,
		PartyID:  confirmation.GetOrder().GetPartyID(),
		MarketID: confirmation.GetOrder().GetMarketID(),
		Price:    &types.Price{Value: 101},
	}
	amended, err := tm.market.AmendOrder(amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	// Amend the quantity to force a cancel_resubmit to the order book
	tm.orderStore.EXPECT().Add(gomock.Any()).Times(1).Do(func(order types.Order) {
		// Validate that the orderID is the same as before
		assert.EqualValues(t, order.Id, orderID)
		assert.EqualValues(t, order.Version, uint64(3))
		assert.EqualValues(t, order.Price, uint64(101))
		assert.EqualValues(t, order.Remaining, uint64(101))
	})

	amend = &types.OrderAmendment{
		OrderID:   orderID,
		PartyID:   confirmation.GetOrder().GetPartyID(),
		MarketID:  confirmation.GetOrder().GetMarketID(),
		Price:     &types.Price{Value: 101},
		SizeDelta: 1,
	}
	amended, err = tm.market.AmendOrder(amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)
}

func TestCancelWithWrongPartyID(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.orderStore.EXPECT().Add(gomock.Any()).Times(1)
	tm.accountBuf.EXPECT().Add(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Type:        types.Order_LIMIT,
		TimeInForce: types.Order_GTC,
		Id:          "someid",
		Side:        types.Side_Buy,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       100,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Submit the original order
	confirmation, err := tm.market.SubmitOrder(orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)

	// Now attempt to cancel it with the wrong partyID
	cancelOrder := &types.OrderCancellation{
		OrderID:  confirmation.GetOrder().Id,
		MarketID: confirmation.GetOrder().MarketID,
		PartyID:  party2,
	}
	cancelconf, err := tm.market.CancelOrder(cancelOrder)
	assert.Nil(t, cancelconf)
	assert.Error(t, err, types.ErrInvalidPartyID)
}

func TestMarkPriceUpdateAfterPartialFill(t *testing.T) {
	party1 := "party1"
	party2 := "party2"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt)

	addAccount(tm, party1)
	addAccount(tm, party2)
	tm.orderStore.EXPECT().Add(gomock.Any()).AnyTimes()
	tm.accountBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	tm.tradeStore.EXPECT().Add(gomock.Any()).AnyTimes()
	tm.candleStore.EXPECT().AddTrade(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		TimeInForce: types.Order_GTC,
		Id:          "someid",
		Side:        types.Side_Buy,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       10,
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
		Type:        types.Order_LIMIT,
	}
	// Submit the original order
	buyConfirmation, err := tm.market.SubmitOrder(orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	orderSell := &types.Order{
		TimeInForce: types.Order_IOC,
		Id:          "someid",
		Side:        types.Side_Sell,
		PartyID:     party2,
		MarketID:    tm.market.GetID(),
		Size:        50,
		Price:       10,
		Remaining:   50,
		CreatedAt:   now.UnixNano(),
		Reference:   "party2-sell-order",
		Type:        types.Order_MARKET,
	}
	// Submit an opposite order to partially fill
	sellConfirmation, err := tm.market.SubmitOrder(orderSell)
	assert.NotNil(t, sellConfirmation)
	assert.NoError(t, err)

	// Validate that the mark price has been updated
	assert.EqualValues(t, tm.market.GetMarketData().MarkPrice, 10)
}

func TestExpireCancelGTCOrder(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt)

	addAccount(tm, party1)
	tm.orderStore.EXPECT().Add(gomock.Any()).AnyTimes()
	tm.accountBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	tm.tradeStore.EXPECT().Add(gomock.Any()).AnyTimes()
	tm.candleStore.EXPECT().AddTrade(gomock.Any()).AnyTimes()
	tm.candleStore.EXPECT().Flush(gomock.Any(), gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		TimeInForce: types.Order_GTC,
		Id:          "someid",
		Side:        types.Side_Buy,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       10,
		Remaining:   100,
		Reference:   "party1-buy-order",
		Type:        types.Order_LIMIT,
	}
	// Submit the original order
	buyConfirmation, err := tm.market.SubmitOrder(orderBuy)
	assert.NotNil(t, buyConfirmation)
	assert.NoError(t, err)

	// Move the current time forward
	tm.market.OnChainTimeUpdate(time.Unix(10, 100))

	amend := &types.OrderAmendment{
		OrderID:     buyConfirmation.GetOrder().GetId(),
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		ExpiresAt:   &types.Timestamp{Value: 10000000010},
		TimeInForce: types.Order_GTT,
	}
	amended, err := tm.market.AmendOrder(amend)
	assert.NotNil(t, amended)
	assert.NoError(t, err)

	// Validate that the mark price has been updated
	assert.EqualValues(t, amended.Order.TimeInForce, types.Order_GTT)
	assert.EqualValues(t, amended.Order.Status, types.Order_Expired)
	assert.EqualValues(t, amended.Order.CreatedAt, 10000000000)
	assert.EqualValues(t, amended.Order.ExpiresAt, 10000000010)
	assert.EqualValues(t, amended.Order.UpdatedAt, 10000000100)
}
