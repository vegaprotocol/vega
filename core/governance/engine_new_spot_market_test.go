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

func TesSpottProposalForNewMarket(t *testing.T) {
	t.Helper()
	t.Run("Submitting a proposal for new spot market succeeds", testSubmittingProposalForNewSpotMarketSucceeds)
	t.Run("Submitting a duplicated proposal for new spot market fails", testSubmittingDuplicatedProposalForNewSpotMarketFails)
	t.Run("Submitting a proposal for new spot market with bad risk parameter fails", testSubmittingProposalForNewSpotMarketWithBadRiskParameterFails)
	t.Run("Rejecting a proposal for new spot market succeeds", testRejectingProposalForNewSpotMarketSucceeds)
	t.Run("Voting for a new spot market proposal succeeds", testVotingForNewSpotMarketProposalSucceeds)
	t.Run("Voting with a majority of 'yes' makes the new spot market proposal passed", testVotingWithMajorityOfYesMakesNewSpotMarketProposalPassed)
	t.Run("Voting with a majority of 'no' makes the new spot market proposal declined", testVotingWithMajorityOfNoMakesNewSpotMarketProposalDeclined)
	t.Run("Voting with insufficient participation makes the new spot market proposal declined", testVotingWithInsufficientParticipationMakesNewSpotMarketProposalDeclined)
}

func testSubmittingProposalForNewSpotMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewSpotMarket(party.Id, eng.tsvc.GetTimeNow())

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	assert.True(t, toSubmit.IsNewSpotMarket())
	require.NotNil(t, toSubmit.NewSpotMarket().Market())
}

func testSubmittingDuplicatedProposalForNewSpotMarketFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewSpotMarket(party, eng.tsvc.GetTimeNow())

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

func testSubmittingProposalForNewSpotMarketWithBadRiskParameterFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := eng.newValidParty("a-valid-party", 1)
	eng.ensureAllAssetEnabled(t)

	proposal := eng.newProposalForNewSpotMarket(party.Id, eng.tsvc.GetTimeNow())
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

func TestSubmittingProposalForNewSpotMarketWithOutOfRangeRiskParameterFails(t *testing.T) {
	lnm := &types.LogNormalRiskModel{}
	lnm.RiskAversionParameter = num.DecimalFromFloat(1e-8 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.RiskAversionParameter = num.DecimalFromFloat(1e1 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.RiskAversionParameter = num.DecimalFromFloat(1e-6)
	lnm.Tau = num.DecimalFromFloat(1e-8 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Tau = num.DecimalFromFloat(1 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Tau = num.DecimalOne()
	lnm.Params = &types.LogNormalModelParams{}
	lnm.Params.Mu = num.DecimalFromFloat(-1e-6 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Mu = num.DecimalFromFloat(1e-6 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Mu = num.DecimalFromFloat(0.0)
	lnm.Params.R = num.DecimalFromFloat(-1 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.R = num.DecimalFromFloat(1 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.R = num.DecimalFromFloat(0.0)
	lnm.Params.Sigma = num.DecimalFromFloat(1e-3 - 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Sigma = num.DecimalFromFloat(50 + 1e-12)
	testOutOfRangeRiskParamFail(t, lnm)
	lnm.Params.Sigma = num.DecimalFromFloat(1.0)

	// now all risk params are valid
	eng := getTestEngine(t, time.Now())

	// given
	party := eng.newValidParty("a-valid-party", 1)
	eng.ensureAllAssetEnabled(t)

	proposal := eng.newProposalForNewSpotMarket(party.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour))
	proposal.Terms.GetNewSpotMarket().Changes.RiskParameters = &types.NewSpotMarketConfigurationLogNormal{LogNormal: lnm}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
}

func testRejectingProposalForNewSpotMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewSpotMarket(party, eng.tsvc.GetTimeNow())

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

func testVotingForNewSpotMarketProposalSucceeds(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewSpotMarket(proposer, eng.tsvc.GetTimeNow())

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

func testVotingWithMajorityOfYesMakesNewSpotMarketProposalPassed(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewSpotMarket(proposer, eng.tsvc.GetTimeNow())

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
	eng.expectGetMarketState(t, proposal.ID)

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

func testVotingWithMajorityOfNoMakesNewSpotMarketProposalDeclined(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewSpotMarket(proposer, eng.tsvc.GetTimeNow())

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
	eng.expectGetMarketState(t, proposal.ID)

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

func testVotingWithInsufficientParticipationMakesNewSpotMarketProposalDeclined(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewSpotMarket(proposer, eng.tsvc.GetTimeNow())

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
	eng.expectGetMarketState(t, proposal.ID)
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
