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
	"time"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestVolumeRebateSubmission(t *testing.T) {
	t.Run("empty submission", testUpdateRebateProgram)
	t.Run("0095-HVMR-001: invalid end timestamp", testInvalidEndTime)
	t.Run("0095-HVMR-003: tier validation", testInvalidTiers)
	t.Run("0095-HVMR-004: invalid window length", testInvalidWindowLength)
}

func testUpdateRebateProgram(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeRebateProgram{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_rebate_program"), commands.ErrIsRequired)
	// missing changes, same problem
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateVolumeRebateProgram{
				UpdateVolumeRebateProgram: &types.UpdateVolumeRebateProgram{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_rebate_program.changes"), commands.ErrIsRequired)
}

// testInvalidEndTime covers 0095-HVMR-001.
func testInvalidEndTime(t *testing.T) {
	end := time.Now()
	enact := end.Add(time.Second)
	prop := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			EnactmentTimestamp: enact.Unix(),
			Change: &types.ProposalTerms_UpdateVolumeRebateProgram{
				UpdateVolumeRebateProgram: &types.UpdateVolumeRebateProgram{
					Changes: &types.VolumeRebateProgramChanges{
						EndOfProgramTimestamp: end.Unix(),
					},
				},
			},
		},
	}
	err := checkProposalSubmission(prop)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_rebate_program.changes.end_of_program_timestamp"), commands.ErrMustBeGreaterThanEnactmentTimestamp)
}

// testInvalidWindowLength covers 0095-HVMR-004.
func testInvalidWindowLength(t *testing.T) {
	enact := time.Now()
	end := enact.Add(time.Second)
	prop := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			EnactmentTimestamp: enact.Unix(),
			Change: &types.ProposalTerms_UpdateVolumeRebateProgram{
				UpdateVolumeRebateProgram: &types.UpdateVolumeRebateProgram{
					Changes: &types.VolumeRebateProgramChanges{
						EndOfProgramTimestamp: end.Unix(),
						WindowLength:          0, // zero is invalid
					},
				},
			},
		},
	}
	err := checkProposalSubmission(prop)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_rebate_program.changes.window_length"), commands.ErrIsRequired)
	// now too high of a value
	prop = &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			EnactmentTimestamp: enact.Unix(),
			Change: &types.ProposalTerms_UpdateVolumeRebateProgram{
				UpdateVolumeRebateProgram: &types.UpdateVolumeRebateProgram{
					Changes: &types.VolumeRebateProgramChanges{
						EndOfProgramTimestamp: end.Unix(),
						WindowLength:          10000,
					},
				},
			},
		},
	}
	err = checkProposalSubmission(prop)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_volume_rebate_program.changes.window_length"), commands.ErrMustBeAtMost200)
}

// testInvalidTiers covers 0095-HVMR-003.
func testInvalidTiers(t *testing.T) {
	errMap := map[string]error{
		"proposal_submission.terms.change.update_volume_rebate_program.changes.benefit_tiers.0.minimum_party_maker_volume_fraction": commands.ErrIsRequired,
		"proposal_submission.terms.change.update_volume_rebate_program.changes.benefit_tiers.1.minimum_party_maker_volume_fraction": commands.ErrIsNotValidNumber,
		"proposal_submission.terms.change.update_volume_rebate_program.changes.benefit_tiers.2.minimum_party_maker_volume_fraction": commands.ErrMustBePositive,
		"proposal_submission.terms.change.update_volume_rebate_program.changes.benefit_tiers.3.minimum_party_maker_volume_fraction": commands.ErrMustBePositive,
		"proposal_submission.terms.change.update_volume_rebate_program.changes.benefit_tiers.4.additional_maker_rebate":             commands.ErrIsRequired,
		"proposal_submission.terms.change.update_volume_rebate_program.changes.benefit_tiers.5.additional_maker_rebate":             commands.ErrIsNotValidNumber,
		"proposal_submission.terms.change.update_volume_rebate_program.changes.benefit_tiers.6.additional_maker_rebate":             commands.ErrMustBePositiveOrZero,
	}
	enact := time.Now()
	end := enact.Add(time.Second)
	prop := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			EnactmentTimestamp: enact.Unix(),
			Change: &types.ProposalTerms_UpdateVolumeRebateProgram{
				UpdateVolumeRebateProgram: &types.UpdateVolumeRebateProgram{
					Changes: &types.VolumeRebateProgramChanges{
						EndOfProgramTimestamp: end.Unix(),
						WindowLength:          10,
						BenefitTiers: []*types.VolumeRebateBenefitTier{
							{
								MinimumPartyMakerVolumeFraction: "",
								AdditionalMakerRebate:           "0.1",
							},
							{
								MinimumPartyMakerVolumeFraction: "invalid",
								AdditionalMakerRebate:           "0.1",
							},
							{
								MinimumPartyMakerVolumeFraction: "-1",
								AdditionalMakerRebate:           "0.1",
							},
							{
								MinimumPartyMakerVolumeFraction: "0",
								AdditionalMakerRebate:           "0.1",
							},
							{
								MinimumPartyMakerVolumeFraction: "0.1",
								AdditionalMakerRebate:           "",
							},
							{
								MinimumPartyMakerVolumeFraction: "0.1",
								AdditionalMakerRebate:           "invalid",
							},
							{
								MinimumPartyMakerVolumeFraction: "0.1",
								AdditionalMakerRebate:           "-0.1",
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(prop)
	for g, c := range errMap {
		assert.Contains(t, err.Get(g), c, err.Error())
	}
}
