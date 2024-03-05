// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestNilTransferFundsFails(t *testing.T) {
	err := checkTransfer(nil)

	assert.Contains(t, err.Get("transfer"), commands.ErrIsRequired)
}

func TestTransferFunds(t *testing.T) {
	largeRankTable := make([]*vega.Rank, 0, 501)
	for i := uint32(0); i < 501; i++ {
		largeRankTable = append(largeRankTable, &vega.Rank{StartRank: i, ShareRatio: i})
	}

	capInvalidNumber := "banana"
	capNegative := "-1"
	capZero := "0"

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
						EndEpoch:   ptr.From(uint64(10)),
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
						EndEpoch:   ptr.From(uint64(0)),
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
						EndEpoch:   ptr.From(uint64(11)),
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
						EndEpoch:   ptr.From(uint64(11)),
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
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.asset_for_metric (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.entity_scope (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							CapRewardFeeMultiple: &capInvalidNumber,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.cap_reward_fee_multiple (is not a valid number)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							CapRewardFeeMultiple: &capNegative,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.cap_reward_fee_multiple (must be positive)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							CapRewardFeeMultiple: &capZero,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.cap_reward_fee_multiple (must be positive)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.entity_scope (ENTITY_SCOPE_TEAMS is not allowed for ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:  "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_ALL,
						},
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
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:  "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_ALL,
							Markets:         []string{"market1", "market2"},
						},
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
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_ALL,
							Markets:         []string{"market1", "market2"},
						},
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
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_ALL,
						},
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
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.n_top_performers (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							NTopPerformers:       "5",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.n_top_performers (must not be set when entity scope is not ENTITY_SCOPE_TEAMS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
							NTopPerformers:       "banana",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.n_top_performers (is not a valid number)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
							NTopPerformers:       "-0.5",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.n_top_performers (must be between 0 (excluded) and 1 (included))",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
							NTopPerformers:       "1.1",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.n_top_performers (must be between 0 (excluded) and 1 (included))",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.individual_scope (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_ALL,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.individual_scope (should not be set when entity_scope is set to ENTITY_SCOPE_TEAMS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:  "",
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.distribution_strategy (is required)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.distribution_strategy (should not be set when to_account is set to ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:     "",
							Metric:             vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:        vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:    vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							StakingRequirement: "banana",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.staking_requirement (not a valid integer)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:     "",
							Metric:             vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:        vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:    vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							StakingRequirement: "-1",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.staking_requirement (must be positive or zero)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:     "",
							Metric:             vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:        vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:    vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							StakingRequirement: "1",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.staking_requirement (should not be set if to_account is set to ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:     "",
							Metric:             vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:        vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:    vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							StakingRequirement: "1",
							NotionalTimeWeightedAveragePositionRequirement: "banana",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.notional_time_weighted_average_position_requirement (not a valid integer)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:  "",
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							NotionalTimeWeightedAveragePositionRequirement: "-1",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.notional_time_weighted_average_position_requirement (must be positive or zero)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:  "",
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							NotionalTimeWeightedAveragePositionRequirement: "1",
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.notional_time_weighted_average_position_requirement (should not be set if to_account is set to ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:  "",
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							WindowLength:    10,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.window_length (should not be set for ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:  "",
							Metric:          vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:     vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope: vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							WindowLength:    101,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.window_length (must be at most 100)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.rank_table (must be positive)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							RankTable: []*vega.Rank{
								{StartRank: 1, ShareRatio: 10},
							},
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.rank_table (should not be set for distribution strategy DISTRIBUTION_STRATEGY_PRO_RATA)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
							RankTable:            largeRankTable,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.rank_table (must be at most 500)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
							RankTable: []*vega.Rank{
								{StartRank: 1, ShareRatio: 10},
								{StartRank: 3, ShareRatio: 5},
								{StartRank: 2, ShareRatio: 2},
							},
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.rank_table.%!i(int=2).start_rank (must be greater than start_rank of element #1)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
							RankTable: []*vega.Rank{
								{StartRank: 1, ShareRatio: 10},
								{StartRank: 2, ShareRatio: 5},
								{StartRank: 2, ShareRatio: 2},
							},
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.rank_table.%!i(int=2).start_rank (must be greater than start_rank of element #1)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							StakingRequirement:   "1",
							NotionalTimeWeightedAveragePositionRequirement: "2",
							WindowLength: 0,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.window_length (must bet between 1 and 100)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							StakingRequirement:   "1",
							NotionalTimeWeightedAveragePositionRequirement: "2",
							WindowLength: 1,
						},
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
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
							IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
							TeamScope:            []string{"zohafr"},
							StakingRequirement:   "1",
							NotionalTimeWeightedAveragePositionRequirement: "2",
							WindowLength: 1,
						},
					},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1",
				Reference: "testing",
			},
			errString: "transfer.kind.dispatch_strategy.team_scope (should not be set when entity_scope is set to ENTITY_SCOPE_INDIVIDUALS)",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_VESTED_REWARDS,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_MARGIN,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{},
				},
				To:        "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				Asset:     "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
				Amount:    "1000",
				Reference: "testing",
			},
			errString: "transfer.from_account_type (account type is not valid for one recurring transfer",
		},
		{
			transfer: commandspb.Transfer{
				FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
				ToAccountType:   vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
				Kind: &commandspb.Transfer_Recurring{
					Recurring: &commandspb.RecurringTransfer{
						StartEpoch: 10,
						EndEpoch:   ptr.From(uint64(11)),
						Factor:     "1",
						DispatchStrategy: &vega.DispatchStrategy{
							AssetForMetric:       "080538b7cc2249de568cb4272a17f4d5e0b0a69a1a240acbf5119d816178daff",
							Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
							DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA,
							EntityScope:          vega.EntityScope_ENTITY_SCOPE_TEAMS,
							TeamScope:            []string{"team1"},
							StakingRequirement:   "1",
							NotionalTimeWeightedAveragePositionRequirement: "2",
							WindowLength:   1,
							NTopPerformers: "0.5",
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
		vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION,
		vega.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN,
		vega.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY,
		vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING,
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
			assert.NoError(t, err, c.transfer.String())
			continue
		}
		assert.Contains(t, err.Error(), c.errString)
	}
}

func checkTransfer(cmd *commandspb.Transfer) commands.Errors {
	err := commands.CheckTransfer(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
