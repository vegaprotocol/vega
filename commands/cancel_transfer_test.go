package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestNilCancelTransferInstructionFails(t *testing.T) {
	err := checkCancelTransferInstruction(nil)

	assert.Contains(t, err.Get("cancel_transfer_instruction"), commands.ErrIsRequired)
}

func TestCancelTransferInstruction(t *testing.T) {
	cases := []struct {
		ctransfer commandspb.CancelTransferInstruction
		errString string
	}{
		{
			ctransfer: commandspb.CancelTransferInstruction{
				TransferInstructionId: "18f8b607aad9ef2cd57f2d233766b0c576b27a3e0c50c9db713c00e518c0bbdc",
			},
		},
		{
			ctransfer: commandspb.CancelTransferInstruction{},
			errString: "cancel_transfer_instruction.transfer_id (is required)",
		},
	}

	for _, c := range cases {
		err := commands.CheckCancelTransferInstruction(&c.ctransfer)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.EqualError(t, err, c.errString)
	}
}

func checkCancelTransferInstruction(cmd *commandspb.CancelTransferInstruction) commands.Errors {
	err := commands.CheckCancelTransferInstruction(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
