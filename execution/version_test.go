package execution_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestVersioning(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	price := uint64(100)
	size := uint64(100)

	addAccount(tm, party1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Type:        types.Order_TYPE_LIMIT,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Id:          "someid",
		Side:        types.Side_SIDE_BUY,
		PartyId:     party1,
		MarketId:    tm.market.GetID(),
		Size:        size,
		Price:       num.NewUint(price),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Create an order and check version is set to 1
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.EqualValues(t, confirmation.Order.Version, uint64(1))

	orderID := confirmation.Order.Id

	// Amend price up, check version moves to 2
	amend := &types.OrderAmendment{
		OrderId:  orderID,
		MarketId: tm.market.GetID(),
		Price:    num.NewUint(price + 1),
	}

	amendment, err := tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend price down, check version moves to 3
	amend.Price = num.NewUint(price - 1)
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend quantity up, check version moves to 4
	amend.Price = nil
	amend.SizeDelta = 1
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend quantity down, check version moves to 5
	amend.SizeDelta = -2
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Flip to GTT, check version moves to 6
	amend.TimeInForce = types.Order_TIME_IN_FORCE_GTT
	exp := now.UnixNano() + 100000000000
	amend.ExpiresAt = &exp
	amend.SizeDelta = 0
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Update expiry time, check version moves to 7
	exp = now.UnixNano() + 100000000000
	amend.ExpiresAt = &exp
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Flip back GTC, check version moves to 8
	amend.TimeInForce = types.Order_TIME_IN_FORCE_GTC
	amend.ExpiresAt = nil
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1)
	assert.NotNil(t, amendment)
	assert.NoError(t, err)
}
