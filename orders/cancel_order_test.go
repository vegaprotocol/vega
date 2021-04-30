package orders_test

import (
	"context"
	"testing"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/stretchr/testify/assert"
)

var (
	cancel = commandspb.OrderCancellation{
		OrderId:  "order_id",
		MarketId: "market",
	}
)

func TestCancelOrder(t *testing.T) {
	t.Run("Cancel order - success", testCancelOrderSuccess)
	t.Run("Cancel order - missing orderID", testCancelOrderNoOrderID)
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
	arg := commandspb.OrderCancellation{
		MarketId: "marketid",
	}
	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.NoError(t, err)
}

func testCancelOrderNoMarketID(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	ctx := context.Background()
	arg := commandspb.OrderCancellation{
		OrderId: "orderid",
	}

	err := svc.svc.PrepareCancelOrder(ctx, &arg)
	assert.Error(t, err)
}
