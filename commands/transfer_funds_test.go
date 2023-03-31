package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestNilTransferFundsFails(t *testing.T) {
	err := checkTransfer(nil)

	assert.Contains(t, err.Get("transfer"), commands.ErrIsRequired)
}

func TestTransferFunds(t *testing.T) {
	cases := []struct {
		transfer  commandspb.Transfer
		errString string
	}{
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_MARGIN,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.from_account_type (is not a valid value)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "",
			},
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.to (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.asset (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "",
				Reference: "testing",
			},
			errString: "transfer.amount (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "-1",
				Reference: "testing",
			},
			errString: "transfer.amount (must be positive)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "0",
				Reference: "testing",
			},
			errString: "transfer.amount (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtestingtest",
			},
			errString: "transfer.reference (must be less than 100 characters)",
		},
		{
			transfer: commandspb.Transfer{
				To:        "",
				Asset:     "",
				Amount:    "",
				Reference: "",
			},
			errString: "transfer.amount (is required), transfer.asset (is required), transfer.from_account_type (is not a valid value), transfer.kind (is required), transfer.to (is required), transfer.to_account_type (is not a valid value)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{
						DeliverOn: -1,
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.deliver_on (must be positive or zero)",
		},
		{
			transfer: commandspb.Transfer{
				ToAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.from_account_type (is not a valid value)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.to_account_type (is not a valid value)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.factor (not a valid float), transfer.kind.start_epoch (must be positive)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
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
			errString: "transfer.kind.start_epoch (must be positive)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
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
			errString: "transfer.kind.end_epoch (must be positive, must be after start_epoch)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
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
			errString: "transfer.kind.factor (must be positive)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
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
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.account.to (transfers to metric-based reward accounts must be recurring transfers that specify a distribution metric)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.account.to (transfers to metric-based reward accounts must be recurring transfers that specify a distribution metric)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.account.to (transfers to metric-based reward accounts must be recurring transfers that specify a distribution metric)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.account.to (transfers to metric-based reward accounts must be recurring transfers that specify a distribution metric)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "0000000000000000000000000000000000000000000000000000000000000000",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
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
			errString: "transfer.kind.dispatch_strategy.asset_for_metric (unknown asset)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
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

	invalidAccountTypesForOneOff := []vega.AccountType{
		vega.AccountType_ACCOUNT_TYPE_INSURANCE,
		vega.AccountType_ACCOUNT_TYPE_SETTLEMENT,
		vega.AccountType_ACCOUNT_TYPE_MARGIN,
		vega.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE,
		vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
		vega.AccountType_ACCOUNT_TYPE_FEES_MAKER,
		vega.AccountType_ACCOUNT_TYPE_BOND,
		vega.AccountType_ACCOUNT_TYPE_EXTERNAL,
		vega.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE,
		vega.AccountType_ACCOUNT_TYPE_PENDING_TRANSFERS,
		vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES,
		vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
		vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
		vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
	}

	for _, at := range invalidAccountTypesForOneOff {
		cases = append(cases, struct {
			transfer  commandspb.Transfer
			errString string
		}{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   at,
				Kind: &commandspb.Transfer_OneOff{
					OneOff: &commandspb.OneOffTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.to_account_type (account type is not valid for one off transfer)",
		})
	}

	for _, c := range cases {
		err := commands.CheckTransfer(&c.transfer)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.Contains(t, err.Error(), c.errString)
	}
}

func checkTransfer(cmd *commandspb.Transfer) commands.Errors {
	err := commands.CheckTransfer(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
