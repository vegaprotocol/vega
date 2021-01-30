package orders_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

var (
	cancel = proto.OrderCancellation{
		OrderId:  "order_id",
		MarketId: "market",
		PartyId:  "party",
	}
)

func TestCancelOrder(t *testing.T) {
	t.Run("Cancel order - success", testCancelOrderSuccess)
	t.Run("Cancel order - missing orderID", testCancelOrderNoOrderID)
	t.Run("Cancel order - missing partyID", testCancelOrderNoPartyID)
	t.Run("Cancel order - missing marketID", testCancelOrderNoMarketID)
}

func testCancelOrderSuccess(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := cancel
	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

func testCancelOrderNoOrderID(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := proto.OrderCancellation{
		MarketId: "marketid",
		PartyId:  "partyid",
	}
	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

func testCancelOrderNoPartyID(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := proto.OrderCancellation{
		MarketId: "marketid",
		OrderId:  "partyid",
	}

	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.Error(t, err)
}

func testCancelOrderNoMarketID(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := proto.OrderCancellation{
		OrderId: "orderid",
		PartyId: "partyid",
	}

	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.Error(t, err)
}
