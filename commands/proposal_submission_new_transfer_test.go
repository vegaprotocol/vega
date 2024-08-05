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
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/crypto"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/require"
)

func TestCheckProposalSubmissionForNewTransfer(t *testing.T) {
	t.Run("Submitting a new transfer change without a transfer market fails", testNewTransferChangeSubmissionWithoutTransferFails)
	t.Run("Submitting a new transfer change without changes fails", testNewTransferChangeSubmissionWithoutChangesFails)
	t.Run("Submitting a new transfer change without source account type fails", testNewTransferChangeSubmissionWithoutSourceTypeFails)
	t.Run("Submitting a new transfer change with an invalid source account type fails", testNewTransferChangeSubmissionInvalidSourceTypeFails)
	t.Run("Submitting a new transfer change without destination account type fails", testNewTransferChangeSubmissionWithoutDestinationTypeFails)
	t.Run("Submitting a new transfer change with an invalid destination account type fails", testNewTransferChangeSubmissionInvalidDestinationTypeFails)
	t.Run("Submitting a new transfer change with an invalid source fails", testNewTransferChangeSubmissionInvalidSourceFails)
	t.Run("Submitting a new transfer change with an invalid destination fails", testNewTransferChangeSubmissionInvalidDestinationFails)
	t.Run("Submitting a new transfer change with an invalid governance transfer type fails", testNewTransferChangeSubmissionInvalidTransferTypeFails)
	t.Run("Submitting a new transfer change with an invalid amounts fails", testNewTransferChangeSubmissionInvalidAmountFails)
	t.Run("Submitting a new transfer change with an invalid asset fails", testNewTransferChangeSubmissionInvalidAseetFails)
	t.Run("Submitting a new transfer change with an invalid fraction fails", testNewTransferChangeSubmissionInvalidFractionFails)
	t.Run("Submitting a new transfer change with neither one off nor recurring fails", testNewTransferWithNoKind)
	t.Run("Submitting a new transfer change with recurring end epoch before the start epoch", testNewRecurringGovernanceTransferInvalidEndEpoch)
	t.Run("Submitting a new transfer change with identical source/destination accounts", testNewTransferChangeSubmissionIneffectualTransferFails)
	t.Run("Submitting a cancel transfer change with missing transfer id fails", testCancelTransferChangeSubmission)
	t.Run("Submitting a new transfer change with an invalid destination type for one off transfer", testOneOffWithInvalidDestinationType)
	t.Run("Submitting a new transfer change with an negative deliverOn", testOneOffWithNegativeDeliverOn)
	t.Run("Submitting a new recurring transfer change with a dispatch strategy and destination party", testRecurringWithDestinationAndDispatch)
	t.Run("Submitting a new recurring transfer change with a dispatch strategy and an invalid type", testRecurringWithDispatchInvalidTypes)
	t.Run("Submitting a new recurring transfer change with a dispatch strategy and incompatible empty asset for metric", testInvalidAssetForMetric)
	t.Run("Submitting a new recurring transfer change with a dispatch strategy and mismatching destination type for metric", testInvalidDestForMetric)
	t.Run("Submitting a new transfer change with destination type general and an invalid vega public key", testInvalidGeneralPubKey)
	t.Run("Submitting a new transfer change with destination type general and an invalid vega public key", testOnlyGeneralValid)
	t.Run("Submitting a new transfer change with invalid decay factor", testInvalidDecayFactor)
}

func testInvalidDestForMetric(t *testing.T) {
	metricsMismatches := map[types.AccountType][]types.DispatchMetric{
		types.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES: {
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN,
		},
		types.AccountType_ACCOUNT_TYPE_REWARD_REALISED_RETURN: {
			types.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID,
			types.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED,
			types.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			types.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL,
			types.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
			types.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY,
			types.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
		},
	}

	for tp, metrics := range metricsMismatches {
		for _, metric := range metrics {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewTransfer{
						NewTransfer: &types.NewTransfer{
							Changes: &types.NewTransferConfiguration{
								FractionOfBalance: "0.5",
								Amount:            "1000",
								SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
								DestinationType:   tp,
								Destination:       "",
								TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
								Asset:             "abcde",
								Kind: &types.NewTransferConfiguration_Recurring{
									Recurring: &types.RecurringTransfer{
										StartEpoch: 10,
										DispatchStrategy: &types.DispatchStrategy{
											Metric: metric,
										},
									},
								},
							},
						},
					},
				},
			})
			require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.recurring.dispatch_strategy.dispatch_metric"), errors.New("cannot set toAccountType to "+tp.String()+" when dispatch metric is set to "+metric.String()))
		}
	}
}

func testInvalidAssetForMetric(t *testing.T) {
	invalidTypes := []types.AccountType{
		types.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
		types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES,
		types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES,
		types.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL,
		types.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN,
		types.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY,
		types.AccountType_ACCOUNT_TYPE_REWARD_REALISED_RETURN,
	}

	for _, inv := range invalidTypes {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							FractionOfBalance: "0.5",
							Amount:            "1000",
							SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
							DestinationType:   inv,
							Destination:       "",
							TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
							Asset:             "abcde",
							Kind: &types.NewTransferConfiguration_Recurring{
								Recurring: &types.RecurringTransfer{
									StartEpoch:       10,
									DispatchStrategy: &types.DispatchStrategy{},
								},
							},
						},
					},
				},
			},
		})

		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.recurring.dispatch_strategy.asset_for_metric"), commands.ErrIsRequired)
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES,
						Destination:       "",
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
						Kind: &types.NewTransferConfiguration_Recurring{
							Recurring: &types.RecurringTransfer{
								StartEpoch: 10,
								DispatchStrategy: &types.DispatchStrategy{
									AssetForMetric: "zohar",
								},
							},
						},
					},
				},
			},
		},
	})

	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.recurring.dispatch_strategy.asset_for_metric"), commands.ErrShouldBeAValidVegaID, err.Error())
}

func testRecurringWithDispatchInvalidTypes(t *testing.T) {
	invalidTypes := make(map[types.AccountType]struct{}, len(types.AccountType_name))
	for k := range types.AccountType_name {
		invalidTypes[types.AccountType(k)] = struct{}{}
	}
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_REALISED_RETURN)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_INSURANCE)
	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_GENERAL)

	delete(invalidTypes, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)

	for inv := range invalidTypes {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							FractionOfBalance: "0.5",
							Amount:            "1000",
							SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
							DestinationType:   inv,
							Destination:       "",
							TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
							Asset:             "abcde",
							Kind: &types.NewTransferConfiguration_Recurring{
								Recurring: &types.RecurringTransfer{
									StartEpoch:       10,
									DispatchStrategy: &types.DispatchStrategy{},
								},
							},
						},
					},
				},
			},
		})
		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination_type"), commands.ErrIsNotValid, inv.String())
	}
}

func testRecurringWithDestinationAndDispatch(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
						Destination:       "zohar",
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
						Kind: &types.NewTransferConfiguration_Recurring{
							Recurring: &types.RecurringTransfer{
								StartEpoch:       10,
								DispatchStrategy: &types.DispatchStrategy{},
							},
						},
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination"), commands.ErrIsNotValid)
}

func testOneOffWithNegativeDeliverOn(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
						Destination:       "zohar",
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
						Kind: &types.NewTransferConfiguration_OneOff{
							OneOff: &types.OneOffTransfer{
								DeliverOn: -100,
							},
						},
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.oneoff.deliveron"), commands.ErrMustBePositiveOrZero)
}

func testInvalidDecayFactor(t *testing.T) {
	decayFactors := []string{"", "-0.1", "bc", "0"}
	errors := []string{
		"proposal_submission.terms.change.new_transfer.changes.recurring.factor (not a valid float)",
		"proposal_submission.terms.change.new_transfer.changes.recurring.factor (must be positive)",
		"proposal_submission.terms.change.new_transfer.changes.recurring.factor (not a valid float)",
		"proposal_submission.terms.change.new_transfer.changes.recurring.factor (must be positive)",
	}
	for i, decay := range decayFactors {
		prop := &commandspb.ProposalSubmission{
			Rationale: &types.ProposalRationale{
				Description: "valid",
				Title:       "test",
			},
			Terms: &types.ProposalTerms{
				ClosingTimestamp:   time.Now().Unix() + 100,
				EnactmentTimestamp: time.Now().Unix() + 200,
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							FractionOfBalance: "0.5",
							Amount:            "1000",
							SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
							DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
							Destination:       crypto.RandomHash(),
							TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
							Asset:             "abcde",
							Kind: &types.NewTransferConfiguration_Recurring{
								Recurring: &types.RecurringTransfer{
									StartEpoch: 0,
									Factor:     decay,
								},
							},
						},
					},
				},
			},
		}
		err := checkProposalSubmission(prop)
		require.Equal(t, errors[i], err.Error())
	}
	prop := &commandspb.ProposalSubmission{
		Rationale: &types.ProposalRationale{
			Description: "valid",
			Title:       "test",
		},
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   time.Now().Unix() + 100,
			EnactmentTimestamp: time.Now().Unix() + 200,
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
						Destination:       crypto.RandomHash(),
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
						Kind: &types.NewTransferConfiguration_Recurring{
							Recurring: &types.RecurringTransfer{
								StartEpoch: 0,
								Factor:     "0.5",
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(prop)
	require.True(t, err.Empty())
}

func testOnlyGeneralValid(t *testing.T) {
	partyAccs := []types.AccountType{
		types.AccountType_ACCOUNT_TYPE_MARGIN,
		types.AccountType_ACCOUNT_TYPE_BOND,
	}
	// start with a valid transfer to the general account
	prop := &commandspb.ProposalSubmission{
		Rationale: &types.ProposalRationale{
			Description: "valid",
			Title:       "test",
		},
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   time.Now().Unix() + 100,
			EnactmentTimestamp: time.Now().Unix() + 200,
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
						Destination:       crypto.RandomHash(),
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
						Kind: &types.NewTransferConfiguration_OneOff{
							OneOff: &types.OneOffTransfer{
								DeliverOn: 0,
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(prop)
	require.True(t, err.Empty())
	// none of the other accounts are valid destination types:
	for _, at := range partyAccs {
		prop := &commandspb.ProposalSubmission{
			Rationale: &types.ProposalRationale{
				Description: "invalid",
				Title:       "test",
			},
			Terms: &types.ProposalTerms{
				ClosingTimestamp:   time.Now().Unix() + 100,
				EnactmentTimestamp: time.Now().Unix() + 200,
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							FractionOfBalance: "0.5",
							Amount:            "1000",
							SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
							DestinationType:   at,
							Destination:       crypto.RandomHash(), // ensure a valid hash
							TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
							Asset:             "abcde",
							Kind: &types.NewTransferConfiguration_OneOff{
								OneOff: &types.OneOffTransfer{
									DeliverOn: 0,
								},
							},
						},
					},
				},
			},
		}
		err = checkProposalSubmission(prop)
		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination_type"), commands.ErrIsNotValid)
	}
}

func testInvalidGeneralPubKey(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
						Destination:       "zohar",
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
						Kind: &types.NewTransferConfiguration_OneOff{
							OneOff: &types.OneOffTransfer{
								DeliverOn: -100,
							},
						},
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination"), commands.ErrShouldBeAValidVegaPublicKey)
}

func testOneOffWithInvalidDestinationType(t *testing.T) {
	dests := []types.AccountType{
		types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
		types.AccountType_ACCOUNT_TYPE_SETTLEMENT,
		types.AccountType_ACCOUNT_TYPE_MARGIN,
		types.AccountType_ACCOUNT_TYPE_BOND,
		types.AccountType_ACCOUNT_TYPE_FEES_MAKER,
		types.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE,
		types.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
	}

	for _, dest := range dests {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							FractionOfBalance: "0.5",
							Amount:            "1000",
							SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
							DestinationType:   dest,
							TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
							Asset:             "abcde",
							Kind: &types.NewTransferConfiguration_OneOff{
								OneOff: &types.OneOffTransfer{},
							},
						},
					},
				},
			},
		})
		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination_type"), commands.ErrIsNotValid)
	}
}

func testNewRecurringGovernanceTransferInvalidEndEpoch(t *testing.T) {
	endEpoch := uint64(8)
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
						Destination:       crypto.RandomHash(),
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
						Kind: &types.NewTransferConfiguration_Recurring{
							Recurring: &types.RecurringTransfer{
								StartEpoch: 10,
								EndEpoch:   &endEpoch,
							},
						},
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.recurring.end_epoch"), commands.ErrIsNotValid)
}

func testNewTransferWithNoKind(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						FractionOfBalance: "0.5",
						Amount:            "1000",
						SourceType:        types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
						Destination:       crypto.RandomHash(),
						TransferType:      types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
						Asset:             "abcde",
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.kind"), commands.ErrIsRequired)
}

func testNewTransferChangeSubmissionWithoutTransferFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{},
		},
	})

	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer"), commands.ErrIsRequired)
}

func testNewTransferChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{},
			},
		},
	})

	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes"), commands.ErrIsRequired)
}

func testNewTransferChangeSubmissionWithoutSourceTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{},
				},
			},
		},
	})

	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.source_type"), commands.ErrIsRequired)
}

func testNewTransferChangeSubmissionWithoutDestinationTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						SourceType: types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
					},
				},
			},
		},
	})

	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination_type"), commands.ErrIsRequired)
}

func testNewTransferChangeSubmissionInvalidDestinationTypeFails(t *testing.T) {
	allAccountTypes := make(map[int32]struct{}, len(types.AccountType_name))
	for k := range types.AccountType_name {
		allAccountTypes[k] = struct{}{}
	}
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_GENERAL))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_INSURANCE))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_REWARD_REALISED_RETURN))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_UNSPECIFIED))

	for at := range allAccountTypes {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							SourceType:      types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
							DestinationType: types.AccountType(at),
						},
					},
				},
			},
		})
		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination_type"), commands.ErrIsNotValid)
	}
	validDestinationAccountTypes := []types.AccountType{types.AccountType_ACCOUNT_TYPE_GENERAL, types.AccountType_ACCOUNT_TYPE_INSURANCE}
	for _, at := range validDestinationAccountTypes {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							SourceType:      types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
							DestinationType: at,
						},
					},
				},
			},
		})
		require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination_type"), commands.ErrIsNotValid)
	}
}

func testNewTransferChangeSubmissionInvalidSourceTypeFails(t *testing.T) {
	allAccountTypes := make(map[int32]struct{}, len(types.AccountType_name))
	for k := range types.AccountType_name {
		allAccountTypes[k] = struct{}{}
	}
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_INSURANCE))
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_UNSPECIFIED))

	for at := range allAccountTypes {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							SourceType: types.AccountType(at),
						},
					},
				},
			},
		})
		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.source_type"), commands.ErrIsNotValid, types.AccountType(at).String())
	}

	validSourceAccountTypes := []types.AccountType{types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY, types.AccountType_ACCOUNT_TYPE_INSURANCE}
	for _, at := range validSourceAccountTypes {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							SourceType: at,
						},
					},
				},
			},
		})
		require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.source_type"), commands.ErrIsNotValid)
	}
}

func testNewTransferChangeSubmissionInvalidSourceFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						SourceType:      types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
						Source:          "some source",
						DestinationType: types.AccountType_ACCOUNT_TYPE_GENERAL,
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.source"), commands.ErrIsNotValid)
}

func testNewTransferChangeSubmissionInvalidDestinationFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						SourceType:      types.AccountType_ACCOUNT_TYPE_INSURANCE,
						Source:          "some market",
						DestinationType: types.AccountType_ACCOUNT_TYPE_INSURANCE,
						Destination:     "",
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination"), commands.ErrIsNotValid, err.Error())
}

func testCancelTransferChangeSubmission(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_CancelTransfer{},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.cancel_transfer"), commands.ErrIsRequired)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_CancelTransfer{
				CancelTransfer: &types.CancelTransfer{},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.cancel_transfer.changes"), commands.ErrIsRequired)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_CancelTransfer{
				CancelTransfer: &types.CancelTransfer{
					Changes: &types.CancelTransferConfiguration{},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.cancel_transfer.changes.transferId"), commands.ErrIsRequired)
}

func testNewTransferChangeSubmissionIneffectualTransferFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: &types.NewTransferConfiguration{
						SourceType:      types.AccountType_ACCOUNT_TYPE_INSURANCE,
						DestinationType: types.AccountType_ACCOUNT_TYPE_INSURANCE,
						Source:          "some destination",
						Destination:     "some destination",
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination"), commands.ErrIsNotValid)
}

func testNewTransferChangeSubmissionInvalidTransferTypeFails(t *testing.T) {
	expectation := map[types.GovernanceTransferType]bool{
		types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_UNSPECIFIED:    true,
		types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING: false,
		types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_BEST_EFFORT:    false,
	}
	for tp, expectedError := range expectation {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							SourceType:      types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
							DestinationType: types.AccountType_ACCOUNT_TYPE_GENERAL,
							Destination:     crypto.RandomHash(),
							TransferType:    tp,
						},
					},
				},
			},
		})
		if expectedError {
			require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.transfer_type"), commands.ErrIsRequired)
		} else {
			require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.transfer_type"), commands.ErrIsRequired)
		}
	}
}

func testNewTransferChangeSubmissionInvalidAmountFails(t *testing.T) {
	transfer := &types.NewTransferConfiguration{
		SourceType:      types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
		DestinationType: types.AccountType_ACCOUNT_TYPE_GENERAL,
		Destination:     crypto.RandomHash(),
		TransferType:    types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
	}
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.amount"), commands.ErrIsRequired)
	transfer.Amount = "abc"
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.amount"), commands.ErrIsNotValid)
	transfer.Amount = "500.1234"
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.amount"), commands.ErrIsNotValid)

	transfer.Amount = "-500"
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.amount"), commands.ErrIsNotValid)

	transfer.Amount = "500"
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.amount"), commands.ErrIsNotValid)
}

func testNewTransferChangeSubmissionInvalidAseetFails(t *testing.T) {
	transfer := &types.NewTransferConfiguration{
		SourceType:      types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
		DestinationType: types.AccountType_ACCOUNT_TYPE_GENERAL,
		Destination:     crypto.RandomHash(),
		TransferType:    types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
		Amount:          "500",
	}
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.asset"), commands.ErrIsRequired)
	transfer.Asset = "abcde"
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.asset"), commands.ErrIsRequired)
}

func testNewTransferChangeSubmissionInvalidFractionFails(t *testing.T) {
	expectation := map[string]error{
		"":     commands.ErrIsRequired,
		"abc":  commands.ErrIsNotValid,
		"-1":   commands.ErrMustBePositive,
		"1.01": commands.ErrMustBeLTE1,
	}

	transfer := &types.NewTransferConfiguration{
		SourceType:      types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY,
		DestinationType: types.AccountType_ACCOUNT_TYPE_GENERAL,
		Destination:     crypto.RandomHash(),
		TransferType:    types.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
		Amount:          "500",
		Asset:           "abcde",
	}
	for fraction, expectedErr := range expectation {
		transfer.FractionOfBalance = fraction
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: transfer,
					},
				},
			},
		})
		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance"), expectedErr)
	}
	transfer.FractionOfBalance = "0.5"
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewTransfer{
				NewTransfer: &types.NewTransfer{
					Changes: transfer,
				},
			},
		},
	})
	require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance"), commands.ErrMustBePositive)
	require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance"), commands.ErrIsNotValid)
	require.NotContains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance"), commands.ErrIsRequired)
}
