package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestNilTransferFundsFails(t *testing.T) {
	err := checkTransferInstruction(nil)

	assert.Contains(t, err.Get("transfer"), commands.ErrIsRequired)
}

func TestTransferInstructionFunds(t *testing.T) {
	cases := []struct {
		transferInstruction commandspb.TransferInstruction
		errString           string
	}{
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "",
			},
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.to (is required)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.asset (is required)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "",
				Reference: "testing",
			},
			errString: "transfer_instruction.amount (is required)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "-1",
				Reference: "testing",
			},
			errString: "transfer_instruction.amount (must be positive)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "0",
				Reference: "testing",
			},
			errString: "transfer_instruction.amount (is required)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtest",
			},
			errString: "transfer_instruction.reference (must be less than 100 characters)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				To:        "",
				Asset:     "",
				Amount:    "",
				Reference: "",
			},
			errString: "transfer_instruction.amount (is required), transfer_instruction.asset (is required), transfer_instruction.from_account_type (is not a valid value), transfer_instruction.kind (is required), transfer_instruction.to (is required), transfer_instruction.to_account_type (is not a valid value)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{
						DeliverOn: -1,
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.kind.deliver_on (must be positive or zero)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				ToAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.from_account_type (is not a valid value)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.to_account_type (is not a valid value)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_Recurring{
					Recurring: &commandspb.RecurringTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.kind.factor (not a valid float), transfer_instruction.kind.start_epoch (must be positive)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_Recurring{
					Recurring: &commandspb.RecurringTransferInstruction{
						StartEpoch: 0,
						EndEpoch:   Uint64Ptr(10),
						Factor:     "1",
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.kind.start_epoch (must be positive)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_Recurring{
					Recurring: &commandspb.RecurringTransferInstruction{
						StartEpoch: 10,
						EndEpoch:   Uint64Ptr(0),
						Factor:     "1",
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.kind.end_epoch (must be positive, must be after start_epoch)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_Recurring{
					Recurring: &commandspb.RecurringTransferInstruction{
						StartEpoch: 10,
						EndEpoch:   Uint64Ptr(11),
						Factor:     "-1",
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.kind.factor (must be positive)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.TransferInstruction_Recurring{
					Recurring: &commandspb.RecurringTransferInstruction{
						StartEpoch: 10,
						EndEpoch:   Uint64Ptr(11),
						Factor:     "0.01",
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.account.to (transfer instructions to metric-based reward accounts must be recurring transfer instructions that specify a distribution metric)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.account.to (transfer instructions to metric-based reward accounts must be recurring transfer instructions that specify a distribution metric)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.account.to (transfers to metric-based reward accounts must be recurring transfers that specify a distribution metric)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.account.to (transfer instructions to metric-based reward accounts must be recurring transfer instructions that specify a distribution metric)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
				Kind: &commandspb.TransferInstruction_OneOff{
					OneOff: &commandspb.OneOffTransferInstruction{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.TransferInstruction_Recurring{
					Recurring: &commandspb.RecurringTransferInstruction{
						StartEpoch: 10,
						EndEpoch:   Uint64Ptr(11),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric: "",
							Metric:         vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer_instruction.kind.dispatch_strategy.asset_for_metric (unknown asset)",
		},
		{
			transferInstruction: commandspb.TransferInstruction{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.TransferInstruction_Recurring{
					Recurring: &commandspb.RecurringTransferInstruction{
						StartEpoch: 10,
						EndEpoch:   Uint64Ptr(11),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric: "",
							Metric:         vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
		},
	}

	for _, c := range cases {
		err := commands.CheckTransferInstruction(&c.transferInstruction)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.EqualError(t, err, c.errString)
	}
}

func checkTransferInstruction(cmd *commandspb.TransferInstruction) commands.Errors {
	err := commands.CheckTransferInstruction(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
