package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckBatchMarketInstructions(cmd *commandspb.BatchMarketInstructions) error {
	return checkBatchMarketInstructions(cmd).ErrorOrNil()
}

func checkBatchMarketInstructions(cmd *commandspb.BatchMarketInstructions) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("batch_market_instructions", ErrIsRequired)
	}

	// there's very little to verify here, only if the batch is not empty
	// all transaction verification is done when processing then.
	if len(cmd.Cancellations)+len(cmd.Amendments)+len(cmd.Submissions) == 0 {
		return errs.FinalAddForProperty("batch_martket_instructions", ErrEmptyBatchMarketInstructions)
	}

	return errs
}
