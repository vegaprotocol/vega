package orders_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	amend = proto.OrderAmendment{
		OrderID:   "order_id",
		PartyID:   "party",
		Price:     10000,
		SizeDelta: 1,
		MarketID:  "market",
	}
)

type amendMatcher struct {
	e proto.OrderAmendment
}

func TestPrepareAmendOrder(t *testing.T) {
	// PETETODO remove these and replace with new checks
	t.Run("Prepare amend order - success", testPrepareAmendOrderSuccess)
	t.Run("Prepare amend order - expired", testPrepareAmendOrderExpired)
	t.Run("Prepare amend order - not active", testPrepareAmendOrderNotActive)
	t.Run("Prepare amend order - invalid payload", testPrepareAmendOrderInvalidPayload)
	t.Run("Prepare amend order - time service error", testPrepareAmendOrderTimeSvcErr)
}

func testPrepareAmendOrderSuccess(t *testing.T) {
	// now := vegatime.Now()
	arg := amend
	arg.TimeInForce = proto.Order_GTC
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.NoError(t, err)
}

func testPrepareAmendOrderExpired(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(-2 * time.Hour)
	arg := amend
	arg.ExpiresAt = expires.UnixNano()
	arg.TimeInForce = types.Order_GTT
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)
}

func testPrepareAmendOrderNotActive(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(2 * time.Hour)
	arg := amend
	arg.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)
}

func testPrepareAmendOrderInvalidPayload(t *testing.T) {
	arg := amend
	arg.OrderID = ""
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)
}

func testPrepareAmendOrderTimeSvcErr(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(-2 * time.Hour)
	expErr := errors.New("time service error")
	arg := amend
	arg.SizeDelta = 0
	arg.ExpiresAt = expires.UnixNano()
	arg.TimeInForce = types.Order_GTT
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, expErr)
	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.EqualError(t, err, expErr.Error())
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
	return (m.e.OrderID == v.OrderID && m.e.PartyID == v.PartyID && m.e.Price == v.Price && m.e.SizeDelta == v.SizeDelta &&
		m.e.ExpiresAt == v.ExpiresAt)
}
