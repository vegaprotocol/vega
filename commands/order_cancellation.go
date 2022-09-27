package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckOrderCancellation(cmd *commandspb.OrderCancellation) error {
	return checkOrderCancellation(cmd).ErrorOrNil()
}

func checkOrderCancellation(cmd *commandspb.OrderCancellation) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("order_cancellation", ErrIsRequired)
	}

	if cmd.MarketId != nil && len(*cmd.MarketId) > 0 && !IsVegaPubkey(*cmd.MarketId) {
		errs.AddForProperty("order_cancellation.market_id", ErrShouldBeAValidVegaID)
	}

	if cmd.OrderId != nil && len(*cmd.OrderId) > 0 && !IsVegaPubkey(*cmd.OrderId) {
		errs.AddForProperty("order_cancellation.order_id", ErrShouldBeAValidVegaID)
	}

	return errs
}
