package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

// DELEGATION

func TestSubmittingNoDelegateCommandFails(t *testing.T) {
	err := checkDelegateSubmission(nil)

	assert.Contains(t, err.Get("delegate_submission"), commands.ErrIsRequired)
}

func TestSubmittingNoDelegateNodeIdFails(t *testing.T) {
	cmd := &commandspb.DelegateSubmission{
		Amount: "1000",
	}
	err := checkDelegateSubmission(cmd)

	assert.Contains(t, err.Get("delegate_submission.node_id"), commands.ErrIsRequired)
}

func TestSubmittingNoDelegateAmountFails(t *testing.T) {
	cmd := &commandspb.DelegateSubmission{
		NodeId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
	}
	err := checkDelegateSubmission(cmd)

	assert.Contains(t, err.Get("delegate_submission.amount"), commands.ErrIsRequired)
}

func checkDelegateSubmission(cmd *commandspb.DelegateSubmission) commands.Errors {
	err := commands.CheckDelegateSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}
	return e
}

// UNDELEGATION

func TestSubmittingNoUndelegateCommandFails(t *testing.T) {
	err := checkUndelegateSubmission(nil)

	assert.Contains(t, err.Get("undelegate_submission"), commands.ErrIsRequired)
}

func TestSubmittingNoUndelegateNodeIdFails(t *testing.T) {
	cmd := &commandspb.UndelegateSubmission{
		Amount: "1000",
	}
	err := checkUndelegateSubmission(cmd)

	assert.Contains(t, err.Get("undelegate_submission.node_id"), commands.ErrIsRequired)
}

func TestSubmittingInvalidUndelegateMethod(t *testing.T) {
	invalidMethod := len(commandspb.UndelegateSubmission_Method_value)
	cmd := &commandspb.UndelegateSubmission{
		NodeId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		Method: commandspb.UndelegateSubmission_Method(invalidMethod),
	}
	err := checkUndelegateSubmission(cmd)

	assert.Contains(t, err.Get("undelegate_submission.method"), commands.ErrIsRequired)

	cmd = &commandspb.UndelegateSubmission{
		NodeId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
	}
	err = checkUndelegateSubmission(cmd)

	assert.Contains(t, err.Get("undelegate_submission.method"), commands.ErrIsRequired)
}

func TestSubmittingNoUndelegateAmountSucceeds(t *testing.T) {
	cmd := &commandspb.UndelegateSubmission{
		NodeId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
		Method: 1,
	}
	err := checkUndelegateSubmission(cmd)

	assert.Equal(t, 0, len(err))
}

func checkUndelegateSubmission(cmd *commandspb.UndelegateSubmission) commands.Errors {
	err := commands.CheckUndelegateSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}
	return e
}
