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
	"fmt"
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
	t.Run("Submitting a referral program update with window length over limit fails", testSubmissionForReferralProgramUpdateWithWindowLengthOverLimitFails)
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
	t.Run("Submitting a referral program update without staking tier minimum staked tokens fails", testSubmissionForReferralProgramUpdateWithoutStakingTierMinimumStakedTokensFails)
	t.Run("Submitting a referral program update with bad format for staking tier minimum staked tokens fails", testSubmissionForReferralProgramUpdateWithBadFormatForStakingTierMinimumStakedTokensFails)
	t.Run("Submitting a referral program update with bad value for staking tier minimum staked tokens fails", testSubmissionForReferralProgramUpdateWithBadValueForStakingTierMinimumStakedTokensFails)
	t.Run("Submitting a referral program update without staking tier referral reward multiplier fails", testSubmissionForReferralProgramUpdateWithoutStakingTierReferralRewardMultiplierFails)
	t.Run("Submitting a referral program update with bad format for staking tier referral reward multiplier fails", testSubmissionForReferralProgramUpdateWithBadFormatForStakingTierReferralRewardMultiplierFails)
	t.Run("Submitting a referral program update with bad value for staking tier referral reward multiplier fails", testSubmissionForReferralProgramUpdateWithBadValueForStakingTierReferralRewardMultiplierFails)
	t.Run("Submitting a referral program update with multiple identical staking tiers fails", testSubmissionForReferralProgramUpdateWithDuplicateStakingTierEntriesFails)
	t.Run("Submitting a referral program update with multiple identical benefit tiers fails", testSubmissionForReferralProgramUpdateWithDuplicateBenefitTierEntriesFails)
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
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
						WindowLength: 0,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.window_length"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithWindowLengthOverLimitFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						WindowLength: 101,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.window_length"), commands.ErrMustBeAtMost100)
}

func testSubmissionForReferralProgramUpdateWithoutTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
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

func testSubmissionForReferralProgramUpdateWithDuplicateBenefitTierEntriesFails(t *testing.T) {
	factors := []string{"1.1", "1.2", "1.3", "1.4", "1.5", "1.6"}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						BenefitTiers: []*types.BenefitTier{
							{
								MinimumRunningNotionalTakerVolume: "100",
								MinimumEpochs:                     "10",
								ReferralRewardFactors: &types.RewardFactors{
									InfrastructureRewardFactor: factors[0],
									MakerRewardFactor:          factors[1],
									LiquidityRewardFactor:      factors[2],
								},
								ReferralDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: factors[1],
									MakerDiscountFactor:          factors[2],
									LiquidityDiscountFactor:      factors[3],
								},
							},
							{
								MinimumRunningNotionalTakerVolume: "100",
								MinimumEpochs:                     "10",
								ReferralRewardFactors: &types.RewardFactors{
									InfrastructureRewardFactor: factors[1],
									MakerRewardFactor:          factors[2],
									LiquidityRewardFactor:      factors[3],
								},
								ReferralDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: factors[2],
									MakerDiscountFactor:          factors[3],
									LiquidityDiscountFactor:      factors[4],
								},
							},
							{
								MinimumRunningNotionalTakerVolume: "100",
								MinimumEpochs:                     "20",
								ReferralRewardFactors: &types.RewardFactors{
									InfrastructureRewardFactor: factors[2],
									MakerRewardFactor:          factors[3],
									LiquidityRewardFactor:      factors[4],
								},
								ReferralDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: factors[3],
									MakerDiscountFactor:          factors[4],
									LiquidityDiscountFactor:      factors[5],
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1"), fmt.Errorf("duplicate benefit tier"))
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.2"), fmt.Errorf("duplicate benefit tier"))
}

func testSubmissionForReferralProgramUpdateWithDuplicateStakingTierEntriesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						StakingTiers: []*types.StakingTier{
							{
								MinimumStakedTokens:      "100",
								ReferralRewardMultiplier: "1.2",
							}, {
								MinimumStakedTokens:      "100",
								ReferralRewardMultiplier: "1.3",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.1"), fmt.Errorf("duplicate staking tier"))
}

func testSubmissionForReferralProgramUpdateWithBadFormatForTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
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
					Changes: &types.ReferralProgramChanges{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factors"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factors"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForTierReferralRewardFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralRewardFactors: &types.RewardFactors{
									InfrastructureRewardFactor: "qbc",
									MakerRewardFactor:          "qbc",
									LiquidityRewardFactor:      "qbc",
								},
							}, {
								ReferralRewardFactors: &types.RewardFactors{
									InfrastructureRewardFactor: "0x32",
									MakerRewardFactor:          "0x32",
									LiquidityRewardFactor:      "0x32",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factors.infrastructure_reward_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factors.infrastructure_reward_factor"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForTierReferralRewardFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralRewardFactors: &types.RewardFactors{
									InfrastructureRewardFactor: "-10",
									MakerRewardFactor:          "-10",
									LiquidityRewardFactor:      "-10",
								},
							}, {
								ReferralRewardFactors: &types.RewardFactors{
									InfrastructureRewardFactor: "-1",
									MakerRewardFactor:          "-1",
									LiquidityRewardFactor:      "-1",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factors.infrastructure_reward_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factors.infrastructure_reward_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factors.maker_reward_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factors.maker_reward_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_reward_factors.liquidity_reward_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_reward_factors.liquidity_reward_factor"), commands.ErrMustBePositiveOrZero)
}

func testSubmissionForReferralProgramUpdateWithoutTierReferralDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralDiscountFactors: &types.DiscountFactors{},
							}, {
								ReferralDiscountFactors: &types.DiscountFactors{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.infrastructure_discount_factor"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.infrastructure_discount_factor"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.maker_discount_factor"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.maker_discount_factor"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.liquidity_discount_factor"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.liquidity_discount_factor"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForTierReferralDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: "qbc",
									LiquidityDiscountFactor:      "qbc",
									MakerDiscountFactor:          "qbc",
								},
							}, {
								ReferralDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: "0x32",
									LiquidityDiscountFactor:      "0x32",
									MakerDiscountFactor:          "0x32",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.infrastructure_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.infrastructure_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.maker_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.maker_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.liquidity_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.liquidity_discount_factor"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForTierReferralDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						BenefitTiers: []*types.BenefitTier{
							{
								ReferralDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: "-10",
									MakerDiscountFactor:          "-10",
									LiquidityDiscountFactor:      "-10",
								},
							}, {
								ReferralDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: "-1",
									MakerDiscountFactor:          "-1",
									LiquidityDiscountFactor:      "-1",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.infrastructure_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.infrastructure_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.liquidity_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.liquidity_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.0.referral_discount_factors.maker_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.benefit_tiers.1.referral_discount_factors.maker_discount_factor"), commands.ErrMustBePositiveOrZero)
}

func testSubmissionForReferralProgramUpdateWithoutStakingTierMinimumStakedTokensFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						StakingTiers: []*types.StakingTier{
							{
								MinimumStakedTokens: "",
							}, {
								MinimumStakedTokens: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.0.minimum_staked_tokens"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.1.minimum_staked_tokens"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForStakingTierMinimumStakedTokensFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						StakingTiers: []*types.StakingTier{
							{
								MinimumStakedTokens: "qbc",
							}, {
								MinimumStakedTokens: "0x32",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.0.minimum_staked_tokens"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.1.minimum_staked_tokens"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForStakingTierMinimumStakedTokensFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						StakingTiers: []*types.StakingTier{
							{
								MinimumStakedTokens: "-100",
							}, {
								MinimumStakedTokens: "-1",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.0.minimum_staked_tokens"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.1.minimum_staked_tokens"), commands.ErrMustBePositive)
}

func testSubmissionForReferralProgramUpdateWithoutStakingTierReferralRewardMultiplierFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						StakingTiers: []*types.StakingTier{
							{
								ReferralRewardMultiplier: "",
							}, {
								ReferralRewardMultiplier: "",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.0.referral_reward_multiplier"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.1.referral_reward_multiplier"), commands.ErrIsRequired)
}

func testSubmissionForReferralProgramUpdateWithBadFormatForStakingTierReferralRewardMultiplierFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						StakingTiers: []*types.StakingTier{
							{
								ReferralRewardMultiplier: "qbc",
							}, {
								ReferralRewardMultiplier: "0x32",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.0.referral_reward_multiplier"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.1.referral_reward_multiplier"), commands.ErrIsNotValidNumber)
}

func testSubmissionForReferralProgramUpdateWithBadValueForStakingTierReferralRewardMultiplierFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateReferralProgram{
				UpdateReferralProgram: &types.UpdateReferralProgram{
					Changes: &types.ReferralProgramChanges{
						StakingTiers: []*types.StakingTier{
							{
								ReferralRewardMultiplier: "-0.1",
							}, {
								ReferralRewardMultiplier: "0",
							}, {
								ReferralRewardMultiplier: "0.9",
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.0.referral_reward_multiplier"), commands.ErrMustBeGTE1)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.1.referral_reward_multiplier"), commands.ErrMustBeGTE1)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_referral_program.changes.staking_tiers.2.referral_reward_multiplier"), commands.ErrMustBeGTE1)
}
