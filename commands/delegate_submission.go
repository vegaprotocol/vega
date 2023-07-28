package commands

import (
	"math/big"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckDelegateSubmission(cmd *commandspb.DelegateSubmission) error {
	return checkDelegateSubmission(cmd).ErrorOrNil()
}

func checkDelegateSubmission(cmd *commandspb.DelegateSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("delegate_submission", ErrIsRequired)
	}

	if len(cmd.Amount) <= 0 {
		errs.AddForProperty("delegate_submission.amount", ErrIsRequired)
	} else {
		if amount, ok := big.NewInt(0).SetString(cmd.Amount, 10); !ok {
			errs.AddForProperty("delegate_submission.amount", ErrNotAValidInteger)
		} else {
			if amount.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("delegate_submission.amount", ErrIsRequired)
			}
		}
	}

	if len(cmd.NodeId) <= 0 {
		errs.AddForProperty("delegate_submission.node_id", ErrIsRequired)
	} else if !IsVegaPublicKey(cmd.NodeId) {
		errs.AddForProperty("delegate_submission.node_id", ErrShouldBeAValidVegaPublicKey)
	}

	return errs
}

func CheckUndelegateSubmission(cmd *commandspb.UndelegateSubmission) error {
	return checkUndelegateSubmission(cmd).ErrorOrNil()
}

func checkUndelegateSubmission(cmd *commandspb.UndelegateSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("undelegate_submission", ErrIsRequired)
	}

	if _, ok := commandspb.UndelegateSubmission_Method_value[cmd.Method.String()]; !ok || cmd.Method == commandspb.UndelegateSubmission_METHOD_UNSPECIFIED {
		errs.AddForProperty("undelegate_submission.method", ErrIsRequired)
	}

	if len(cmd.NodeId) <= 0 {
		errs.AddForProperty("undelegate_submission.node_id", ErrIsRequired)
	} else if !IsVegaPublicKey(cmd.NodeId) {
		errs.AddForProperty("undelegate_submission.node_id", ErrShouldBeAValidVegaPublicKey)
	}

	return errs
}
