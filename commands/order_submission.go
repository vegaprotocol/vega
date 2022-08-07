package commands

import (
	"errors"
	"math/big"

	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckOrderSubmission(cmd *commandspb.OrderSubmission) error {
	return checkOrderSubmission(cmd).ErrorOrNil()
}

func checkOrderSubmission(cmd *commandspb.OrderSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("order_submission", ErrIsRequired)
	}

	if len(cmd.MarketId) == 0 {
		errs.AddForProperty("order_submission.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.MarketId) {
		errs.AddForProperty("order_submission.market_id", ErrShouldBeAValidVegaID)
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
	if _, ok := types.Order_TimeInForce_name[int32(cmd.TimeInForce)]; !ok {
		errs.AddForProperty("order_submission.time_in_force", ErrIsNotValid)
	}

	if cmd.Size <= 0 {
		errs.AddForProperty("order_submission.size", ErrMustBePositive)
	}

	if cmd.TimeInForce == types.Order_TIME_IN_FORCE_GTT {
		if cmd.ExpiresAt <= 0 {
			errs.AddForProperty("order_submission.expires_at", ErrMustBePositive)
		}
	} else if cmd.ExpiresAt != 0 {
		errs.AddForProperty("order_submission.expires_at",
			errors.New("is only available when the time in force is of type GTT"),
		)
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
				if offset, ok := big.NewInt(0).SetString(cmd.PeggedOrder.Offset, 10); !ok {
					errs.AddForProperty(
						"order_submission.pegged_order.offset",
						ErrNotAValidInteger,
					)
				} else if offset.Cmp(big.NewInt(0)) == -1 {
					errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBePositiveOrZero)
				}
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if offset, ok := big.NewInt(0).SetString(cmd.PeggedOrder.Offset, 10); !ok {
					errs.AddForProperty(
						"order_submission.pegged_order.offset",
						ErrNotAValidInteger,
					)
				} else if offset.Cmp(big.NewInt(0)) == -1 || offset.Cmp(big.NewInt(0)) == 0 {
					errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBePositive)
				}
			}
			return errs
		}

		switch cmd.PeggedOrder.Reference {
		case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
			errs.AddForProperty("order_submission.pegged_order.reference",
				errors.New("cannot have a reference of type BEST_BID when on SELL side"),
			)
		case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
			if offset, ok := big.NewInt(0).SetString(cmd.PeggedOrder.Offset, 10); !ok {
				errs.AddForProperty(
					"order_submission.pegged_order.offset",
					ErrNotAValidInteger,
				)
			} else if offset.Cmp(big.NewInt(0)) == -1 {
				errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBePositiveOrZero)
			}
		case types.PeggedReference_PEGGED_REFERENCE_MID:
			if offset, ok := big.NewInt(0).SetString(cmd.PeggedOrder.Offset, 10); !ok {
				errs.AddForProperty(
					"order_submission.pegged_order.offset",
					ErrNotAValidInteger,
				)
			} else if offset.Cmp(big.NewInt(0)) == -1 || offset.Cmp(big.NewInt(0)) == 0 {
				errs.AddForProperty("order_submission.pegged_order.offset", ErrMustBePositive)
			}
		}

		return errs
	}

	switch cmd.Type {
	case types.Order_TYPE_MARKET:
		if len(cmd.Price) > 0 {
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
	case types.Order_TYPE_LIMIT:
		if len(cmd.Price) <= 0 {
			errs.AddForProperty("order_submission.price",
				errors.New("is required when the order is of type LIMIT"),
			)
		} else {
			if price, ok := big.NewInt(0).SetString(cmd.Price, 10); !ok {
				errs.AddForProperty("order_submission.price", ErrNotAValidInteger)
			} else if price.Cmp(big.NewInt(0)) != 1 {
				errs.AddForProperty("order_submission.price",
					errors.New("must be positive when the order is of type LIMIT"),
				)
			}
		}
	}

	return errs
}
