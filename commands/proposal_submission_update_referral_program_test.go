package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForReferralProgramUpdate(t *testing.T) {
	t.Run("Submitting a referral program update without update fails", testSubmissionForReferralProgramUpdateWithoutUpdateFails)
	t.Run("Submitting a referral program update without changes fails", testSubmissionForReferralProgramUpdateWithoutChangesFails)
	t.Run("Submitting a referral program update without end of program timestamp fails", testSubmissionForReferralProgramUpdateWithoutEndOfProgramFails)
	t.Run("Submitting a referral program update with negative end of program timestamp fails", testSubmissionForReferralProgramUpdateWithNegativeEndOfProgramFails)
	t.Run("Submitting a referral program update with end of program before enactment timestamp fails", testSubmissionForReferralProgramUpdateWithEndOfProgramBeforeEnactmentFails)
	t.Run("Submitting a referral program update without window length fails", testSubmissionForReferralProgramUpdateWithoutWindowLengthFails)
	t.Run("Submitting a referral program update without tier minimum running volume fails", testSubmissionForReferralProgramUpdateWithoutTierMinimumRunningNotionalTakerVolumeFails)
	t.Run("Submitting a referral program update with bad format for tier minimum running volume fails", testSubmissionForReferralProgramUpdateWithBadFormatForTierMinimumRunningNotionalTakerVolumeFails)
	t.Run("Submitting a referral program update with bad value for tier minimum running volume fails", testSubmissionForReferralProgramUpdateWithBadValueForTierMinimumRunningNotionalTakerVolumeFails)
	t.Run("Submitting a referral program update without tier minimum epochs fails", testSubmissionForReferralProgramUpdateWithoutTierMinimumEpochsFails)
	t.Run("Submitting a referral program update with bad format for tier minimum epochs fails", testSubmissionForReferralProgramUpdateWithBadFormatForTierMinimumEpochsFails)
	t.Run("Submitting a referral program update with bad value for tier minimum epochs fails", testSubmissionForReferralProgramUpdateWithBadValueForTierMinimumEpochsFails)
	t.Run("Submitting a referral program update without tier referral reward factor fails", testSubmissionForReferralProgramUpdateWithoutTierReferralRewardFactorFails)
	t.Run("Submitting a referral program update with bad format for tier referral reward factor fails", testSubmissionForReferralProgramUpdateWithBadFormatForTierReferralRewardFactorFails)
	t.Run("Submitting a referral program update with bad value for tier referral reward factor fails", testSubmissionForReferralProgramUpdateWithBadValueForTierReferralRewardFactorFails)
	t.Run("Submitting a referral program update without tier referral discount factor fails", testSubmissionForReferralProgramUpdateWithoutTierReferralDiscountFactorFails)
	t.Run("Submitting a referral program update with bad format for tier referral discount factor fails", testSubmissionForReferralProgramUpdateWithBadFormatForTierReferralDiscountFactorFails)
	t.Run("Submitting a referral program update with bad value for tier referral discount factor fails", testSubmissionForReferralProgramUpdateWithBadValueForTierReferralDiscountFactorFails)
}

func testSubmissionForReferralProgramUpdateWithoutUpdateFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithoutEndOfProgramFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						EndOfProgramTimestamp: 0,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.end_of_program_timestamp"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithNegativeEndOfProgramFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						EndOfProgramTimestamp: -1,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.end_of_program_timestamp"), commands.ErrMustBePositive)
}

func testSubmissionForReferralProgramUpdateWithEndOfProgramBeforeEnactmentFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			EnactmentTimestamp: 10,
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						EndOfProgramTimestamp: 5,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.end_of_program_timestamp"), commands.ErrMustBeGreaterThanEnactmentTimestamp)
}

func testSubmissionForReferralProgramUpdateWithoutWindowLengthFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						WindowLength: 0,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.end_of_program_timestamp"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithoutTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								MinimumRunningNotionalTakerVolume: "",
							}, {
								MinimumRunningNotionalTakerVolume: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.minimum_running_notional_taker_volume"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.minimum_running_notional_taker_volume"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								MinimumRunningNotionalTakerVolume: "qbc",
							}, {
								MinimumRunningNotionalTakerVolume: "0x32",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.minimum_running_notional_taker_volume"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.minimum_running_notional_taker_volume"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								MinimumRunningNotionalTakerVolume: "0",
							}, {
								MinimumRunningNotionalTakerVolume: "-1",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.minimum_running_notional_taker_volume"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.minimum_running_notional_taker_volume"), commands.ErrMustBePositive)
}

func testSubmissionForReferralProgramUpdateWithoutTierMinimumEpochsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								MinimumEpochs: "",
							}, {
								MinimumEpochs: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.minimum_epochs"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.minimum_epochs"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForTierMinimumEpochsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								MinimumEpochs: "qbc",
							}, {
								MinimumEpochs: "0x32",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.minimum_epochs"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.minimum_epochs"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForTierMinimumEpochsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								MinimumEpochs: "0",
							}, {
								MinimumEpochs: "-1",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.minimum_epochs"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.minimum_epochs"), commands.ErrMustBePositive)
}

func testSubmissionForReferralProgramUpdateWithoutTierReferralRewardFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralRewardFactor: "",
							}, {
								ReferralRewardFactor: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factor"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factor"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForTierReferralRewardFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralRewardFactor: "qbc",
							}, {
								ReferralRewardFactor: "0x32",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factor"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForTierReferralRewardFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralRewardFactor: "-10",
							}, {
								ReferralRewardFactor: "-1",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factor"), commands.ErrMustBePositiveOrZero)
}

func testSubmissionForReferralProgramUpdateWithoutTierReferralDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralDiscountFactor: "",
							}, {
								ReferralDiscountFactor: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factor"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factor"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForTierReferralDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralDiscountFactor: "qbc",
							}, {
								ReferralDiscountFactor: "0x32",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factor"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForTierReferralDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgram{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralDiscountFactor: "-10",
							}, {
								ReferralDiscountFactor: "-1",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factor"), commands.ErrMustBePositiveOrZero)
}
