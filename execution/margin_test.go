package execution_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMargins(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt)
	price := uint64(100)
	size := uint64(100)

	addAccount(tm, party1)
	tm.orderStore.EXPECT().Add(gomock.Any()).AnyTimes()
	tm.accountBuf.EXPECT().Add(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_LIMIT,
		TimeInForce: types.Order_TIF_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyID:     party1,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       price,
		Remaining:   size,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Create an order to amend
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	if !assert.NoError(t, err) {
		t.Fatalf("Error: %v", err)
	}
	if !assert.NotNil(t, confirmation) {
		t.Fatal("SubmitOrder confirmation was nil, but no error.")
	}

	orderID := confirmation.GetOrder().Id

	// Amend size up
	amend := &types.OrderAmendment{
		OrderID:   orderID,
		MarketID:  tm.market.GetID(),
		PartyID:   party1,
		SizeDelta: int64(10000),
	}
	amendment, err := tm.market.AmendOrder(context.TODO(), amend)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend price and size up to breach margin
	amend.SizeDelta = 1000000000
	amend.Price = &types.Price{Value: 1000000000}
	amendment, err = tm.market.AmendOrder(context.TODO(), amend)
	assert.Nil(t, amendment)
	assert.Error(t, err)
}
