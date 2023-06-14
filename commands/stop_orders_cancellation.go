package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckStopOrdersCancellation(cmd *commandspb.StopOrdersCancellation) error {
	return checkStopOrdersCancellation(cmd).ErrorOrNil()
}

func checkStopOrdersCancellation(cmd *commandspb.StopOrdersCancellation) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("stop_orders_cancellation", ErrIsRequired)
	}

	if cmd.MarketId != nil && len(*cmd.MarketId) > 0 && !IsVegaPubkey(*cmd.MarketId) {
		errs.AddForProperty("stop_orders_cancellation.market_id", ErrShouldBeAValidVegaID)
	}

	if cmd.StopOrderId != nil && len(*cmd.StopOrderId) > 0 && !IsVegaPubkey(*cmd.StopOrderId) {
		errs.AddForProperty("stop_orders_cancellation.stop_order_id", ErrShouldBeAValidVegaID)
	}

	return errs
}
