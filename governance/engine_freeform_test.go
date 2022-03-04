package governance_test

import (
	"context"
	"testing"
	"time"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreeformProposal(t *testing.T) {
	t.Run("Submitting a freeform proposal succeeds", testSubmittingFreeformProposalSucceeds)
	t.Run("Submitting an invalid freeform proposal fails", testSubmittingInvalidFreeformProposalFails)
	t.Run("Freeform proposal does not wait for enactment timestamp", testFreeformProposalDoesNotWaitToEnact)
}

func testSubmittingFreeformProposalSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newFreeformProposal(party.Id, time.Now())

	// setup
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
}

func testSubmittingInvalidFreeformProposalFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	id := eng.newProposalID()
	now := time.Now()
	d := "I am much too long I am much too long I am much too long I am much too long I am much too long"
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     "a-valid-party",
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change: &types.ProposalTermsNewFreeform{
				NewFreeform: &types.NewFreeform{
					Changes: &types.NewFreeformDetails{
						URL:         "https://example.com",
						Description: d + d + d,
						Hash:        "2fb572edea4af9154edeff680e23689ed076d08934c60f8a4c1f5743a614954e",
					},
				},
			},
		},
	}

	// setup
	eng.expectRejectedProposalEvent(t, party.Id, proposal.ID, types.ProposalErrorInvalidFreeform)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	assert.ErrorIs(t, err, governance.ErrFreeformDescriptionTooLong)
	assert.Nil(t, toSubmit)
}

func testFreeformProposalDoesNotWaitToEnact(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newFreeformProposal(proposer, time.Now())

	// setup
	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 1)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectVoteEvent(t, voter1, proposal.ID)

	// then
	err = eng.addYesVote(t, voter1, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")

	// when the proposal is closed, it is enacted immediately
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// then
	require.Len(t, toBeEnacted, 1)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// given
	voter2 := vgrand.RandomStr(5)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalDoesNotExist.Error())
}
