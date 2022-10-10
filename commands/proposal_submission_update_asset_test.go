package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForUpdateAsset(t *testing.T) {
	t.Run("Submitting an asset update without new asset fails", TestUpdateAssetSubmissionWithoutUpdateAssetFails)
	t.Run("Submitting an asset update without asset ID fails", TestUpdateAssetSubmissionWithoutAssetIDFails)
	t.Run("Submitting an asset update with asset ID succeeds", TestUpdateAssetSubmissionWithAssetIDSucceeds)
	t.Run("Submitting an asset update without changes fails", TestUpdateAssetSubmissionWithoutChangesFails)
	t.Run("Submitting an asset update without source fails", TestUpdateAssetSubmissionWithoutSourceFails)
	t.Run("Submitting an ERC20 asset update without ERC20 asset fails", testUpdateERC20AssetChangeSubmissionWithoutErc20AssetFails)
	t.Run("Submitting an ERC20 asset update with invalid lifetime limit fails", testUpdateERC20AssetChangeSubmissionWithInvalidLifetimeLimitFails)
	t.Run("Submitting an ERC20 asset update with valid lifetime limit succeeds", testUpdateERC20AssetChangeSubmissionWithValidLifetimeLimitSucceeds)
	t.Run("Submitting an ERC20 asset update with invalid withdrawal threshold fails", testUpdateERC20AssetChangeSubmissionWithInvalidWithdrawalThresholdFails)
	t.Run("Submitting an ERC20 asset update with valid withdrawal threshold succeeds", testUpdateERC20AssetChangeSubmissionWithValidWithdrawalThresholdSucceeds)
	t.Run("Submitting an ERC20 asset change with invalid quantum fails", testUpdateERC20AssetChangeSubmissionWithInvalidQuantumFails)
	t.Run("Submitting an ERC20 asset change with valid quantum succeeds", testUpdateERC20AssetChangeSubmissionWithValidQuantumSucceeds)
}

func TestUpdateAssetSubmissionWithoutUpdateAssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_asset"), commands.ErrIsRequired)
}

func TestUpdateAssetSubmissionWithAssetIDSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{
					AssetId: "",
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_asset.asset_id"), commands.ErrIsRequired)
}

func TestUpdateAssetSubmissionWithoutAssetIDFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{
					AssetId: "invalid",
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_asset.asset_id"), commands.ErrShouldBeAValidVegaID)
}

func TestUpdateAssetSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_asset.changes"), commands.ErrIsRequired)
}

func TestUpdateAssetSubmissionWithoutSourceFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{
					Changes: &types.AssetDetailsUpdate{},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_asset.changes.source"), commands.ErrIsRequired)
}

func testUpdateERC20AssetChangeSubmissionWithoutErc20AssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{
					Changes: &types.AssetDetailsUpdate{
						Source: &types.AssetDetailsUpdate_Erc20{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_asset.changes.source.erc20"), commands.ErrIsRequired)
}

func testUpdateERC20AssetChangeSubmissionWithInvalidLifetimeLimitFails(t *testing.T) {
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
					Change: &types.ProposalTerms_UpdateAsset{
						UpdateAsset: &types.UpdateAsset{
							Changes: &types.AssetDetailsUpdate{
								Source: &types.AssetDetailsUpdate_Erc20{
									Erc20: &types.ERC20Update{
										LifetimeLimit: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(tt, err.Get("proposal_submission.terms.change.update_asset.changes.source.erc20.lifetime_limit"), tc.err)
		})
	}
}

func testUpdateERC20AssetChangeSubmissionWithValidLifetimeLimitSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{
					Changes: &types.AssetDetailsUpdate{
						Source: &types.AssetDetailsUpdate_Erc20{
							Erc20: &types.ERC20Update{
								LifetimeLimit: "100",
							},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.update_asset.changes.source.erc20.lifetime_limit"))
}

func testUpdateERC20AssetChangeSubmissionWithInvalidWithdrawalThresholdFails(t *testing.T) {
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
					Change: &types.ProposalTerms_UpdateAsset{
						UpdateAsset: &types.UpdateAsset{
							Changes: &types.AssetDetailsUpdate{
								Source: &types.AssetDetailsUpdate_Erc20{
									Erc20: &types.ERC20Update{
										WithdrawThreshold: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(tt, err.Get("proposal_submission.terms.change.update_asset.changes.source.erc20.withdraw_threshold"), tc.err)
		})
	}
}

func testUpdateERC20AssetChangeSubmissionWithValidWithdrawalThresholdSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{
					Changes: &types.AssetDetailsUpdate{
						Source: &types.AssetDetailsUpdate_Erc20{
							Erc20: &types.ERC20Update{
								WithdrawThreshold: "100",
							},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.update_asset.changes.source.erc20.withdraw_threshold"))
}

func testUpdateERC20AssetChangeSubmissionWithInvalidQuantumFails(t *testing.T) {
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
					Change: &types.ProposalTerms_UpdateAsset{
						UpdateAsset: &types.UpdateAsset{
							Changes: &types.AssetDetailsUpdate{
								Quantum: tc.value,
								Source: &types.AssetDetailsUpdate_Erc20{
									Erc20: &types.ERC20Update{},
								},
							},
						},
					},
				},
			})

			assert.Contains(tt, err.Get("proposal_submission.terms.change.update_asset.changes.quantum"), tc.err)
		})
	}
}

func testUpdateERC20AssetChangeSubmissionWithValidQuantumSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateAsset{
				UpdateAsset: &types.UpdateAsset{
					Changes: &types.AssetDetailsUpdate{
						Quantum: "0.1",
						Source: &types.AssetDetailsUpdate_Erc20{
							Erc20: &types.ERC20Update{},
						},
					},
				},
			},
		},
	})

	assert.Empty(t, err.Get("proposal_submission.terms.change.update_asset.changes.quantum"))
}
