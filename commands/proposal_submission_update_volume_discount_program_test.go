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
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForVolumeDiscountProgramUpdate(t *testing.T) {
	t.Run("Submitting a volume discount program update without update fails", testSubmissionForVolumeDiscountProgramUpdateWithoutUpdateFails)
	t.Run("Submitting a volume discount program update without changes fails", testSubmissionForVolumeDiscountProgramUpdateWithoutChangesFails)
	t.Run("Submitting a volume discount program update without end of program timestamp fails", testSubmissionForVolumeDiscountProgramUpdateWithoutEndOfProgramFails)
	t.Run("Submitting a volume discount program update with negative end of program timestamp fails", testSubmissionForVolumeDiscountProgramUpdateWithNegativeEndOfProgramFails)
	t.Run("Submitting a volume discount program update with end of program before enactment timestamp fails", testSubmissionForVolumeDiscountProgramUpdateWithEndOfProgramBeforeEnactmentFails)
	t.Run("Submitting a volume discount program update without window length fails", testSubmissionForVolumeDiscountProgramUpdateWithoutWindowLengthFails)
	t.Run("Submitting a volume discount program update with window length over limit fails", testSubmissionForVolumeDiscountProgramUpdateWithWindowLengthOverLimitFails)
	t.Run("Submitting a volume discount program update without tier minimum running volume fails", testSubmissionForVolumeDiscountProgramUpdateWithoutTierMinimumRunningNotionalTakerVolumeFails)
	t.Run("Submitting a volume discount program update with bad format for tier minimum running volume fails", testSubmissionForVolumeDiscountProgramUpdateWithBadFormatForTierMinimumRunningNotionalTakerVolumeFails)
	t.Run("Submitting a volume discount program update with bad value for tier minimum running volume fails", testSubmissionForVolumeDiscountProgramUpdateWithBadValueForTierMinimumRunningNotionalTakerVolumeFails)
	t.Run("Submitting a volume discount program update without tier volume discount factor fails", testSubmissionForVolumeDiscountProgramUpdateWithoutTierVolumeDiscountFactorFails)
	t.Run("Submitting a volume discount program update with bad format for tier volume discount factor fails", testSubmissionForVolumeDiscountProgramUpdateWithBadFormatForTierVolumeDiscountFactorFails)
	t.Run("Submitting a volume discount program update with bad value for tier volume discount factor fails", testSubmissionForVolumeDiscountProgramUpdateWithBadValueForTierVolumeDiscountFactorFails)
}

func testSubmissionForVolumeDiscountProgramUpdateWithoutUpdateFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program"), commands.ErrIsRequired)
}

func testSubmissionForVolumeDiscountProgramUpdateWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes"), commands.ErrIsRequired)
}

func testSubmissionForVolumeDiscountProgramUpdateWithoutEndOfProgramFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						EndOfProgramTimestamp: 0,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.end_of_program_timestamp"), commands.ErrIsRequired)
}

func testSubmissionForVolumeDiscountProgramUpdateWithNegativeEndOfProgramFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						EndOfProgramTimestamp: -1,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.end_of_program_timestamp"), commands.ErrMustBePositive)
}

func testSubmissionForVolumeDiscountProgramUpdateWithEndOfProgramBeforeEnactmentFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			EnactmentTimestamp: 10,
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						EndOfProgramTimestamp: 5,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.end_of_program_timestamp"), commands.ErrMustBeGreaterThanEnactmentTimestamp)
}

func testSubmissionForVolumeDiscountProgramUpdateWithoutWindowLengthFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						WindowLength: 0,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.window_length"), commands.ErrIsRequired)
}

func testSubmissionForVolumeDiscountProgramUpdateWithWindowLengthOverLimitFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						WindowLength: 101,
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.window_length"), commands.ErrMustBeAtMost100)
}

func testSubmissionForVolumeDiscountProgramUpdateWithoutTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						BenefitTiers: []*types.VolumeBenefitTier{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.minimum_running_notional_taker_volume"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.1.minimum_running_notional_taker_volume"), commands.ErrIsRequired)
}

func testSubmissionForVolumeDiscountProgramUpdateWithBadFormatForTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						BenefitTiers: []*types.VolumeBenefitTier{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.minimum_running_notional_taker_volume"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.1.minimum_running_notional_taker_volume"), commands.ErrIsNotValidNumber)
}

func testSubmissionForVolumeDiscountProgramUpdateWithBadValueForTierMinimumRunningNotionalTakerVolumeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						BenefitTiers: []*types.VolumeBenefitTier{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.minimum_running_notional_taker_volume"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.1.minimum_running_notional_taker_volume"), commands.ErrMustBePositive)
}

func testSubmissionForVolumeDiscountProgramUpdateWithoutTierVolumeDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						BenefitTiers: []*types.VolumeBenefitTier{
							{}, {},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.1.volume_discount_factors"), commands.ErrIsRequired)
}

func testSubmissionForVolumeDiscountProgramUpdateWithBadFormatForTierVolumeDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						BenefitTiers: []*types.VolumeBenefitTier{
							{
								VolumeDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: "qbc",
									LiquidityDiscountFactor:      "qbc",
									MakerDiscountFactor:          "qbc",
								},
							}, {
								VolumeDiscountFactors: &types.DiscountFactors{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.maker_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.liquidity_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.infrastructure_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.1.volume_discount_factors.maker_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.1.volume_discount_factors.liquidity_discount_factor"), commands.ErrIsNotValidNumber)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.1.volume_discount_factors.infrastructure_discount_factor"), commands.ErrIsNotValidNumber)
}

func testSubmissionForVolumeDiscountProgramUpdateWithBadValueForTierVolumeDiscountFactorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeDiscountProgram{
				UpdateVolumeDiscountProgram: &types.UpdateVolumeDiscountProgram{
					Changes: &types.VolumeDiscountProgramChanges{
						BenefitTiers: []*types.VolumeBenefitTier{
							{
								VolumeDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: "-10",
									LiquidityDiscountFactor:      "-5",
									MakerDiscountFactor:          "-7",
								},
							}, {
								VolumeDiscountFactors: &types.DiscountFactors{
									InfrastructureDiscountFactor: "-1",
									LiquidityDiscountFactor:      "-3",
									MakerDiscountFactor:          "-9",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.maker_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.liquidity_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.infrastructure_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.maker_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.liquidity_discount_factor"), commands.ErrMustBePositiveOrZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_discount_program.changes.benefit_tiers.0.volume_discount_factors.infrastructure_discount_factor"), commands.ErrMustBePositiveOrZero)
}
