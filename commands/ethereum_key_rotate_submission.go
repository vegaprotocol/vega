package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckEthereumKeyRotateSubmission(cmd *commandspb.EthereumKeyRotateSubmission) error {
	return checkEthereumKeyRotateSubmission(cmd).ErrorOrNil()
}

func checkEthereumKeyRotateSubmission(cmd *commandspb.EthereumKeyRotateSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("ethereum_key_rotate_submission", ErrIsRequired)
	}

	if len(cmd.NewAddress) <= 0 {
		errs.AddForProperty("ethereum_key_rotate_submission.new_address", ErrIsRequired)
	}

	if len(cmd.CurrentAddress) <= 0 {
		errs.AddForProperty("ethereum_key_rotate_submission.current_address", ErrIsRequired)
	}

	if cmd.TargetBlock == 0 {
		errs.AddForProperty("ethereum_key_rotate_submission.target_block", ErrIsRequired)
	}

	if cmd.EthereumSignature == nil {
		errs.AddForProperty("ethereum_key_rotate_submission.signature", ErrIsRequired)
	}

	return errs
}
