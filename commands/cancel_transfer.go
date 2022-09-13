package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckCancelTransferInstruction(cmd *commandspb.CancelTransferInstruction) error {
	return checkCancelTransferInstruction(cmd).ErrorOrNil()
}

func checkCancelTransferInstruction(cmd *commandspb.CancelTransferInstruction) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("cancel_transfer_instruction", ErrIsRequired)
	}

	if len(cmd.TransferInstructionId) <= 0 {
		errs.AddForProperty("cancel_transfer_instruction.transfer_id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.TransferInstructionId) {
		errs.AddForProperty("cancel_transfer_instruction.transfer_id", ErrShouldBeAValidVegaID)
	}

	return errs
}
