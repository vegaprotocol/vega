package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForNewAsset(t *testing.T) {
	t.Run("Submitting an asset change without new asset fails", TestNewAssetChangeSubmissionWithoutNewsAssetFails)
	t.Run("Submitting an asset change without changes fails", TestNewAssetChangeSubmissionWithoutChangesFails)
	t.Run("Submitting an asset change without source fails", TestNewAssetChangeSubmissionWithoutSourceFails)
	t.Run("Submitting an asset change without name fails", testNewAssetChangeSubmissionWithoutNameFails)
	t.Run("Submitting an asset change with name succeeds", testNewAssetChangeSubmissionWithNameSucceeds)
	t.Run("Submitting an asset change without symbol fails", testNewAssetChangeSubmissionWithoutSymbolFails)
	t.Run("Submitting an asset change with symbol succeeds", testNewAssetChangeSubmissionWithSymbolSucceeds)
	t.Run("Submitting an asset change without decimal fails", testNewAssetChangeSubmissionWithoutDecimalsFails)
	t.Run("Submitting an asset change with decimal succeeds", testNewAssetChangeSubmissionWithDecimalsSucceeds)
	t.Run("Submitting an built-in asset change without built-in asset fails", testNewAssetChangeSubmissionWithoutBuiltInAssetFails)
	t.Run("Submitting an built-in asset change without max faucet amount fails", testNewBuiltInAssetChangeSubmissionWithoutMaxFaucetAmountMintFails)
	t.Run("Submitting an built-in asset change with max faucet amount succeeds", testNewBuiltInAssetChangeSubmissionWithMaxFaucetAmountMintSucceeds)
	t.Run("Submitting an built-in asset change with not-a-number max faucet amount fails", testNewBuiltInAssetChangeSubmissionWithNaNMaxFaucetAmountMintFails)
	t.Run("Submitting an ERC20 asset change without ERC20 asset fails", testNewERC20AssetChangeSubmissionWithoutErc20AssetFails)
	t.Run("Submitting an ERC20 asset change without contract address fails", testNewERC20AssetChangeSubmissionWithoutContractAddressFails)
	t.Run("Submitting an ERC20 asset change with contract address succeeds", testNewERC20AssetChangeSubmissionWithContractAddressSucceeds)
	t.Run("Submitting an ERC20 asset change with invalid lifetime limit fails", testNewERC20AssetChangeSubmissionWithInvalidLifetimeLimitFails)
	t.Run("Submitting an ERC20 asset change with valid lifetime limit succeeds", testNewERC20AssetChangeSubmissionWithValidLifetimeLimitSucceeds)
	t.Run("Submitting an ERC20 asset change with invalid withdrawal threshold fails", testNewERC20AssetChangeSubmissionWithInvalidWithdrawalThresholdFails)
	t.Run("Submitting an ERC20 asset change with valid withdrawal threshold succeeds", testNewERC20AssetChangeSubmissionWithValidWithdrawalThresholdSucceeds)
	t.Run("Submitting an ERC20 asset change without validation timestamp fails", testNewAssetERC20ChangeSubmissionMissingValidationTimestamp)
	t.Run("Submitting an ERC20 asset change with validation timestamp succeed", testNewAssetERC20ChangeSubmissionWithValidationTimestampSucceeds)
	t.Run("Submitting an ERC20 asset change with validation after closing timestamp fails", testNewAssetERC20ChangeSubmissionValidationAfterClosingTimestampsFails)
	t.Run("Submitting an ERC20 asset change other proposals should omit validation timestamp", testNewAssetERC20ChangeOtherProposalShouldOmitValidationTimestamp)
	t.Run("Submitting an ERC20 asset change with invalid quantum fails", testNewERC20AssetChangeSubmissionWithInvalidQuantumFails)
	t.Run("Submitting an ERC20 asset change with valid quantum succeeds", testNewERC20AssetChangeSubmissionWithValidQuantumSucceeds)
}

func testNewAssetERC20ChangeOtherProposalShouldOmitValidationTimestamp(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ValidationTimestamp: 10,
			ClosingTimestamp:    15,
			EnactmentTimestamp:  20,
			Change:              &types.ProposalTerms_NewMarket{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.validation_timestamp"), commands.ErrIsNotSupported)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   10,
			EnactmentTimestamp: 20,
			Change:             &types.ProposalTerms_NewMarket{},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.validation_timestamp"))
}

func testNewAssetERC20ChangeSubmissionWithValidationTimestampSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ValidationTimestamp: 10,
			ClosingTimestamp:    20,
			EnactmentTimestamp:  30,
			Change:              &types.ProposalTerms_NewAsset{},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.validation_timestamp"))
}

func testNewAssetERC20ChangeSubmissionValidationAfterClosingTimestampsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ValidationTimestamp: 10,
			ClosingTimestamp:    5,
			EnactmentTimestamp:  30,
			Change:              &types.ProposalTerms_NewAsset{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.validation_timestamp"), errors.New("cannot be after closing time"))
}

func testNewAssetERC20ChangeSubmissionMissingValidationTimestamp(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.validation_timestamp"), commands.ErrMustBePositive)
}

func TestNewAssetChangeSubmissionWithoutNewsAssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset"), commands.ErrIsRequired)
}

func TestNewAssetChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes"), commands.ErrIsRequired)
}

func TestNewAssetChangeSubmissionWithoutSourceFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.source"), commands.ErrIsRequired)
}

func testNewAssetChangeSubmissionWithoutNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Name: "",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.name"), commands.ErrIsRequired)
}

func testNewAssetChangeSubmissionWithNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Name: "My built-in asset",
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.name"))
}

func testNewAssetChangeSubmissionWithoutSymbolFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Symbol: "",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.symbol"), commands.ErrIsRequired)
}

func testNewAssetChangeSubmissionWithSymbolSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Symbol: "My symbol",
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.symbol"), commands.ErrIsRequired)
}

func testNewAssetChangeSubmissionWithoutDecimalsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Decimals: 0,
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_asset.changes.decimals"), commands.ErrIsRequired)
}

func testNewAssetChangeSubmissionWithDecimalsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Decimals: RandomPositiveU64(),
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.decimals"))
}

func testNewAssetChangeSubmissionWithoutBuiltInAssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_BuiltinAsset{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.builtin_asset"), commands.ErrIsRequired)
}

func testNewBuiltInAssetChangeSubmissionWithoutMaxFaucetAmountMintFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{
								MaxFaucetAmountMint: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint"), commands.ErrIsRequired)
}

func testNewBuiltInAssetChangeSubmissionWithMaxFaucetAmountMintSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_BuiltinAsset{
							BuiltinAsset: &types.BuiltinAsset{
								MaxFaucetAmountMint: "10000",
							},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint"))
}

func testNewBuiltInAssetChangeSubmissionWithNaNMaxFaucetAmountMintFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value string
		error error
	}{
		{
			msg:   "with not-a-number value",
			value: "hello",
			error: commands.ErrIsNotValidNumber,
		}, {
			msg:   "with value of 0",
			value: "0",
			error: commands.ErrMustBePositive,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{
						NewAsset: &types.NewAsset{
							Changes: &types.AssetDetails{
								Source: &types.AssetDetails_BuiltinAsset{
									BuiltinAsset: &types.BuiltinAsset{
										MaxFaucetAmountMint: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint"), tc.error)
		})
	}
}

func testNewERC20AssetChangeSubmissionWithoutErc20AssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_Erc20{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.erc20"), commands.ErrIsRequired)
}

func testNewERC20AssetChangeSubmissionWithoutContractAddressFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_Erc20{
							Erc20: &types.ERC20{
								ContractAddress: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.erc20.contract_address"), commands.ErrIsRequired)
}

func testNewERC20AssetChangeSubmissionWithContractAddressSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_Erc20{
							Erc20: &types.ERC20{
								ContractAddress: "My address",
							},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.erc20.contract_address"))
}

func testNewERC20AssetChangeSubmissionWithInvalidLifetimeLimitFails(t *testing.T) {
	tcs := []struct {
		name  string
		err   error
		value string
	}{
		{
			name:  "Without lifetime limit",
			value: "",
			err:   commands.ErrIsRequired,
		}, {
			name:  "With not-a-number lifetime limit",
			value: "forty-two",
			err:   commands.ErrIsNotValidNumber,
		}, {
			name:  "With zero lifetime limit",
			value: "0",
			err:   commands.ErrMustBePositive,
		}, {
			name:  "With negative lifetime limit",
			value: "-10",
			err:   commands.ErrMustBePositive,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{
						NewAsset: &types.NewAsset{
							Changes: &types.AssetDetails{
								Source: &types.AssetDetails_Erc20{
									Erc20: &types.ERC20{
										LifetimeLimit: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(tt, err.Get("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit"), tc.err)
		})
	}
}

func testNewERC20AssetChangeSubmissionWithValidLifetimeLimitSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_Erc20{
							Erc20: &types.ERC20{
								LifetimeLimit: "100",
							},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit"))
}

func testNewERC20AssetChangeSubmissionWithInvalidWithdrawalThresholdFails(t *testing.T) {
	tcs := []struct {
		name  string
		err   error
		value string
	}{
		{
			name:  "Without withdraw threshold",
			value: "",
			err:   commands.ErrIsRequired,
		}, {
			name:  "With not-a-number withdraw threshold",
			value: "forty-two",
			err:   commands.ErrIsNotValidNumber,
		}, {
			name:  "With zero withdraw threshold",
			value: "0",
			err:   commands.ErrMustBePositive,
		}, {
			name:  "With negative withdraw threshold",
			value: "-10",
			err:   commands.ErrMustBePositive,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{
						NewAsset: &types.NewAsset{
							Changes: &types.AssetDetails{
								Source: &types.AssetDetails_Erc20{
									Erc20: &types.ERC20{
										WithdrawThreshold: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(tt, err.Get("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold"), tc.err)
		})
	}
}

func testNewERC20AssetChangeSubmissionWithValidWithdrawalThresholdSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetails_Erc20{
							Erc20: &types.ERC20{
								WithdrawThreshold: "100",
							},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold"))
}

func testNewERC20AssetChangeSubmissionWithInvalidQuantumFails(t *testing.T) {
	tcs := []struct {
		name  string
		err   error
		value string
	}{
		{
			name:  "Without withdraw quantum",
			value: "",
			err:   commands.ErrIsRequired,
		}, {
			name:  "With not-a-number quantum",
			value: "forty-two",
			err:   commands.ErrIsNotValidNumber,
		}, {
			name:  "With zero withdraw quantum",
			value: "0",
			err:   commands.ErrMustBePositive,
		}, {
			name:  "With negative withdraw quantum",
			value: "-10",
			err:   commands.ErrMustBePositive,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{
						NewAsset: &types.NewAsset{
							Changes: &types.AssetDetails{
								Quantum: tc.value,
								Source: &types.AssetDetails_Erc20{
									Erc20: &types.ERC20{},
								},
							},
						},
					},
				},
			})

			assert.Contains(tt, err.Get("proposal_submission.terms.change.new_asset.changes.quantum"), tc.err)
		})
	}
}

func testNewERC20AssetChangeSubmissionWithValidQuantumSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Quantum: "0.1",
						Source: &types.AssetDetails_Erc20{
							Erc20: &types.ERC20{},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.new_asset.changes.quantum"))
}
