package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckCancelTransfer(cmd *commandspb.CancelTransfer) error {
	return checkCancelTransfer(cmd).ErrorOrNil()
}

func checkCancelTransfer(cmd *commandspb.CancelTransfer) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("cancel_transfer", ErrIsRequired)
	}

	if len(cmd.TransferId) <= 0 {
		errs.AddForProperty("cancel_transfer.transfer_id", ErrIsRequired)
	} else if !IsVegaID(cmd.TransferId) {
		errs.AddForProperty("cancel_transfer.transfer_id", ErrShouldBeAValidVegaID)
	}

	return errs
}
