package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestVoteSubmission(t *testing.T) {
	var cases = []struct {
		vote      commandspb.VoteSubmission
		errString string
	}{
		{
			vote: commandspb.VoteSubmission{
				Value:      types.Vote_VALUE_YES,
				ProposalId: "OKPROPOSALID",
			},
		},
		{
			vote: commandspb.VoteSubmission{
				ProposalId: "OKPROPOSALID",
			},
			errString: "vote_submission.value(is required)",
		},
		{
			vote: commandspb.VoteSubmission{
				Value:      types.Vote_Value(-42),
				ProposalId: "OKPROPOSALID",
			},
			errString: "vote_submission.value(is not a valid value)",
		},
		{
			vote: commandspb.VoteSubmission{
				Value: types.Vote_VALUE_NO,
			},
			errString: "vote_submission.proposal_id(is required)",
		},
		{
			vote:      commandspb.VoteSubmission{},
			errString: "vote_submission.proposal_id(is required), vote_submission.value(is required)",
		},
	}

	for _, c := range cases {
		err := commands.CheckVoteSubmission(&c.vote)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.Error(t, err)
		assert.EqualError(t, err, c.errString)
	}
}
