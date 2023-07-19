package commands_test

import (
	"testing"

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
	t.Run("Submitting a new transfer change with identifal source/destination accounts", testNewTransferChangeSubmissionIneffectualTransferFails)
	t.Run("Submitting a cancel transfer change with missing transfer id fails", testCancelTransferChangeSubmission)
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
						SourceType:        types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
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
						SourceType:        types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
						DestinationType:   types.AccountType_ACCOUNT_TYPE_GENERAL,
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
						SourceType: types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
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
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_UNSPECIFIED))

	for at := range allAccountTypes {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewTransfer{
					NewTransfer: &types.NewTransfer{
						Changes: &types.NewTransferConfiguration{
							SourceType:      types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
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
	delete(allAccountTypes, int32(types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD))
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
		require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.source_type"), commands.ErrIsNotValid)
	}

	validSourceAccountTypes := []types.AccountType{types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD, types.AccountType_ACCOUNT_TYPE_INSURANCE}
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
						SourceType:      types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
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
						DestinationType: types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
						Destination:     "some destination",
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.new_transfer.changes.destination"), commands.ErrIsNotValid)
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
							SourceType:      types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
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
		SourceType:      types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
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
		SourceType:      types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
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
		SourceType:      types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
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
