package commands

import (
	"errors"

	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

func CheckOrderSubmission(cmd *commandspb.OrderSubmission) error {
	return checkOrderSubmission(cmd).ErrorOrNil()
}

func checkOrderSubmission(cmd *commandspb.OrderSubmission) Errors {
	errs := NewErrors()

	if len(cmd.MarketId) == 0 {
		errs.AddForProperty("order_submission.market_id", ErrIsRequired)
	}

	if cmd.Side == types.Side_SIDE_UNSPECIFIED {
		errs.AddForProperty("order_submission.side", ErrIsRequired)
	}
	if _, ok := types.Side_name[int32(cmd.Side)]; !ok {
		errs.AddForProperty("order_submission.side", ErrIsNotValid)
	}

	if cmd.Type == types.Order_TYPE_UNSPECIFIED {
		errs.AddForProperty("order_submission.type", ErrIsRequired)
	}
	if _, ok := types.Order_Type_name[int32(cmd.Type)]; !ok {
		errs.AddForProperty("order_submission.type", ErrIsNotValid)
	}
	if cmd.Type == types.Order_TYPE_NETWORK {
		errs.AddForProperty("order_submission.type", ErrIsUnauthorised)
	}

	if cmd.TimeInForce == types.Order_TIME_IN_FORCE_UNSPECIFIED {
		errs.AddForProperty("order_submission.time_in_force", ErrIsRequired)
	}
	if _, ok := types.Order_Type_name[int32(cmd.TimeInForce)]; !ok {
		errs.AddForProperty("order_submission.time_in_force", ErrIsNotValid)
	}

	if cmd.Size <= 0 {
		errs.AddForProperty("order_submission.size", ErrMustBePositive)
	}

	if cmd.TimeInForce == types.Order_TIME_IN_FORCE_GTT {
		if cmd.ExpiresAt <= 0 {
			errs.AddForProperty("order_submission.expires_at", ErrMustBePositive)
		}
	} else {
		if cmd.ExpiresAt != 0 {
			errs.AddForProperty("order_submission.expires_at",
				errors.New("is only available when the time in force is of type GTT"),
			)
		}
	}

	if cmd.PeggedOrder != nil {
		if cmd.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
			errs.AddForProperty("order_submission.pegged_order.reference", ErrIsRequired)
		}
		if _, ok := types.PeggedReference_name[int32(cmd.PeggedOrder.Reference)]; !ok {
			errs.AddForProperty("order_submission.pegged_order.reference", ErrIsNotValid)
		}

		if cmd.Type != types.Order_TYPE_LIMIT {
			errs.AddForProperty("order_submission.type",
				errors.New("is expected to be an order of type LIMIT when the order is pegged"),
			)
		}

		if cmd.TimeInForce != types.Order_TIME_IN_FORCE_GTT &&
			cmd.TimeInForce != types.Order_TIME_IN_FORCE_GTC {
			errs.AddForProperty("order_submission.time_in_force",
				errors.New("is expected to have a time in force of type GTT or GTC when the order is pegged"),
			)
		}

		if cmd.Side == types.Side_SIDE_BUY {
			switch cmd.PeggedOrder.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				errs.AddForProperty("order_submission.pegged_order.reference",
					errors.New("cannot have a reference of type BEST_ASK when on BUY side"),
				)
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				if cmd.PeggedOrder.Offset > 0 {
					errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBeNegativeOrZero)
				}
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if cmd.PeggedOrder.Offset >= 0 {
					errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBeNegative)
				}
			}
		} else {
			switch cmd.PeggedOrder.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				errs.AddForProperty("order_submission.pegged_order.reference",
					errors.New("cannot have a reference of type BEST_BID when on SELL side"),
				)
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				if cmd.PeggedOrder.Offset < 0 {
					errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBePositiveOrZero)
				}
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if cmd.PeggedOrder.Offset <= 0 {
					errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBePositive)
				}
			}
		}
	} else {
		if cmd.Type == types.Order_TYPE_MARKET {
			if cmd.Price != 0 {
				errs.AddForProperty("order_submission.price",
					errors.New("is unavailable when the order is of type MARKET"),
				)
			}
			if cmd.TimeInForce != types.Order_TIME_IN_FORCE_FOK &&
				cmd.TimeInForce != types.Order_TIME_IN_FORCE_IOC {
				errs.AddForProperty("order_submission.time_in_force",
					errors.New("is expected to be of type FOK or IOC when order is of type MARKET"),
				)
			}
		} else if cmd.Type == types.Order_TYPE_LIMIT {
			if cmd.Price == 0 {
				errs.AddForProperty("order_submission.price",
					errors.New("is required when the order is of type LIMIT"),
				)
			}
		}
	}

	return errs
}
