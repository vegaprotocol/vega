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
