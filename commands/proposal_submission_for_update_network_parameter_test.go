// Copyright (C) 2023  Gobalsky Labs Limited
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

func TestCheckProposalSubmissionForNetworkParameterUpdate(t *testing.T) {
	t.Run("Submitting a network parameter changes without network parameter fails", testNetworkParameterChangeSubmissionWithoutNetworkParameterFails)
	t.Run("Submitting a network parameter changes without changes fails", testNetworkParameterChangeSubmissionWithoutChangesFails)
	t.Run("Submitting a network parameter change without key fails", testNetworkParameterChangeSubmissionWithoutKeyFails)
	t.Run("Submitting a network parameter change with key succeeds", testNetworkParameterChangeSubmissionWithKeySucceeds)
	t.Run("Submitting a network parameter change without value fails", testNetworkParameterChangeSubmissionWithoutValueFails)
	t.Run("Submitting a network parameter change with value succeeds", testNetworkParameterChangeSubmissionWithValueSucceeds)
}

func testNetworkParameterChangeSubmissionWithoutNetworkParameterFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateNetworkParameter{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_network_parameter"), commands.ErrIsRequired)
}

func testNetworkParameterChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateNetworkParameter{
				UpdateNetworkParameter: &types.UpdateNetworkParameter{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_network_parameter.changes"), commands.ErrIsRequired)
}

func testNetworkParameterChangeSubmissionWithoutKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateNetworkParameter{
				UpdateNetworkParameter: &types.UpdateNetworkParameter{
					Changes: &types.NetworkParameter{
						Key: "",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_network_parameter.changes.key"), commands.ErrIsRequired)
}

func testNetworkParameterChangeSubmissionWithKeySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateNetworkParameter{
				UpdateNetworkParameter: &types.UpdateNetworkParameter{
					Changes: &types.NetworkParameter{
						Key: "My key",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_network_parameter.changes.key"), commands.ErrIsRequired)
}

func testNetworkParameterChangeSubmissionWithoutValueFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateNetworkParameter{
				UpdateNetworkParameter: &types.UpdateNetworkParameter{
					Changes: &types.NetworkParameter{
						Value: "",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_network_parameter.changes.value"), commands.ErrIsRequired)
}

func testNetworkParameterChangeSubmissionWithValueSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateNetworkParameter{
				UpdateNetworkParameter: &types.UpdateNetworkParameter{
					Changes: &types.NetworkParameter{
						Value: "My value",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_network_parameter.changes.value"), commands.ErrIsRequired)
}
