package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestNilWithdrawSubmissionFails(t *testing.T) {
	err := checkWithdrawSubmission(nil)

	assert.Contains(t, err.Get("withdraw_submission"), commands.ErrIsRequired)
}

func TestWithdrawSubmission(t *testing.T) {
	cases := []struct {
		withdraw  commandspb.WithdrawSubmission
		errString string
	}{
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: "100",
				Asset:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: "100",
				Asset:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
				Ext: &types.WithdrawExt{
					Ext: &types.WithdrawExt_Erc20{
						Erc20: &types.Erc20WithdrawExt{
							ReceiverAddress: "0x9135f5afd6F055e731bca2348429482eE614CFfA",
						},
					},
				},
			},
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: "100",
				Asset:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
				Ext: &types.WithdrawExt{
					Ext: &types.WithdrawExt_Erc20{
						Erc20: &types.Erc20WithdrawExt{
							ReceiverAddress: "0xsomething",
						},
					},
				},
			},
			errString: "withdraw_ext.erc20.received_address (is not a valid ethereum address)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Asset: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "withdraw_submission.amount (is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: "100",
			},
			errString: "withdraw_submission.asset (is required)",
		},
		{
			withdraw:  commandspb.WithdrawSubmission{},
			errString: "withdraw_submission.amount (is required), withdraw_submission.asset (is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Ext: &types.WithdrawExt{},
			},
			errString: "withdraw_ext.ext (unsupported withdraw extended details), withdraw_submission.amount (is required), withdraw_submission.asset (is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: "100",
				Asset:  "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
				Ext: &types.WithdrawExt{
					Ext: &types.WithdrawExt_Erc20{
						Erc20: &types.Erc20WithdrawExt{},
					},
				},
			},
			errString: "withdraw_ext.erc20.received_address (is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Ext: &types.WithdrawExt{
					Ext: &types.WithdrawExt_Erc20{
						Erc20: &types.Erc20WithdrawExt{},
					},
				},
			},
			errString: "withdraw_ext.erc20.received_address (is required), withdraw_submission.amount (is required), withdraw_submission.asset (is required)",
		},
	}

	for _, c := range cases {
		err := commands.CheckWithdrawSubmission(&c.withdraw)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.Error(t, err)
		assert.EqualError(t, err, c.errString)
	}
}

func checkWithdrawSubmission(cmd *commandspb.WithdrawSubmission) commands.Errors {
	err := commands.CheckWithdrawSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
