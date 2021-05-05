package commands_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/stretchr/testify/assert"
)

func TestCheckOrderAmendment(t *testing.T) {
	t.Run("amend order price - success", testAmendOrderJustPriceSuccess)
	t.Run("amend order reduce - success", testAmendOrderJustReduceSuccess)
	t.Run("amend order increase - success", testAmendOrderJustIncreaseSuccess)
	t.Run("amend order expiry - success", testAmendOrderJustExpirySuccess)
	t.Run("amend order tif - success", testAmendOrderJustTIFSuccess)
	t.Run("amend order expiry before creation time - success", testAmendOrderPastExpiry)
	t.Run("amend order empty - fail", testAmendOrderEmptyFail)
	t.Run("amend order empty - fail", testAmendOrderEmptyFail)
	t.Run("amend order invalid expiry type - fail", testAmendOrderInvalidExpiryFail)
	t.Run("amend order tif to GFA - fail", testAmendOrderToGFA)
	t.Run("amend order tif to GFN - fail", testAmendOrderToGFN)
}

func testAmendOrderJustPriceSuccess(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:  "orderid",
		MarketId: "marketid",
		Price:    &proto.Price{Value: 1000},
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.NoError(t, err)
}

func testAmendOrderJustReduceSuccess(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:   "orderid",
		MarketId:  "marketid",
		SizeDelta: -10,
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.NoError(t, err)
}

func testAmendOrderJustIncreaseSuccess(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:   "orderid",
		MarketId:  "marketid",
		SizeDelta: 10,
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.NoError(t, err)
}

func testAmendOrderJustExpirySuccess(t *testing.T) {
	now := vegatime.Now()
	expires := now.Add(-2 * time.Hour)
	arg := commandspb.OrderAmendment{
		OrderId:   "orderid",
		MarketId:  "marketid",
		ExpiresAt: &proto.Timestamp{Value: expires.UnixNano()},
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.NoError(t, err)
}

func testAmendOrderJustTIFSuccess(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:     "orderid",
		MarketId:    "marketid",
		TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.NoError(t, err)
}

func testAmendOrderEmptyFail(t *testing.T) {
	arg := commandspb.OrderAmendment{}
	err := commands.CheckOrderAmendment(&arg)
	assert.Error(t, err)

	arg2 := commandspb.OrderAmendment{
		OrderId:  "orderid",
		MarketId: "marketid",
	}
	err = commands.CheckOrderAmendment(&arg2)
	assert.Error(t, err)
}

func testAmendEmptyFail(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:  "orderid",
		MarketId: "marketid",
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.Error(t, err)
}

func testAmendOrderInvalidExpiryFail(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:     "orderid",
		PartyId:     "partyid",
		TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.Error(t, err)

	arg.TimeInForce = proto.Order_TIME_IN_FORCE_FOK
	err = commands.CheckOrderAmendment(&arg)
	assert.Error(t, err)

	arg.TimeInForce = proto.Order_TIME_IN_FORCE_IOC
	err = commands.CheckOrderAmendment(&arg)
	assert.Error(t, err)
}

/*
 * Sending an old expiry date is OK and should not be rejected here.
 * The validation should take place inside the core
 */
func testAmendOrderPastExpiry(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:     "orderid",
		MarketId:    "marketid",
		TimeInForce: proto.Order_TIME_IN_FORCE_GTT,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.NoError(t, err)
}

func testAmendOrderToGFN(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:     "orderid",
		PartyId:     "partyid",
		TimeInForce: proto.Order_TIME_IN_FORCE_GFN,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.Error(t, err)
}

func testAmendOrderToGFA(t *testing.T) {
	arg := commandspb.OrderAmendment{
		OrderId:     "orderid",
		PartyId:     "partyid",
		TimeInForce: proto.Order_TIME_IN_FORCE_GFA,
		ExpiresAt:   &proto.Timestamp{Value: 10},
	}
	err := commands.CheckOrderAmendment(&arg)
	assert.Error(t, err)
}
