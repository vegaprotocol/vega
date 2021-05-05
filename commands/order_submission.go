package commands

import (
	"errors"

	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

func CheckOrderSubmission(cmd *commandspb.OrderSubmission) error {
	var errs []error
	if len(cmd.MarketId) <= 0 {
		errs = append(errs, errors.New("order_submission.market_id is required"))
	}
	if cmd.Side == types.Side_SIDE_UNSPECIFIED {
		errs = append(errs, errors.New("order_submission.side is required"))
	}
	if _, ok := types.Side_name[cmd.Side]; !ok {
		errs = append(errs, errors.New("order_submission.side is not valid"))
	}
	if cmd.Size <= 0 {
		errs = append(errs, errors.New("order_submission.size is required to be > 0"))
	}

	return Errors(errs).ErrorOrNil()
}
