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

package governance_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmitBatchProposals(t *testing.T) {
	t.Run("Submitted batch proposal is declined", testSubmittingBatchProposalDeclined)
	t.Run("Submitted batch proposal has passed", testSubmittingBatchProposalPassed)
	t.Run("Submitting batch fails if any of the terms fails validation", testSubmittingBatchProposalFailsWhenTermValidationFails)

	t.Run("Voting with non-existing account fails", testVotingOnBatchWithNonExistingAccountFails)
	t.Run("Voting without token fails", testVotingOnBatchWithoutTokenFails)
}

func testSubmittingBatchProposalDeclined(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	now := eng.tsvc.GetTimeNow().Add(2 * time.Hour)
	party := vgrand.RandomStr(5)

	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, party, 1)

	batchID := eng.newProposalID()

	newFormProposal := eng.newFreeformProposal(party, now)
	newNetParamProposal := eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now)
	newMarketProposal := eng.newProposalForNewMarket(party, now, nil, nil, true)

	// expect
	eng.expectOpenProposalEvent(t, party, batchID)
	eng.expectProposalEvents(t, []expectedProposal{
		{
			partyID:    party,
			proposalID: newFormProposal.ID,
			state:      types.ProposalStateOpen,
			reason:     types.ProposalErrorUnspecified,
		},
		{
			partyID:    party,
			proposalID: newNetParamProposal.ID,
			state:      types.ProposalStateOpen,
			reason:     types.ProposalErrorUnspecified,
		},
		{
			partyID:    party,
			proposalID: newMarketProposal.ID,
			state:      types.ProposalStateOpen,
			reason:     types.ProposalErrorUnspecified,
		},
	})

	batchClosingTime := now.Add(48 * time.Hour)

	// when
	_, err := eng.submitBatchProposal(t, eng.newBatchSubmission(
		batchClosingTime.Unix(),
		newFormProposal,
		newNetParamProposal,
		newMarketProposal,
	), batchID, party)

	assert.NoError(t, err)
	ctx := context.Background()

	eng.expectDeclinedProposalEvent(t, batchID, types.ProposalErrorProposalInBatchDeclined)
	eng.expectProposalEvents(t, []expectedProposal{
		{
			partyID:    party,
			proposalID: newFormProposal.ID,
			state:      types.ProposalStateDeclined,
			reason:     types.ProposalErrorParticipationThresholdNotReached,
		},
		{
			partyID:    party,
			proposalID: newNetParamProposal.ID,
			state:      types.ProposalStateDeclined,
			reason:     types.ProposalErrorParticipationThresholdNotReached,
		},
		{
			partyID:    party,
			proposalID: newMarketProposal.ID,
			state:      types.ProposalStateDeclined,
			reason:     types.ProposalErrorParticipationThresholdNotReached,
		},
	})

	eng.accounts.EXPECT().GetStakingAssetTotalSupply().AnyTimes().Return(num.NewUint(200))
	eng.OnTick(ctx, batchClosingTime.Add(1*time.Second))
}

func testSubmittingBatchProposalPassed(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	now := eng.tsvc.GetTimeNow().Add(2 * time.Hour)
	party := vgrand.RandomStr(5)

	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, party, 1)

	batchID := eng.newProposalID()

	newFormProposal := eng.newFreeformProposal(party, now)
	newNetParamProposal := eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now)
	newMarketProposal := eng.newProposalForNewMarket(party, now, nil, nil, true)

	// expect
	eng.expectOpenProposalEvent(t, party, batchID)
	eng.expectProposalEvents(t, []expectedProposal{
		{
			partyID:    party,
			proposalID: newFormProposal.ID,
			state:      types.ProposalStateOpen,
			reason:     types.ProposalErrorUnspecified,
		},
		{
			partyID:    party,
			proposalID: newNetParamProposal.ID,
			state:      types.ProposalStateOpen,
			reason:     types.ProposalErrorUnspecified,
		},
		{
			partyID:    party,
			proposalID: newMarketProposal.ID,
			state:      types.ProposalStateOpen,
			reason:     types.ProposalErrorUnspecified,
		},
	})

	batchClosingTime := now.Add(48 * time.Hour)

	// when
	_, err := eng.submitBatchProposal(t, eng.newBatchSubmission(
		batchClosingTime.Unix(),
		newFormProposal,
		newNetParamProposal,
		newMarketProposal,
	), batchID, party)

	assert.NoError(t, err)
	ctx := context.Background()

	eng.accounts.EXPECT().GetStakingAssetTotalSupply().AnyTimes().Return(num.NewUint(200))

	for i := 0; i < 10; i++ {
		partyID := fmt.Sprintf("party-%d", i)
		eng.ensureTokenBalanceForParty(t, partyID, 20)
		eng.expectVoteEvent(t, partyID, batchID)
		err = eng.addYesVote(t, partyID, batchID)
		assert.NoError(t, err)
	}

	eng.expectPassedProposalEvent(t, batchID)

	expectedProposals := []expectedProposal{
		{
			partyID:    party,
			proposalID: newFormProposal.ID,
			state:      types.ProposalStatePassed,
		},
		{
			partyID:    party,
			proposalID: newNetParamProposal.ID,
			state:      types.ProposalStatePassed,
		},
		{
			partyID:    party,
			proposalID: newMarketProposal.ID,
			state:      types.ProposalStatePassed,
		},
	}

	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		i := 0
		for _, evt := range evts {
			switch e := evt.(type) {
			case *events.Proposal:
				p := e.Proposal()
				assert.Equal(t, expectedProposals[i].proposalID, p.Id)
				assert.Equal(t, expectedProposals[i].partyID, p.PartyId)
				assert.Equal(t, expectedProposals[i].state.String(), p.State.String())
				i++
			}
		}
	})

	eng.OnTick(ctx, batchClosingTime.Add(1*time.Second))
}

func testSubmittingBatchProposalFailsWhenTermValidationFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	now := eng.tsvc.GetTimeNow().Add(2 * time.Hour)
	party := vgrand.RandomStr(5)

	newFormProposal := eng.newFreeformProposal(party, now)
	newNetParamProposal := eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now)
	newMarketProposal := eng.newProposalForNewMarket(party, now, nil, nil, true)
	newMarketProposal.Terms.EnactmentTimestamp = time.Now().Unix()

	newFormProposal2 := eng.newFreeformProposal(party, now)
	newNetParamProposal2 := eng.newProposalForNetParam(party, netparams.MarketAuctionMaximumDuration, "10h", now)
	newNetParamProposal2.Terms.EnactmentTimestamp = now.Add(24 * 365 * time.Hour).Unix()
	newMarketProposal2 := eng.newProposalForNewMarket(party, now, nil, nil, true)

	batchClosingTime := now.Add(48 * time.Hour).Unix()

	cases := []struct {
		msg            string
		submission     types.BatchProposalSubmission
		expectProposal []expectedProposal
		containsError  string
	}{
		{
			msg:           "New market rejected and other proposals with it",
			containsError: "proposal enactment time too soon",
			submission: eng.newBatchSubmission(
				batchClosingTime,
				newFormProposal,
				newNetParamProposal,
				newMarketProposal,
			),
			expectProposal: []expectedProposal{
				{
					partyID:    party,
					proposalID: newFormProposal.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorProposalInBatchRejected,
				},
				{
					partyID:    party,
					proposalID: newNetParamProposal.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorProposalInBatchRejected,
				},
				{
					partyID:    party,
					proposalID: newMarketProposal.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorEnactTimeTooSoon,
				},
			},
		},
		{
			msg:           "Net parameter is rejected and the whole batch with it",
			containsError: "proposal enactment time too late",
			submission: eng.newBatchSubmission(
				batchClosingTime,
				newNetParamProposal2,
				newFormProposal2,
				newMarketProposal2,
			),
			expectProposal: []expectedProposal{
				{
					partyID:    party,
					proposalID: newNetParamProposal2.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorEnactTimeTooLate,
				},
				{
					partyID:    party,
					proposalID: newFormProposal2.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorProposalInBatchRejected,
				},
				{
					partyID:    party,
					proposalID: newMarketProposal2.ID,
					state:      types.ProposalStateRejected,
					reason:     types.ProposalErrorProposalInBatchRejected,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// setup
			eng.ensureAllAssetEnabled(tt)
			eng.ensureTokenBalanceForParty(t, party, 1)

			batchID := eng.newProposalID()

			// expect
			eng.expectRejectedProposalEvent(tt, party, batchID, types.ProposalErrorProposalInBatchRejected)
			eng.expectProposalEvents(tt, tc.expectProposal)

			// when
			_, err := eng.submitBatchProposal(tt, tc.submission, batchID, party)

			// then
			require.Error(tt, err)
			if tc.containsError != "" {
				assert.Contains(tt, err.Error(), tc.containsError)
			}
		})
	}
}

func testVotingOnBatchWithNonExistingAccountFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow().Add(2*time.Hour), nil, nil, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 1)

	batchID := eng.newProposalID()

	// expect
	eng.expectOpenProposalEvent(t, proposer, batchID)
	eng.expectProposalEvents(t, []expectedProposal{
		{
			partyID:    proposer,
			proposalID: proposal.ID,
			state:      types.ProposalStateOpen,
		},
	})

	// when
	sub := eng.newBatchSubmission(
		proposal.Terms.ClosingTimestamp,
		proposal,
	)
	_, err := eng.submitBatchProposal(t, sub, batchID, proposer)

	// then
	require.NoError(t, err)

	// given
	voterWithoutAccount := "voter-no-account"

	// setup
	eng.ensureNoAccountForParty(t, voterWithoutAccount)

	// when
	err = eng.addYesVote(t, voterWithoutAccount, batchID)

	// then
	require.Error(t, err)
	assert.ErrorContains(t, err, "no balance for party")
}

func testVotingOnBatchWithoutTokenFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	proposer := eng.newValidParty("proposer", 1)
	proposal := eng.newProposalForNewMarket(proposer.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), nil, nil, true)

	// setup
	batchID := eng.newProposalID()

	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, batchID)
	eng.expectProposalEvents(t, []expectedProposal{
		{
			partyID:    proposer.Id,
			proposalID: proposal.ID,
			state:      types.ProposalStateOpen,
		},
	})

	// when
	sub := eng.newBatchSubmission(
		proposal.Terms.ClosingTimestamp,
		proposal,
	)
	_, err := eng.submitBatchProposal(t, sub, batchID, proposer.Id)

	// then
	require.NoError(t, err)

	// given
	voterWithEmptyAccount := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithEmptyAccount, 0)

	// when
	err = eng.addYesVote(t, voterWithEmptyAccount, batchID)

	// then
	require.Error(t, err)
	assert.ErrorContains(t, err, governance.ErrVoterInsufficientTokens.Error())
}
