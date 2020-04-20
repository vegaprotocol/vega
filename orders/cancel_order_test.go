package orders_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

var (
	cancel = proto.OrderCancellation{
		OrderID:  "order_id",
		MarketID: "market",
		PartyID:  "party",
	}
)

type cancelMatcher struct {
	e proto.OrderCancellation
}

func TestCancelOrder(t *testing.T) {
	// PETETODO repalce with new versions for validation checks
	t.Run("Cancel order - success", testCancelOrderSuccess)
	t.Run("Cancel order - order not in storage", testCancelOrderNotFound)
	t.Run("Cancel order - already cancelled", testCancelOrderDuplicate)
	t.Run("Cancel order - order filled", testCancelOrderFilled)
	t.Run("Cancel order - party mismatch", testCancelOrderPartyMismatch)
}

func testCancelOrderSuccess(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := cancel
	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

func testCancelOrderNotFound(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := cancel

	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

/*
 * If we try to prepare a cancel for an order that is already cancelled, the prepare statement
 * will succeed as it does not have access to the order book.
 */
func testCancelOrderDuplicate(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := cancel
	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

/*
 * If we try to prepare a cancel for an order that is already filled, the prepare statement
 * will succeed as it does not have access to the order book.
 */
func testCancelOrderFilled(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := cancel
	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

/*
 * If we try to prepare a cancel for an order with an incorrect partyID, the prepare statement
 * will succeed as it does not have access to the order book to validate it.
 */
func testCancelOrderPartyMismatch(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := cancel
	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

func (m cancelMatcher) String() string {
	return fmt.Sprintf("%#v", m.e)
}

func (m cancelMatcher) Matches(x interface{}) bool {
	var v proto.Order
	switch val := x.(type) {
	case *proto.Order:
		v = *val
	case proto.Order:
		v = val
	default:
		return false
	}
	return (m.e.OrderID == v.Id && m.e.MarketID == v.MarketID && m.e.PartyID == v.PartyID)
}
