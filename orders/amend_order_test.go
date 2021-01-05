package orders_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/vegatime"
	"github.com/stretchr/testify/assert"
)

var (
	amend = proto.OrderAmendment{
		OrderID:   "order_id",
		PartyID:   "party",
		Price:     &proto.Price{Value: 10000},
		SizeDelta: 1,
		MarketID:  "market",
	}
)

func TestPrepareAmendOrder(t *testing.T) {
	t.Run("Prepare amend order price - success", testPrepareAmendOrderJustPriceSuccess)
	t.Run("Prepare amend order reduce - success", testPrepareAmendOrderJustReduceSuccess)
	t.Run("Prepare amend order increase - success", testPrepareAmendOrderJustIncreaseSuccess)
	t.Run("Prepare amend order expiry - success", testPrepareAmendOrderJustExpirySuccess)
	t.Run("Prepare amend order tif - success", testPrepareAmendOrderJustTIFSuccess)
	t.Run("Prepare amend order expiry before creation time - success", testPrepareAmendOrderPastExpiry)
	t.Run("Prepare amend order empty - fail", testPrepareAmendOrderEmptyFail)
	t.Run("Prepare amend order nil - fail", testPrepareAmendOrderNilFail)
	t.Run("Prepare amend order invalid expiry type - fail", testPrepareAmendOrderInvalidExpiryFail)
	t.Run("Prepare amend order tif to GFA - fail", testPrepareAmendOrderToGFA)
	t.Run("Prepare amend order tif to GFN - fail", testPrepareAmendOrderToGFN)
}

func testPrepareAmendOrderJustPriceSuccess(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID: "orderid",
		PartyID: "partyid",
		Price:   &proto.Price{Value: 1000},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.NoError(t, err)
}

func testPrepareAmendOrderJustReduceSuccess(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID:   "orderid",
		PartyID:   "partyid",
		SizeDelta: -10,
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.NoError(t, err)
}

func testPrepareAmendOrderJustIncreaseSuccess(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID:   "orderid",
		PartyID:   "partyid",
		SizeDelta: 10,
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.NoError(t, err)
}

func testPrepareAmendOrderJustExpirySuccess(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(-2 * time.Hour)
	arg := proto.OrderAmendment{
		OrderID:   "orderid",
		PartyID:   "partyid",
		ExpiresAt: &proto.Timestamp{Value: expires.UnixNano()},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.NoError(t, err)
}

func testPrepareAmendOrderJustTIFSuccess(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID:     "orderid",
		PartyID:     "partyid",
		TimeInForce: proto.Order_TIF_GTC,
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.NoError(t, err)
}

func testPrepareAmendOrderEmptyFail(t *testing.T) {
	arg := proto.OrderAmendment{}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)

	arg2 := proto.OrderAmendment{
		OrderID: "orderid",
		PartyID: "partyid",
	}
	err = svc.svc.PrepareAmendOrder(context.Background(), &arg2)
	assert.Error(t, err)
}

func testPrepareAmendOrderNilFail(t *testing.T) {
	var arg proto.OrderAmendment
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)
}

func testPrepareAmendOrderInvalidExpiryFail(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID:     "orderid",
		PartyID:     "partyid",
		TimeInForce: proto.Order_TIF_GTC,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)

	arg.TimeInForce = proto.Order_TIF_FOK
	err = svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)

	arg.TimeInForce = proto.Order_TIF_IOC
	err = svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)
}

/*
 * Sending an old expiry date is OK and should not be rejected here.
 * The validation should take place inside the core
 */
func testPrepareAmendOrderPastExpiry(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID:     "orderid",
		PartyID:     "partyid",
		TimeInForce: proto.Order_TIF_GTT,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.NoError(t, err)
}

func testPrepareAmendOrderToGFN(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID:     "orderid",
		PartyID:     "partyid",
		TimeInForce: proto.Order_TIF_GFN,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)
}

func testPrepareAmendOrderToGFA(t *testing.T) {
	arg := proto.OrderAmendment{
		OrderID:     "orderid",
		PartyID:     "partyid",
		TimeInForce: proto.Order_TIF_GFA,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), &arg)
	assert.Error(t, err)
}
