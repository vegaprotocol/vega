package commands_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/commands"
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckOrderAmendment(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilOrderAmendmentFails)
	t.Run("amend order price - success", testAmendOrderJustPriceSuccess)
	t.Run("amend order reduce - success", testAmendOrderJustReduceSuccess)
	t.Run("amend order increase - success", testAmendOrderJustIncreaseSuccess)
	t.Run("amend order expiry - success", testAmendOrderJustExpirySuccess)
	t.Run("amend order tif - success", testAmendOrderJustTIFSuccess)
	t.Run("amend order expiry before creation time - success", testAmendOrderPastExpiry)
	t.Run("amend order empty - fail", testAmendOrderEmptyFail)
	t.Run("amend order empty - fail", testAmendEmptyFail)
	t.Run("amend order invalid expiry type - fail", testAmendOrderInvalidExpiryFail)
	t.Run("amend order tif to GFA - fail", testAmendOrderToGFA)
	t.Run("amend order tif to GFN - fail", testAmendOrderToGFN)
	t.Run("amend order pegged_offset", testAmendOrderPeggedOffset)
}

func testNilOrderAmendmentFails(t *testing.T) {
	err := checkOrderAmendment(nil)

	assert.Contains(t, err.Get("order_amendment"), commands.ErrIsRequired)
}

func testAmendOrderJustPriceSuccess(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		Price:    StringPtr("1000"),
	}
	err := checkOrderAmendment(arg)

	assert.NoError(t, err.ErrorOrNil())
}

func testAmendOrderJustReduceSuccess(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:   "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		SizeDelta: -10,
	}
	err := checkOrderAmendment(arg)
	assert.NoError(t, err.ErrorOrNil())
}

func testAmendOrderJustIncreaseSuccess(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:   "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		SizeDelta: 10,
	}
	err := checkOrderAmendment(arg)
	assert.NoError(t, err.ErrorOrNil())
}

func testAmendOrderJustExpirySuccess(t *testing.T) {
	now := time.Now()
	expires := now.Add(-2 * time.Hour)
	arg := &commandspb.OrderAmendment{
		OrderId:   "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		ExpiresAt: Int64Ptr(expires.UnixNano()),
	}
	err := checkOrderAmendment(arg)
	assert.NoError(t, err.ErrorOrNil())
}

func testAmendOrderJustTIFSuccess(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:     "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId:    "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
	}
	err := checkOrderAmendment(arg)
	assert.NoError(t, err.ErrorOrNil())
}

func testAmendOrderEmptyFail(t *testing.T) {
	arg := &commandspb.OrderAmendment{}
	err := checkOrderAmendment(arg)
	assert.Error(t, err)

	arg2 := &commandspb.OrderAmendment{
		OrderId:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
	}
	err = checkOrderAmendment(arg2)
	assert.Error(t, err)
}

func testAmendEmptyFail(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
	}
	err := checkOrderAmendment(arg)
	assert.Error(t, err)
}

func testAmendOrderInvalidExpiryFail(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:     "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
		ExpiresAt:   Int64Ptr(10),
	}
	err := checkOrderAmendment(arg)
	assert.Error(t, err)

	arg.TimeInForce = proto.Order_TIME_IN_FORCE_FOK
	err = checkOrderAmendment(arg)
	assert.Error(t, err)

	arg.TimeInForce = proto.Order_TIME_IN_FORCE_IOC
	err = checkOrderAmendment(arg)
	assert.Error(t, err)
}

func testAmendOrderPeggedOffset(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:      "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId:     "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		PeggedOffset: "-1789",
		TimeInForce:  proto.Order_TIME_IN_FORCE_GTC, // just here to test the case with empty pegged offset
	}

	err := checkOrderAmendment(arg)
	assert.Error(t, err.ErrorOrNil())

	arg.PeggedOffset = ""
	err = checkOrderAmendment(arg)
	assert.NoError(t, err.ErrorOrNil())

	arg.PeggedOffset = "1000"
	err = checkOrderAmendment(arg)
	assert.NoError(t, err.ErrorOrNil())
}

/*
 * Sending an old expiry date is OK and should not be rejected here.
 * The validation should take place inside the core.
 */
func testAmendOrderPastExpiry(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:     "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		MarketId:    "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		TimeInForce: proto.Order_TIME_IN_FORCE_GTT,
		ExpiresAt:   Int64Ptr(10),
	}
	err := checkOrderAmendment(arg)
	assert.NoError(t, err.ErrorOrNil())
}

func testAmendOrderToGFN(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:     "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		TimeInForce: proto.Order_TIME_IN_FORCE_GFN,
		ExpiresAt:   Int64Ptr(10),
	}
	err := checkOrderAmendment(arg)
	assert.Error(t, err)
}

func testAmendOrderToGFA(t *testing.T) {
	arg := &commandspb.OrderAmendment{
		OrderId:     "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		TimeInForce: proto.Order_TIME_IN_FORCE_GFA,
		ExpiresAt:   Int64Ptr(10),
	}
	err := checkOrderAmendment(arg)
	assert.Error(t, err)
}

func checkOrderAmendment(cmd *commandspb.OrderAmendment) commands.Errors {
	err := commands.CheckOrderAmendment(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
