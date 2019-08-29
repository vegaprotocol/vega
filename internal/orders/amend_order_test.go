package orders_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/vegatime"
	"code.vegaprotocol.io/vega/proto"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	amend = proto.OrderAmendment{
		OrderID: "order_id",
		PartyID: "party",
		Price:   10000,
		Size:    1,
	}
)

type amendMatcher struct {
	e proto.OrderAmendment
}

func TestAmendOrder(t *testing.T) {
	t.Run("Amend order - success", testAmendOrderSuccess)
	t.Run("Amend order - expired", testAmendOrderExpired)
	t.Run("Amend order - not active", testAmendOrderNotActive)
	t.Run("Amend order - invalid payload", testAmendOrderInvalidPayload)
	t.Run("Amend order - time service error", testAmendOrderTimeSvcErr)
}

func testAmendOrderSuccess(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(2 * time.Hour)
	arg := amend
	arg.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	order := proto.Order{
		Id:          arg.OrderID,
		MarketID:    "market",
		PartyID:     arg.PartyID,
		Status:      proto.Order_Active,
		TimeInForce: proto.Order_GTT,
	}
	svc.orderStore.EXPECT().GetByPartyAndId(gomock.Any(), arg.PartyID, arg.OrderID).Times(1).Return(&order, nil)
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	svc.block.EXPECT().AmendOrder(gomock.Any(), amendMatcher{e: arg}).Times(1).Return(true, nil)

	pendingOrder, err := svc.svc.AmendOrder(context.Background(), &arg)
	assert.NotNil(t, pendingOrder)
	assert.NoError(t, err)
}

func testAmendOrderExpired(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(-2 * time.Hour)
	arg := amend
	arg.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	order := proto.Order{
		Id:          arg.OrderID,
		MarketID:    "market",
		PartyID:     arg.PartyID,
		Status:      proto.Order_Active,
		TimeInForce: proto.Order_GTT,
	}
	svc.orderStore.EXPECT().GetByPartyAndId(gomock.Any(), arg.PartyID, arg.OrderID).Times(1).Return(&order, nil)
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)

	pendingOrder, err := svc.svc.AmendOrder(context.Background(), &arg)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
}

func testAmendOrderNotActive(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(2 * time.Hour)
	arg := amend
	arg.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	order := proto.Order{
		Id:       arg.OrderID,
		MarketID: "market",
		PartyID:  arg.PartyID,
		Status:   proto.Order_Expired,
	}
	svc.orderStore.EXPECT().GetByPartyAndId(gomock.Any(), arg.PartyID, arg.OrderID).Times(1).Return(&order, nil)

	pendingOrder, err := svc.svc.AmendOrder(context.Background(), &arg)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
}

func testAmendOrderInvalidPayload(t *testing.T) {
	arg := amend
	arg.Size = 0
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	pendingOrder, err := svc.svc.AmendOrder(context.Background(), &arg)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
}

func testAmendOrderTimeSvcErr(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(-2 * time.Hour)
	expErr := errors.New("time service error")
	arg := amend
	arg.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	order := proto.Order{
		Id:          arg.OrderID,
		MarketID:    "market",
		PartyID:     arg.PartyID,
		Status:      proto.Order_Active,
		TimeInForce: proto.Order_GTT,
	}
	svc.orderStore.EXPECT().GetByPartyAndId(gomock.Any(), arg.PartyID, arg.OrderID).Times(1).Return(&order, nil)
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, expErr)

	pendingOrder, err := svc.svc.AmendOrder(context.Background(), &arg)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
}

func (m amendMatcher) String() string {
	return fmt.Sprintf("%#v", m.e)
}

func (m amendMatcher) Matches(x interface{}) bool {
	var v proto.OrderAmendment
	switch val := x.(type) {
	case *proto.OrderAmendment:
		v = *val
	case proto.OrderAmendment:
		v = val
	default:
		return false
	}
	return (m.e.OrderID == v.OrderID && m.e.PartyID == v.PartyID && m.e.Price == v.Price && m.e.Size == v.Size &&
		m.e.ExpiresAt == v.ExpiresAt)
}
