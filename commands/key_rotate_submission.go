package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckKeyRotateSubmission(cmd *commandspb.KeyRotateSubmission) error {
	return checkKeyRotateSubmission(cmd).ErrorOrNil()
}

func checkKeyRotateSubmission(cmd *commandspb.KeyRotateSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("key_rotate_submission", ErrIsRequired)
	}

	if len(cmd.NewPubKey) <= 0 {
		errs.AddForProperty("key_rotate_submission.new_pub_key", ErrIsRequired)
	}

	if len(cmd.CurrentPubKeyHash) <= 0 {
		errs.AddForProperty("key_rotate_submission.current_pub_key_hash", ErrIsRequired)
	}

	if cmd.NewPubKeyIndex == 0 {
		errs.AddForProperty("key_rotate_submission.new_pub_key_index", ErrIsRequired)
	}

	if cmd.TargetBlock == 0 {
		errs.AddForProperty("key_rotate_submission.target_block", ErrIsRequired)
	}

	return errs
}
