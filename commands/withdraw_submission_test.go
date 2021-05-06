package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestWithdrawSubmission(t *testing.T) {
	var cases = []struct {
		withdraw  commandspb.WithdrawSubmission
		errString string
	}{
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: 100,
				Asset:  "OKASSETID",
			},
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: 100,
				Asset:  "OKASSETID",
				Ext: &types.WithdrawExt{
					Ext: &types.WithdrawExt_Erc20{
						Erc20: &types.Erc20WithdrawExt{
							ReceiverAddress: "0xsomething",
						},
					},
				},
			},
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Asset: "OKASSETID",
			},
			errString: "withdraw_submission.amount(is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: 100,
			},
			errString: "withdraw_submission.asset(is required)",
		},
		{
			withdraw:  commandspb.WithdrawSubmission{},
			errString: "withdraw_submission.amount(is required), withdraw_submission.asset(is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Ext: &types.WithdrawExt{},
			},
			errString: "withdraw_ext.ext(unsupported withdraw extended details), withdraw_submission.amount(is required), withdraw_submission.asset(is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Amount: 100,
				Asset:  "OKASSETID",
				Ext: &types.WithdrawExt{
					Ext: &types.WithdrawExt_Erc20{
						Erc20: &types.Erc20WithdrawExt{},
					},
				},
			},
			errString: "withdraw_ext.erc20.received_address(is required)",
		},
		{
			withdraw: commandspb.WithdrawSubmission{
				Ext: &types.WithdrawExt{
					Ext: &types.WithdrawExt_Erc20{
						Erc20: &types.Erc20WithdrawExt{},
					},
				},
			},
			errString: "withdraw_ext.erc20.received_address(is required), withdraw_submission.amount(is required), withdraw_submission.asset(is required)",
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
