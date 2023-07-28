package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestSubmittingNilVoteFails(t *testing.T) {
	err := checkVoteSubmission(nil)

	assert.Contains(t, err.Get("vote_submission"), commands.ErrIsRequired)
}

func TestVoteSubmission(t *testing.T) {
	cases := []struct {
		vote      commandspb.VoteSubmission
		errString string
	}{
		{
			vote: commandspb.VoteSubmission{
				Value:      types.Vote_VALUE_YES,
				ProposalId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
		},
		{
			vote: commandspb.VoteSubmission{
				ProposalId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "vote_submission.value (is required)",
		},
		{
			vote: commandspb.VoteSubmission{
				Value:      types.Vote_Value(-42),
				ProposalId: "08dce6ebf50e34fedee32860b6f459824e4b834762ea66a96504fdc57a9c4741",
			},
			errString: "vote_submission.value (is not a valid value)",
		},
		{
			vote: commandspb.VoteSubmission{
				Value: types.Vote_VALUE_NO,
			},
			errString: "vote_submission.proposal_id (is required)",
		},
		{
			vote:      commandspb.VoteSubmission{},
			errString: "vote_submission.proposal_id (is required), vote_submission.value (is required)",
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

func checkVoteSubmission(cmd *commandspb.VoteSubmission) commands.Errors {
	err := commands.CheckVoteSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
