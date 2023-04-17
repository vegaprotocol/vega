package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckOrderSubmission(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilOrderSubmissionFails)
	t.Run("Submitting an empty order fails", testEmptyOrderSubmissionFails)
	t.Run("Submitting an order without market ID fails", testOrderSubmissionWithoutMarketIDFails)
	t.Run("Submitting an order with unspecified side fails", testOrderSubmissionWithUnspecifiedSideFails)
	t.Run("Submitting an order with undefined side fails", testOrderSubmissionWithUndefinedSideFails)
	t.Run("Submitting an order with unspecified type fails", testOrderSubmissionWithUnspecifiedTypeFails)
	t.Run("Submitting an order with undefined type fails", testOrderSubmissionWithUndefinedTypeFails)
	t.Run("Submitting an order with NETWORK type fails", testOrderSubmissionWithNetworkTypeFails)
	t.Run("Submitting an order with undefined time in force fails", testOrderSubmissionWithUndefinedTimeInForceFails)
	t.Run("Submitting an order with unspecified time in force fails", testOrderSubmissionWithUnspecifiedTimeInForceFails)
	t.Run("Submitting an order with non-positive size fails", testOrderSubmissionWithNonPositiveSizeFails)
	t.Run("Submitting an order with GTT and non-positive expiration date fails", testOrderSubmissionWithGTTAndNonPositiveExpirationDateFails)
	t.Run("Submitting an order without GTT and expiration date fails", testOrderSubmissionWithoutGTTAndExpirationDateFails)
	t.Run("Submitting an order with MARKET type and price fails", testOrderSubmissionWithMarketTypeAndPriceFails)
	t.Run("Submitting an order with MARKET type and wrong time in force fails", testOrderSubmissionWithMarketTypeAndWrongTimeInForceFails)
	t.Run("Submitting an order with LIMIT type and no price fails", testOrderSubmissionWithLimitTypeAndNoPriceFails)
	t.Run("Submitting an order with LIMIT type and negative price fails", testOrderSubmissionWithLimitTypeAndNegativePriceFails)
	t.Run("Submitting a pegged order with LIMIT type and no price succeeds", testPeggedOrderSubmissionWithLimitTypeAndNoPriceSucceeds)
	t.Run("Submitting a pegged order with undefined time in force fails", testPeggedOrderSubmissionWithUndefinedReferenceFails)
	t.Run("Submitting a pegged order with unspecified time in force fails", testPeggedOrderSubmissionWithUnspecifiedReferenceFails)
	t.Run("Submitting a pegged order without LIMIT type fails", testPeggedOrderSubmissionWithoutLimitTypeFails)
	t.Run("Submitting a pegged order with LIMIT type succeeds", testPeggedOrderSubmissionWithLimitTypeSucceeds)
	t.Run("Submitting a pegged order with wrong time in force fails", testPeggedOrderSubmissionWithWrongTimeInForceFails)
	t.Run("Submitting a pegged order with right time in force succeeds", testPeggedOrderSubmissionWithRightTimeInForceSucceeds)
	t.Run("Submitting a pegged order with side buy and best ask reference fails", testPeggedOrderSubmissionWithSideBuyAndBestAskReferenceFails)
	t.Run("Submitting a pegged order with side buy and best bid reference succeeds", testPeggedOrderSubmissionWithSideBuyAndBestBidReferenceSucceeds)
	t.Run("Submitting a pegged order with side buy and best bid reference and negative offset fails", testPeggedOrderSubmissionWithSideBuyAndBestBidReferenceAndNegativeOffsetFails)
	t.Run("Submitting a pegged order with side buy and best bid reference and non-negative offset succeeds", testPeggedOrderSubmissionWithSideBuyAndBestBidReferenceAndNonNegativeOffsetSucceeds)
	t.Run("Submitting a pegged order with side buy and mid reference and non-negative offset fails", testPeggedOrderSubmissionWithSideBuyAndMidReferenceAndNonPositiveOffsetFails)
	t.Run("Submitting a pegged order with side buy and mid reference and negative offset succeeds", testPeggedOrderSubmissionWithSideBuyAndMidReferenceAndNegativeOffsetSucceeds)
	t.Run("Submitting a pegged order with side sell and best bid reference fails", testPeggedOrderSubmissionWithSideSellAndBestBidReferenceFails)
	t.Run("Submitting a pegged order with side sell and best ask reference succeeds", testPeggedOrderSubmissionWithSideSellAndBestAskReferenceSucceeds)
	t.Run("Submitting a pegged order with side sell and best ask reference and negative offset fails", testPeggedOrderSubmissionWithSideSellAndBestAskReferenceAndNegativeOffsetFails)
	t.Run("Submitting a pegged order with side sell and best ask reference and non negative offset succeeds", testPeggedOrderSubmissionWithSideSellAndBestAskReferenceAndNonNegativeOffsetSucceeds)
	t.Run("Submitting a pegged order with side sell and mid reference and non-positive offset fails", testPeggedOrderSubmissionWithSideSellAndMidReferenceAndNonPositiveOffsetFails)
	t.Run("Submitting a pegged order with side sell and mid reference and positive offset succeeds", testPeggedOrderSubmissionWithSideSellAndMidReferenceAndPositiveOffsetSucceeds)
	t.Run("Submitting Post or Reduce only orders", testSubmittingPostOrReduceOnlyOrders)
}

func testSubmittingPostOrReduceOnlyOrders(t *testing.T) {
	testCases := []struct {
		submission commandspb.OrderSubmission
		errString  string
		field      string
	}{
		{
			submission: commandspb.OrderSubmission{
				PostOnly:   true,
				ReduceOnly: true,
			},
			errString: "cannot be true at the same time as order_submission.reduce_only",
			field:     "order_submission.post_only",
		},
		{
			submission: commandspb.OrderSubmission{
				Type:     types.Order_TYPE_MARKET,
				PostOnly: true,
			},
			errString: "only valid for limit orders",
			field:     "order_submission.post_only",
		},
		{
			submission: commandspb.OrderSubmission{
				Type:        types.Order_TYPE_MARKET,
				TimeInForce: types.Order_TIME_IN_FORCE_FOK,
				PostOnly:    true,
			},
			errString: "only valid for persistent orders",
			field:     "order_submission.post_only",
		},
		{
			submission: commandspb.OrderSubmission{
				Type:        types.Order_TYPE_MARKET,
				TimeInForce: types.Order_TIME_IN_FORCE_FOK,
				PostOnly:    true,
			},
			errString: "only valid for persistent orders",
			field:     "order_submission.post_only",
		},
		{
			submission: commandspb.OrderSubmission{
				Type:        types.Order_TYPE_LIMIT,
				TimeInForce: types.Order_TIME_IN_FORCE_GTC,
				ReduceOnly:  true,
			},
			errString: "only valid for non-persistent orders",
			field:     "order_submission.reduce_only",
		},
		{
			submission: commandspb.OrderSubmission{
				Type:        types.Order_TYPE_LIMIT,
				TimeInForce: types.Order_TIME_IN_FORCE_IOC,
				ReduceOnly:  true,
				PeggedOrder: &types.PeggedOrder{
					Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
				},
			},
			errString: "cannot be pegged",
			field:     "order_submission.reduce_only",
		},
		// valid cases
		{
			submission: commandspb.OrderSubmission{
				Type:        types.Order_TYPE_LIMIT,
				TimeInForce: types.Order_TIME_IN_FORCE_IOC,
				ReduceOnly:  true,
			},
			errString: "",
			field:     "order_submission.reduce_only",
		},
		{
			submission: commandspb.OrderSubmission{
				Type:        types.Order_TYPE_LIMIT,
				TimeInForce: types.Order_TIME_IN_FORCE_GTC,
				PostOnly:    true,
			},
			errString: "",
			field:     "order_submission.post_only",
		},
	}

	for _, tc := range testCases {
		errs := checkOrderSubmission(&tc.submission).Get(tc.field)
		if len(tc.errString) == 0 {
			assert.Len(t, errs, 0)
			continue
		}
		assert.Contains(t, errs, errors.New(tc.errString))
	}
}

func testEmptyOrderSubmissionFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{})

	assert.Error(t, err)
}

func testNilOrderSubmissionFails(t *testing.T) {
	err := checkOrderSubmission(nil)

	assert.Contains(t, err.Get("order_submission"), commands.ErrIsRequired)
}

func testOrderSubmissionWithoutMarketIDFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		MarketId: "",
	})

	assert.Contains(t, err.Get("order_submission.market_id"), commands.ErrIsRequired)
}

func testOrderSubmissionWithUnspecifiedSideFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_UNSPECIFIED,
	})

	assert.Contains(t, err.Get("order_submission.side"), commands.ErrIsRequired)
}

func testOrderSubmissionWithUndefinedSideFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side(-42),
	})

	assert.Contains(t, err.Get("order_submission.side"), commands.ErrIsNotValid)
}

func testOrderSubmissionWithUnspecifiedTypeFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type: types.Order_TYPE_UNSPECIFIED,
	})

	assert.Contains(t, err.Get("order_submission.type"), commands.ErrIsRequired)
}

func testOrderSubmissionWithUndefinedTypeFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type: types.Order_Type(-42),
	})

	assert.Contains(t, err.Get("order_submission.type"), commands.ErrIsNotValid)
}

func testOrderSubmissionWithNetworkTypeFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type: types.Order_TYPE_NETWORK,
	})

	assert.Contains(t, err.Get("order_submission.type"), commands.ErrIsUnauthorised)
}

func testOrderSubmissionWithUnspecifiedTimeInForceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		TimeInForce: types.Order_TIME_IN_FORCE_UNSPECIFIED,
	})

	assert.Contains(t, err.Get("order_submission.time_in_force"), commands.ErrIsRequired)
}

func testOrderSubmissionWithUndefinedTimeInForceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		TimeInForce: types.Order_TimeInForce(-42),
	})

	assert.Contains(t, err.Get("order_submission.time_in_force"), commands.ErrIsNotValid)
}

func testOrderSubmissionWithNonPositiveSizeFails(t *testing.T) {
	// FIXME(big int) doesn't test negative numbers since it's an unsigned int
	// 	but that will definitely be needed when moving to big int.
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Size: 0,
	})

	assert.Contains(t, err.Get("order_submission.size"), commands.ErrMustBePositive)
}

func testOrderSubmissionWithGTTAndNonPositiveExpirationDateFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value int64
	}{
		{
			msg:   "with 0 as expiration date",
			value: 0,
		}, {
			msg:   "with negative expiration date",
			value: RandomNegativeI64(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				TimeInForce: types.Order_TIME_IN_FORCE_GTT,
				ExpiresAt:   tc.value,
			})

			assert.Contains(t, err.Get("order_submission.expires_at"), commands.ErrMustBePositive)
		})
	}
}

func testOrderSubmissionWithoutGTTAndExpirationDateFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value types.Order_TimeInForce
	}{
		{
			msg:   "with GTC",
			value: types.Order_TIME_IN_FORCE_GTC,
		}, {
			msg:   "with IOC",
			value: types.Order_TIME_IN_FORCE_IOC,
		}, {
			msg:   "with FOK",
			value: types.Order_TIME_IN_FORCE_FOK,
		}, {
			msg:   "with GFA",
			value: types.Order_TIME_IN_FORCE_GFA,
		}, {
			msg:   "with GFN",
			value: types.Order_TIME_IN_FORCE_GFN,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				TimeInForce: tc.value,
				ExpiresAt:   RandomI64(),
			})

			assert.Contains(t, err.Get("order_submission.expires_at"), errors.New("is only available when the time in force is of type GTT"))
		})
	}
}

func testOrderSubmissionWithMarketTypeAndPriceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type:  types.Order_TYPE_MARKET,
		Price: RandomPositiveU64AsString(),
	})

	assert.Contains(t, err.Get("order_submission.price"), errors.New("is unavailable when the order is of type MARKET"))
}

func testOrderSubmissionWithMarketTypeAndWrongTimeInForceFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value types.Order_TimeInForce
	}{
		{
			msg:   "with GTC",
			value: types.Order_TIME_IN_FORCE_GTC,
		}, {
			msg:   "with GTT",
			value: types.Order_TIME_IN_FORCE_GTT,
		}, {
			msg:   "with GFA",
			value: types.Order_TIME_IN_FORCE_GFA,
		}, {
			msg:   "with GFN",
			value: types.Order_TIME_IN_FORCE_GFN,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				Type:        types.Order_TYPE_MARKET,
				TimeInForce: tc.value,
			})

			assert.Contains(t, err.Get("order_submission.time_in_force"), errors.New("is expected to be of type FOK or IOC when order is of type MARKET"))
		})
	}
}

func testOrderSubmissionWithLimitTypeAndNoPriceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type: types.Order_TYPE_LIMIT,
	})

	assert.Contains(t, err.Get("order_submission.price"), errors.New("is required when the order is of type LIMIT"))
}

func testOrderSubmissionWithLimitTypeAndNegativePriceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type:  types.Order_TYPE_LIMIT,
		Price: "-1000",
	})

	assert.Contains(t, err.Get("order_submission.price"), errors.New("must be positive when the order is of type LIMIT"))
}

func testPeggedOrderSubmissionWithLimitTypeAndNoPriceSucceeds(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type:        types.Order_TYPE_LIMIT,
		PeggedOrder: &types.PeggedOrder{},
	})

	assert.NotContains(t, err.Get("order_submission.price"), errors.New("is required when the order is of type LIMIT"))
}

func testPeggedOrderSubmissionWithUnspecifiedReferenceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED,
		},
	})

	assert.Contains(t, err.Get("order_submission.pegged_order.reference"), commands.ErrIsRequired)
}

func testPeggedOrderSubmissionWithUndefinedReferenceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference(-42),
		},
	})

	assert.Contains(t, err.Get("order_submission.pegged_order.reference"), commands.ErrIsNotValid)
}

func testPeggedOrderSubmissionWithoutLimitTypeFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value types.Order_Type
	}{
		{
			msg:   "with MARKET",
			value: types.Order_TYPE_MARKET,
		}, {
			msg:   "with NETWORK",
			value: types.Order_TYPE_NETWORK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				Type:        tc.value,
				PeggedOrder: &types.PeggedOrder{},
			})

			assert.Contains(t, err.Get("order_submission.type"), errors.New("is expected to be an order of type LIMIT when the order is pegged"))
		})
	}
}

func testPeggedOrderSubmissionWithLimitTypeSucceeds(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Type:        types.Order_TYPE_LIMIT,
		PeggedOrder: &types.PeggedOrder{},
	})

	assert.NotContains(t, err.Get("order_submission.type"), errors.New("is expected to be an order of type LIMIT when the order is pegged"))
}

func testPeggedOrderSubmissionWithWrongTimeInForceFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value types.Order_TimeInForce
	}{
		{
			msg:   "with IOC",
			value: types.Order_TIME_IN_FORCE_IOC,
		}, {
			msg:   "with FOK",
			value: types.Order_TIME_IN_FORCE_FOK,
		}, {
			msg:   "with GFA",
			value: types.Order_TIME_IN_FORCE_GFA,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				TimeInForce: tc.value,
				PeggedOrder: &types.PeggedOrder{},
			})

			assert.Contains(t, err.Get("order_submission.time_in_force"), errors.New("is expected to have a time in force of type GTT, GTC or GFN when the order is pegged"))
		})
	}
}

func testPeggedOrderSubmissionWithRightTimeInForceSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value types.Order_TimeInForce
	}{
		{
			msg:   "with GTC",
			value: types.Order_TIME_IN_FORCE_GTC,
		}, {
			msg:   "with GTT",
			value: types.Order_TIME_IN_FORCE_GTT,
		}, {
			msg:   "with GFN",
			value: types.Order_TIME_IN_FORCE_GFN,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				TimeInForce: tc.value,
				PeggedOrder: &types.PeggedOrder{},
			})

			assert.NotContains(t, err.Get("order_submission.time_in_force"), errors.New("is expected to have a time in force of type GTT, GTC or GFN when the order is pegged"))
		})
	}
}

func testPeggedOrderSubmissionWithSideBuyAndBestAskReferenceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_BUY,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
		},
	})

	assert.Contains(t, err.Get("order_submission.pegged_order.reference"), errors.New("cannot have a reference of type BEST_ASK when on BUY side"))
}

func testPeggedOrderSubmissionWithSideBuyAndBestBidReferenceSucceeds(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_BUY,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
		},
	})

	assert.NotContains(t, err.Get("order_submission.pegged_order.reference"), errors.New("cannot have a reference of type BEST_ASK when on BUY side"))
}

func testPeggedOrderSubmissionWithSideBuyAndBestBidReferenceAndNegativeOffsetFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_BUY,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
			Offset:    "-1",
		},
	})

	assert.Contains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be positive or zero"))
}

func testPeggedOrderSubmissionWithSideBuyAndBestBidReferenceAndNonNegativeOffsetSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value string
	}{
		{
			msg:   "with 0 offset",
			value: "0",
		}, {
			msg:   "with positive offset",
			value: RandomPositiveU64AsString(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				Side: types.Side_SIDE_BUY,
				PeggedOrder: &types.PeggedOrder{
					Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
					Offset:    tc.value,
				},
			})

			assert.NotContains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be positive or zero"))
		})
	}
}

func testPeggedOrderSubmissionWithSideBuyAndMidReferenceAndNonPositiveOffsetFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value string
	}{
		{
			msg:   "with 0 offset",
			value: "0",
		}, {
			msg:   "with negative offset",
			value: RandomNegativeI64AsString(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				Side: types.Side_SIDE_BUY,
				PeggedOrder: &types.PeggedOrder{
					Reference: types.PeggedReference_PEGGED_REFERENCE_MID,
					Offset:    tc.value,
				},
			})

			assert.Contains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be positive"))
		})
	}
}

func testPeggedOrderSubmissionWithSideBuyAndMidReferenceAndNegativeOffsetSucceeds(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_BUY,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_MID,
			Offset:    RandomPositiveU64AsString(),
		},
	})

	assert.NotContains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be negative"))
}

func testPeggedOrderSubmissionWithSideSellAndBestBidReferenceFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_SELL,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID,
		},
	})

	assert.Contains(t, err.Get("order_submission.pegged_order.reference"), errors.New("cannot have a reference of type BEST_BID when on SELL side"))
}

func testPeggedOrderSubmissionWithSideSellAndBestAskReferenceSucceeds(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_SELL,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
		},
	})

	assert.NotContains(t, err.Get("order_submission.pegged_order.reference"), errors.New("cannot have a reference of type BEST_BID when on SELL side"))
}

func testPeggedOrderSubmissionWithSideSellAndBestAskReferenceAndNegativeOffsetFails(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_SELL,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
			Offset:    RandomNegativeI64AsString(),
		},
	})

	assert.Contains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be positive or zero"))
}

func testPeggedOrderSubmissionWithSideSellAndBestAskReferenceAndNonNegativeOffsetSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value string
	}{
		{
			msg:   "with 0 offset",
			value: "0",
		}, {
			msg:   "with positive offset",
			value: RandomPositiveU64AsString(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				Side: types.Side_SIDE_SELL,
				PeggedOrder: &types.PeggedOrder{
					Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_ASK,
					Offset:    tc.value,
				},
			})

			assert.NotContains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be positive or zero"))
		})
	}
}

func testPeggedOrderSubmissionWithSideSellAndMidReferenceAndNonPositiveOffsetFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value string
	}{
		{
			msg:   "with 0 offset",
			value: "0",
		}, {
			msg:   "with negative offset",
			value: "-1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOrderSubmission(&commandspb.OrderSubmission{
				Side: types.Side_SIDE_SELL,
				PeggedOrder: &types.PeggedOrder{
					Reference: types.PeggedReference_PEGGED_REFERENCE_MID,
					Offset:    tc.value,
				},
			})

			assert.Contains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be positive"))
		})
	}
}

func testPeggedOrderSubmissionWithSideSellAndMidReferenceAndPositiveOffsetSucceeds(t *testing.T) {
	err := checkOrderSubmission(&commandspb.OrderSubmission{
		Side: types.Side_SIDE_SELL,
		PeggedOrder: &types.PeggedOrder{
			Reference: types.PeggedReference_PEGGED_REFERENCE_MID,
			Offset:    RandomPositiveU64AsString(),
		},
	})

	assert.NotContains(t, err.Get("order_submission.pegged_order.offset"), errors.New("must be positive"))
}

func checkOrderSubmission(cmd *commandspb.OrderSubmission) commands.Errors {
	err := commands.CheckOrderSubmission(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
