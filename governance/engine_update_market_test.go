package governance_test

import (
	"context"
	"testing"
	"time"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalForMarketUpdate(t *testing.T) {
	t.Run("Submitting a proposal for market update succeeds", testSubmittingProposalForMarketUpdateSucceeds)
	t.Run("Submitting a proposal for market update on unknown market fails", testSubmittingProposalForMarketUpdateForUnknownMarketFails)
	t.Run("Submitting a proposal for market update with insufficient equity-like share fails", testSubmittingProposalForMarketUpdateWithInsufficientEquityLikeShareFails)

	t.Run("Rejecting a proposal for market update succeeds", testRejectingProposalForMarketUpdateSucceeds)

	t.Run("Voting without reaching minimum of tokens and equity-like shares makes the market update proposal declined", testVotingWithoutMinimumTokenHoldersAndEquityLikeShareMakesMarketUpdateProposalPassed)
	t.Run("Voting with a majority of 'yes' from tokens makes the market update proposal passed", testVotingWithMajorityOfYesFromTokenHoldersMakesMarketUpdateProposalPassed)
	t.Run("Voting with a majority of 'no' from tokens makes the market update proposal declined", testVotingWithMajorityOfNoFromTokenHoldersMakesMarketUpdateProposalDeclined)
	t.Run("Voting without reaching minimum of tokens and a majority of 'yes' from equity-like shares makes the market update proposal passed", testVotingWithoutTokenAndMajorityOfYesFromEquityLikeShareHoldersMakesMarketUpdateProposalPassed)
	t.Run("Voting without reaching minimum of tokens and a majority of 'no' from equity-like shares makes the market update proposal declined", testVotingWithoutTokenAndMajorityOfNoFromEquityLikeShareHoldersMakesMarketUpdateProposalDeclined)
}

func testSubmittingProposalForMarketUpdateSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(proposer, time.Now())

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, proposer, 0.1)
	eng.ensureExistingMarket(t, proposal.ID)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
}

func testSubmittingProposalForMarketUpdateForUnknownMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(proposer, time.Now())

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 123456789)
	eng.ensureNonExistingMarket(t, proposal.ID)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidMarket)

	// when
	toSubmit, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer)

	// then
	require.ErrorIs(t, governance.ErrMarketDoesNotExist, err)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForMarketUpdateWithInsufficientEquityLikeShareFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(party, time.Now())

	// setup
	eng.ensureTokenBalanceForParty(t, party, 100)
	eng.ensureExistingMarket(t, proposal.ID)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, party, 0.05)

	// expect
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInsufficientEquityLikeShare)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proposer have insufficient equity-like share, expected >=")
	require.Nil(t, toSubmit)
}

func testRejectingProposalForMarketUpdateSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(party, time.Now())

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureExistingMarket(t, proposal.ID)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, party, 0.7)
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketMinProposerEquityLikeShare, "0.1")
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

func testVotingWithoutMinimumTokenHoldersAndEquityLikeShareMakesMarketUpdateProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(proposer, time.Now())

	// setup
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipation, "0.5")
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipationLP, "0.5")
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, proposer, 0.1)
	eng.ensureExistingMarket(t, proposal.ID)
	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// Vote using a token holder without equity-like share.
	// when
	voterWithToken := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithToken, 1)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken, proposal.ID)

	// when
	err = eng.addYesVote(t, voterWithToken, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote using equity-like share holder without tokens.
	// given
	voterWithELS := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS, 0.1)

	// expect
	eng.expectVoteEvent(t, voterWithELS, proposal.ID)

	// when
	err = eng.addNoVote(t, voterWithELS, proposal.ID)

	// then
	require.NoError(t, err)

	// Closing the proposal.
	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureStakingAssetTotalSupply(t, 10)
	eng.ensureTokenBalanceForParty(t, voterWithToken, 1)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS, 0.1)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorParticipationThresholdNotReached)
	eng.expectVoteEvents(t)

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)
}

func testVotingWithMajorityOfYesFromTokenHoldersMakesMarketUpdateProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(proposer, time.Now())

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, proposer, 0.7)
	eng.ensureExistingMarket(t, proposal.ID)
	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// Vote 'YES' with 10 tokens.
	// given
	voterWithToken1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithToken1, 10)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken1, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken1, proposal.ID)

	// when
	err = eng.addYesVote(t, voterWithToken1, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'NO' with 2 tokens.
	// given
	voterWithToken2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithToken2, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken2, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken2, proposal.ID)

	// then
	err = eng.addNoVote(t, voterWithToken2, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'NO' with 0.1 of equity-like share.
	// given
	voterWithELS1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS1, 0.1)

	// expect
	eng.expectVoteEvent(t, voterWithELS1, proposal.ID)

	// when
	err = eng.addNoVote(t, voterWithELS1, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'NO' with 0.5 of equity-like share.
	// given
	voterWithELS2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS2, 0.7)

	// expect
	eng.expectVoteEvent(t, voterWithELS2, proposal.ID)

	// when
	err = eng.addNoVote(t, voterWithELS2, proposal.ID)

	// then
	require.NoError(t, err)

	// Close the proposal.
	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureStakingAssetTotalSupply(t, 13)
	eng.ensureTokenBalanceForParty(t, voterWithToken1, 10)
	eng.ensureTokenBalanceForParty(t, voterWithToken2, 2)
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectVoteEvents(t)

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)
}

func testVotingWithMajorityOfNoFromTokenHoldersMakesMarketUpdateProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(proposer, time.Now())

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, proposer, 0.7)
	eng.ensureExistingMarket(t, proposal.ID)
	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// Vote 'NO' with 10 tokens.
	// given
	voterWithToken1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithToken1, 10)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken1, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken1, proposal.ID)

	// when
	err = eng.addNoVote(t, voterWithToken1, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'YES' with 2 tokens.
	// given
	voterWithToken2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithToken2, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken2, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken2, proposal.ID)

	// then
	err = eng.addYesVote(t, voterWithToken2, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'YES' with 0.1 of equity-like share.
	// given
	voterWithELS1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS1, 0.1)

	// expect
	eng.expectVoteEvent(t, voterWithELS1, proposal.ID)

	// when
	err = eng.addYesVote(t, voterWithELS1, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'YES' with 0.5 of equity-like share.
	// given
	voterWithELS2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS2, 0.7)

	// expect
	eng.expectVoteEvent(t, voterWithELS2, proposal.ID)

	// when
	err = eng.addYesVote(t, voterWithELS2, proposal.ID)

	// then
	require.NoError(t, err)

	// Close the proposal.
	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureStakingAssetTotalSupply(t, 13)
	eng.ensureTokenBalanceForParty(t, voterWithToken1, 10)
	eng.ensureTokenBalanceForParty(t, voterWithToken2, 2)
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorMajorityThresholdNotReached)
	eng.expectVoteEvents(t)

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)
}

func testVotingWithoutTokenAndMajorityOfYesFromEquityLikeShareHoldersMakesMarketUpdateProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(proposer, time.Now())

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, proposer, 0.7)
	eng.ensureExistingMarket(t, proposal.ID)
	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// Vote 'NO' with 2 tokens.
	// given
	voterWithToken := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithToken, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken, proposal.ID)

	// when
	err = eng.addNoVote(t, voterWithToken, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'NO' with 0.1 of equity-like share.
	// given
	voterWithELS1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS1, 0.1)

	// expect
	eng.expectVoteEvent(t, voterWithELS1, proposal.ID)

	// when
	err = eng.addNoVote(t, voterWithELS1, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'YES' with 0.5 of equity-like share.
	// given
	voterWithELS2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS2, 0.7)

	// expect
	eng.expectVoteEvent(t, voterWithELS2, proposal.ID)

	// when
	err = eng.addYesVote(t, voterWithELS2, proposal.ID)

	// then
	require.NoError(t, err)

	// Close the proposal.
	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipation, "0.5")
	eng.ensureStakingAssetTotalSupply(t, 13)
	eng.ensureTokenBalanceForParty(t, voterWithToken, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS1, 0.1)
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS2, 0.7)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectVoteEvents(t)

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)
}

func testVotingWithoutTokenAndMajorityOfNoFromEquityLikeShareHoldersMakesMarketUpdateProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate(proposer, time.Now())

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, proposer, 0.7)
	eng.ensureExistingMarket(t, proposal.ID)
	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureAllAssetEnabled(t)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// Vote 'YES' with 2 tokens.
	// given
	voterWithToken := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithToken, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken, proposal.ID)

	// when
	err = eng.addYesVote(t, voterWithToken, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'YES' with 0.1 of equity-like share.
	// given
	voterWithELS1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS1, 0.1)

	// expect
	eng.expectVoteEvent(t, voterWithELS1, proposal.ID)

	// when
	err = eng.addYesVote(t, voterWithELS1, proposal.ID)

	// then
	require.NoError(t, err)

	// Vote 'NO' with 0.5 of equity-like share.
	// given
	voterWithELS2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS2, 0.7)

	// expect
	eng.expectVoteEvent(t, voterWithELS2, proposal.ID)

	// when
	err = eng.addNoVote(t, voterWithELS2, proposal.ID)

	// then
	require.NoError(t, err)

	// Close the proposal.
	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipation, "0.5")
	eng.ensureStakingAssetTotalSupply(t, 13)
	eng.ensureTokenBalanceForParty(t, voterWithToken, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithToken, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS1, 0.1)
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.ID, voterWithELS2, 0.7)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorMajorityThresholdNotReached)
	eng.expectVoteEvents(t)

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)
}
