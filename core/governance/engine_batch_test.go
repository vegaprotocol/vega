package governance_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmitBatchProposals(t *testing.T) {
	t.Run("Submitting batch fails if any of the terms fails validation", testSubmittingBatchProposalFailsWhenTermValidationFails)

	// t.Run("Submitting a batch proposal with enactment time too soon fails", testSubmittingBatchProposalWithEnactmentTimeTooSoonFails)
	// t.Run("Submitting a proposal with enactment time too soon fails", testSubmittingProposalWithEnactmentTimeTooSoonFails)
	// t.Run("Submitting a proposal with enactment time too late fails", testSubmittingProposalWithEnactmentTimeTooLateFails)
	// t.Run("Submitting a proposal with non-existing account fails", testSubmittingProposalWithNonExistingAccountFails)
	// t.Run("Submitting a proposal with internal time termination with non-existing account fails", testSubmittingProposalWithInternalTimeTerminationWithNonExistingAccountFails)
	// t.Run("Submitting a proposal without enough stake fails", testSubmittingProposalWithoutEnoughStakeFails)
	// t.Run("Submitting an update market proposal without enough stake and els fails", testSubmittingUpdateMarketProposalWithoutEnoughStakeAndELSFails)
	// t.Run("Submitting a proposal with internal time termination without enough stake fails", testSubmittingProposalWithInternalTimeTerminationWithoutEnoughStakeFails)

	// t.Run("Submitting a time-triggered proposal for new market with termination time before enactment time fails", testSubmittingTimeTriggeredProposalNewMarketTerminationBeforeEnactmentFails)

	// t.Run("Voting on non-existing proposal fails", testVotingOnNonExistingProposalFails)
	// t.Run("Voting with non-existing account fails", testVotingWithNonExistingAccountFails)
	// t.Run("Voting without token fails", testVotingWithoutTokenFails)

	// t.Run("Test multiple proposal lifecycle", testMultipleProposalsLifecycle)
	// t.Run("Withdrawing vote assets removes vote from proposal state calculation", testWithdrawingVoteAssetRemovesVoteFromProposalStateCalculation)

	// t.Run("Updating voters key on votes succeeds", testUpdatingVotersKeyOnVotesSucceeds)
	// t.Run("Updating voters key on votes with internal time termination succeeds", testUpdatingVotersKeyOnVotesWithInternalTimeTerminationSucceeds)

	// t.Run("Computing the governance state hash is deterministic", testComputingGovernanceStateHashIsDeterministic)
	// t.Run("Submit proposal update market", testSubmitProposalMarketUpdate)
}

func testSubmittingBatchProposalFailsWhenTermValidationFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	now := eng.tsvc.GetTimeNow().Add(2 * time.Hour)
	party := vgrand.RandomStr(5)

	closingTime := now.Add(48 * time.Hour).Unix()

	newMarketProposal := eng.newProposalForNewMarket(party, now, nil, nil, true)
	newMarketProposal.Terms.EnactmentTimestamp = time.Now().Unix()

	newFormProposal := eng.newFreeformProposal(party, now)
	newNetParamProposal := eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now)

	cases := []struct {
		msg            string
		submission     types.BatchProposalSubmission
		expectProposal []expectedProposal
	}{
		{
			msg: "New market fails",
			submission: eng.newBatchSubmission(
				closingTime,
				newFormProposal,
				newNetParamProposal,
				newMarketProposal,
			),
			expectProposal: []expectedProposal{
				{
					partyID:    party,
					proposalID: newFormProposal.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorInsufficientTokens,
				},
				{
					partyID:    party,
					proposalID: newNetParamProposal.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorInsufficientTokens,
				},
				{
					partyID:    party,
					proposalID: newMarketProposal.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorEnactTimeTooSoon,
				},
			},
		},
		// {
		// 	msg: "For market update",
		// 	submission: eng.newBatchSubmission(
		// 		closingTime,
		// 		eng.newFreeformProposal(party, now),
		// 		eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, true),
		// 		eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now),
		// 	),
		// },
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// setup
			eng.ensureAllAssetEnabled(tt)

			proposalID := eng.newProposalID()

			// expect
			eng.expectRejectedProposalEvent(tt, party, proposalID, types.ProposalErrorProposalInBatchRejected)
			eng.expectProposalEvents(tt, tc.expectProposal)

			// when
			_, err := eng.submitBatchProposal(tt, tc.submission, proposalID, party)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal enactment time too soon")
		})
	}
}

func testSubmittingBatchProposalWithEnactmentTimeTooSoonFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	now := eng.tsvc.GetTimeNow().Add(2 * time.Hour)
	closingTimestamp := now.Add(48 * time.Hour).Unix()
	party := vgrand.RandomStr(5)

	newMarketProposal := eng.newProposalForNewMarket(party, now, nil, nil, true)
	newMarketProposal.Terms.EnactmentTimestamp = now.Unix()

	updateMarketProposal := eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, true)
	updateMarketProposal.Terms.EnactmentTimestamp = now.Unix()

	cases := []struct {
		msg        string
		submission types.BatchProposalSubmission
	}{
		{
			msg: "For new market",
			submission: eng.newBatchSubmission(
				closingTimestamp,
				eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now),
				newMarketProposal,
			),
		},
		{
			msg: "For market update",
			submission: eng.newBatchSubmission(
				closingTimestamp,
				eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now),
				updateMarketProposal,
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// setup
			eng.ensureAllAssetEnabled(tt)

			proposalID := eng.newProposalID()

			eng.expectRejectedProposalEvent(tt, party, proposalID, types.ProposalErrorEnactTimeTooSoon)

			// when
			_, err := eng.submitBatchProposal(tt, tc.submission, proposalID, party)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal enactment time too soon, expected >")
		})
	}
}
