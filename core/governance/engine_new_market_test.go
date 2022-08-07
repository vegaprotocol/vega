// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package governance_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalForNewMarket(t *testing.T) {
	t.Run("Submitting a proposal for new market succeeds", testSubmittingProposalForNewMarketSucceeds)
	t.Run("Submitting a duplicated proposal for new market fails", testSubmittingDuplicatedProposalForNewMarketFails)
	t.Run("Submitting a proposal for new market with bad risk parameter fails", testSubmittingProposalForNewMarketWithBadRiskParameterFails)
	t.Run("Submitting a proposal for new market without valid commitment fails", testSubmittingProposalForNewMarketWithoutValidCommitmentFails)

	t.Run("Rejecting a proposal for new market succeeds", testRejectingProposalForNewMarketSucceeds)

	t.Run("Voting for a new market proposal succeeds", testVotingForNewMarketProposalSucceeds)
	t.Run("Voting with a majority of 'yes' makes the new market proposal passed", testVotingWithMajorityOfYesMakesNewMarketProposalPassed)
	t.Run("Voting with a majority of 'no' makes the new market proposal declined", testVotingWithMajorityOfNoMakesNewMarketProposalDeclined)
	t.Run("Voting with insufficient participation makes the new market proposal declined", testVotingWithInsufficientParticipationMakesNewMarketProposalDeclined)
}

func testSubmittingProposalForNewMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow())

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	assert.True(t, toSubmit.IsNewMarket())
	require.NotNil(t, toSubmit.NewMarket().Market())
	require.NotNil(t, toSubmit.NewMarket().LiquidityProvisionSubmission())
}

func testSubmittingDuplicatedProposalForNewMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(party, eng.tsvc.GetTimeNow())

	// setup
	eng.ensureTokenBalanceForParty(t, party, 1000)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, party, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	duplicatedProposal := proposal
	duplicatedProposal.Reference = "this-is-a-copy"

	// when
	_, err = eng.submitProposal(t, duplicatedProposal)

	// then
	require.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	// given
	duplicatedProposal = proposal
	duplicatedProposal.State = types.ProposalStatePassed

	// when
	_, err = eng.submitProposal(t, duplicatedProposal)

	// then
	require.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error(), "reject attempt to change state indirectly")
}

func testSubmittingProposalForNewMarketWithBadRiskParameterFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 1)
	eng.ensureAllAssetEnabled(t)

	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow())
	proposal.Terms.GetNewMarket().Changes.RiskParameters = &types.NewMarketConfigurationLogNormal{
		LogNormal: &types.LogNormalRiskModel{
			Params: nil, // it's nil by zero value, but eh, let's show that's what we test
		},
	}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid risk parameter")
}

func testSubmittingProposalForNewMarketWithoutValidCommitmentFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(party, eng.tsvc.GetTimeNow())

	eng.ensureAllAssetEnabled(t)

	// first we test with no commitment - this should not return an error
	proposal.Terms.GetNewMarket().LiquidityCommitment = nil
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectOpenProposalEvent(t, party, proposal.ID)
	_, err := eng.submitProposal(t, proposal)
	require.NoError(t, err)
	// assert.Contains(t, err.Error(), "market proposal is missing liquidity commitment")

	// ensure unique ID
	proposal.ID += "2"
	// Then no amount
	proposal.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	proposal.Terms.GetNewMarket().LiquidityCommitment.CommitmentAmount = num.UintZero()
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorMissingCommitmentAmount)
	_, err = eng.submitProposal(t, proposal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proposal commitment amount is 0 or missing")

	// Then empty fees
	proposal.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	proposal.Terms.GetNewMarket().LiquidityCommitment.Fee = num.DecimalZero()
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInvalidFeeAmount)
	_, err = eng.submitProposal(t, proposal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid liquidity provision fee")

	// Then negative fees
	proposal.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	proposal.Terms.GetNewMarket().LiquidityCommitment.Fee = num.DecimalFromFloat(-1)
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInvalidFeeAmount)
	_, err = eng.submitProposal(t, proposal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid liquidity provision fee")

	// Then empty shapes
	proposal.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	proposal.Terms.GetNewMarket().LiquidityCommitment.Buys = nil
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInvalidShape)
	_, err = eng.submitProposal(t, proposal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty SIDE_BUY shape")

	proposal.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	proposal.Terms.GetNewMarket().LiquidityCommitment.Sells = nil
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInvalidShape)
	_, err = eng.submitProposal(t, proposal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty SIDE_SELL shape")

	// Then invalid shapes
	proposal.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	proposal.Terms.GetNewMarket().LiquidityCommitment.Buys[0].Reference = types.PeggedReferenceBestAsk
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInvalidShape)
	_, err = eng.submitProposal(t, proposal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order in buy side shape with best ask price reference")

	proposal.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	proposal.Terms.GetNewMarket().LiquidityCommitment.Sells[0].Reference = types.PeggedReferenceBestBid
	eng.ensureTokenBalanceForParty(t, party, 1)
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInvalidShape)
	_, err = eng.submitProposal(t, proposal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order in sell side shape with best bid price reference")
}

func testRejectingProposalForNewMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(party, eng.tsvc.GetTimeNow())

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, party, 10000)

	// expect
	eng.expectOpenProposalEvent(t, party, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)

	// expect
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorCouldNotInstantiateMarket)

	// when
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), types.ProposalErrorCouldNotInstantiateMarket, assert.AnError)

	// then
	require.NoError(t, err)

	// when
	// Just one more time to make sure it was removed from proposals.
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), types.ProposalErrorCouldNotInstantiateMarket, assert.AnError)

	// then
	assert.EqualError(t, err, governance.ErrProposalDoesNotExist.Error())
}

func testVotingForNewMarketProposalSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow())

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 1)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 1)

	// expect
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addYesVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)
}

func testVotingWithMajorityOfYesMakesNewMarketProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow())

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

	// when
	eng.OnTick(context.Background(), afterClosing)

	// given
	voter2 := vgrand.RandomStr(5)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotOpenForVotes.Error())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	require.Len(t, toBeEnacted, 1)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalDoesNotExist.Error())
}

func testVotingWithMajorityOfNoMakesNewMarketProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow())

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureStakingAssetTotalSupply(t, 200)
	eng.ensureTokenBalanceForParty(t, proposer, 100)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addYesVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// setup
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addNoVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorMajorityThresholdNotReached)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "100")

	// when
	_, voteClosed := eng.OnTick(context.Background(), afterClosing)

	// then
	require.Len(t, voteClosed, 1)
	vc := voteClosed[0]
	require.NotNil(t, vc.NewMarket())
	assert.True(t, vc.NewMarket().Rejected())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	assert.Empty(t, toBeEnacted)
}

func testVotingWithInsufficientParticipationMakesNewMarketProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow())

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureStakingAssetTotalSupply(t, 800)
	eng.ensureTokenBalanceForParty(t, proposer, 100)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addYesVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 100)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorParticipationThresholdNotReached)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "100")

	// when
	_, voteClosed := eng.OnTick(context.Background(), afterClosing)

	// then
	require.Len(t, voteClosed, 1)
	vc := voteClosed[0]
	require.NotNil(t, vc.NewMarket())
	assert.True(t, vc.NewMarket().Rejected())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	assert.Empty(t, toBeEnacted)
}
