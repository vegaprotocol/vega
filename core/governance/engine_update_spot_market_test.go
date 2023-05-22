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
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalForSpotMarketUpdate(t *testing.T) {
	t.Run("Submitting a proposal for spot market update succeeds", testSubmittingProposalForSpotMarketUpdateSucceeds)
	t.Run("Submitting a proposal for market update on unknown spot market fails", testSubmittingProposalForMarketUpdateForUnknownSpotMarketFails)

	t.Run("Submitting a proposal for market update for not-enacted market fails", testSubmittingProposalForSpotMarketUpdateForNotEnactedMarketFails)
	t.Run("Submitting a proposal for spot market update with insufficient equity-like share fails", testSubmittingProposalForSpotMarketUpdateWithInsufficientEquityLikeShareFails)
	t.Run("Pre-enactment of spot market update proposal succeeds", testPreEnactmentOfSpotMarketUpdateSucceeds)

	t.Run("Rejecting a proposal for market update succeeds", testRejectingProposalForSpotMarketUpdateSucceeds)

	t.Run("Voting without reaching minimum of tokens and equity-like shares makes the spot market update proposal declined", testVotingWithoutMinimumTokenHoldersAndEquityLikeShareMakesSpotMarketUpdateProposalPassed)
	t.Run("Voting with a majority of 'yes' from tokens makes the spot market update proposal passed", testVotingWithMajorityOfYesFromTokenHoldersMakesSpotMarketUpdateProposalPassed)
	t.Run("Voting with a majority of 'no' from tokens makes the spot market update proposal declined", testVotingWithMajorityOfNoFromTokenHoldersMakesSpotMarketUpdateProposalDeclined)
	t.Run("Voting without reaching minimum of tokens and a majority of 'yes' from equity-like shares makes the spot market update proposal passed", testVotingWithoutTokenAndMajorityOfYesFromEquityLikeShareHoldersMakesSpotMarketUpdateProposalPassed)
	t.Run("Voting without reaching minimum of tokens and a majority of 'no' from equity-like shares makes the spot market update proposal declined", testVotingWithoutTokenAndMajorityOfNoFromEquityLikeShareHoldersMakesSpotMarketUpdateProposalDeclined)
}

func testSubmittingProposalForSpotMarketUpdateSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, proposer, 0.1)
	eng.ensureExistingMarket(t, marketID)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
}

func testSubmittingProposalForMarketUpdateForUnknownSpotMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("״market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 123456789)
	eng.ensureNonExistingMarket(t, marketID)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidMarket)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.ErrorIs(t, governance.ErrMarketDoesNotExist, err)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForSpotMarketUpdateForNotEnactedMarketFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	newMarketProposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := newMarketProposal.ID

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 123456789)
	eng.expectOpenProposalEvent(t, proposer, marketID)

	// when
	toSubmit, err := eng.submitProposal(t, newMarketProposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	assert.True(t, toSubmit.IsNewMarket())

	// given
	updateMarketProposal := eng.newProposalForMarketUpdate("״market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	updateMarketProposal.MarketUpdate().MarketID = marketID

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 123456789)
	eng.ensureExistingMarket(t, marketID)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, updateMarketProposal.ID, types.ProposalErrorInvalidMarket)

	// when
	toSubmit, err = eng.submitProposal(t, updateMarketProposal)

	// then
	require.ErrorIs(t, governance.ErrMarketNotEnactedYet, err)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForSpotMarketUpdateWithInsufficientEquityLikeShareFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("״market-1", party, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureTokenBalanceForParty(t, party, 100)
	eng.ensureExistingMarket(t, marketID)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, party, 0.05)

	// expect
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorInsufficientEquityLikeShare)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proposer have insufficient equity-like share, expected >=")
	require.Nil(t, toSubmit)
}

func testPreEnactmentOfSpotMarketUpdateSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("״market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, proposer, 0.7)
	eng.ensureExistingMarket(t, marketID)
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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken1, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken2, 0)

	// expect
	eng.expectVoteEvent(t, voterWithToken2, proposal.ID)

	// then
	err = eng.addNoVote(t, voterWithToken2, proposal.ID)

	// then
	require.NoError(t, err)

	// Close the proposal.
	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureStakingAssetTotalSupply(t, 13)
	eng.ensureTokenBalanceForParty(t, voterWithToken1, 10)
	eng.ensureTokenBalanceForParty(t, voterWithToken2, 2)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectVoteEvents(t)
	eng.expectGetMarketState(t, marketID)

	// when
	eng.OnTick(context.Background(), afterClosing)

	// Enact the proposal.
	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	existingMarket := types.Market{
		ID: marketID,
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Name: vgrand.RandomStr(10),
				Product: &types.InstrumentFuture{
					Future: &types.Future{
						SettlementAsset: "BTC",
					},
				},
			},
		},
		DecimalPlaces:         3,
		PositionDecimalPlaces: 4,
		OpeningAuction: &types.AuctionDuration{
			Duration: 42,
		},
	}

	// setup
	eng.ensureGetMarket(t, marketID, existingMarket)

	// when
	enacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	require.NotEmpty(t, enacted)
	require.True(t, enacted[0].IsUpdateMarket())
	updatedMarket := enacted[0].UpdateMarket()
	assert.Equal(t, existingMarket.ID, updatedMarket.ID)
	assert.Equal(t, existingMarket.TradableInstrument.Instrument.Name, updatedMarket.TradableInstrument.Instrument.Name)
	assert.Equal(t, existingMarket.TradableInstrument.Instrument.Product.(*types.InstrumentFuture).Future.SettlementAsset, updatedMarket.TradableInstrument.Instrument.Product.(*types.InstrumentFuture).Future.SettlementAsset)
	assert.Equal(t, existingMarket.DecimalPlaces, updatedMarket.DecimalPlaces)
	assert.Equal(t, existingMarket.PositionDecimalPlaces, updatedMarket.PositionDecimalPlaces)
	assert.Equal(t, existingMarket.OpeningAuction.Duration, updatedMarket.OpeningAuction.Duration)
}

func testRejectingProposalForSpotMarketUpdateSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("market-1", party, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.ensureExistingMarket(t, marketID)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, party, 0.7)
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

func testVotingWithoutMinimumTokenHoldersAndEquityLikeShareMakesSpotMarketUpdateProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("״market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipation, "0.5")
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipationLP, "0.5")
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, proposer, 0.1)
	eng.ensureExistingMarket(t, marketID)
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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS, 0.1)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS, 0.1)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorParticipationThresholdNotReached)
	eng.expectVoteEvents(t)
	eng.expectGetMarketState(t, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterClosing)
}

func testVotingWithMajorityOfYesFromTokenHoldersMakesSpotMarketUpdateProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("״market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, proposer, 0.7)
	eng.ensureExistingMarket(t, marketID)
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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken1, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken2, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS1, 0.1)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS2, 0.7)

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
	eng.expectGetMarketState(t, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterClosing)
}

func testVotingWithMajorityOfNoFromTokenHoldersMakesSpotMarketUpdateProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, proposer, 0.7)
	eng.ensureExistingMarket(t, marketID)
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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken1, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken2, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS1, 0.1)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS2, 0.7)

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
	eng.expectGetMarketState(t, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterClosing)
}

func testVotingWithoutTokenAndMajorityOfYesFromEquityLikeShareHoldersMakesSpotMarketUpdateProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipation, "0.5")

	// Submit proposal.
	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, proposer, 0.7)
	eng.ensureExistingMarket(t, marketID)
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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS1, 0.1)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS2, 0.7)

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
	eng.ensureTokenBalanceForParty(t, voterWithToken, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS1, 0.1)
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS2, 0.7)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectVoteEvents(t)
	eng.expectGetMarketState(t, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterClosing)
}

func testVotingWithoutTokenAndMajorityOfNoFromEquityLikeShareHoldersMakesSpotMarketUpdateProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// Submit proposal.
	// given

	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipation, "0.5")

	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForMarketUpdate("market-1", proposer, eng.tsvc.GetTimeNow(), nil, nil, true)
	marketID := proposal.MarketUpdate().MarketID

	// setup
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, proposer, 0.7)
	eng.ensureExistingMarket(t, marketID)
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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken, 0)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS1, 0.1)

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
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS2, 0.7)

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
	eng.ensureTokenBalanceForParty(t, voterWithToken, 2)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithToken, 0)
	eng.ensureTokenBalanceForParty(t, voterWithELS1, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS1, 0.1)
	eng.ensureTokenBalanceForParty(t, voterWithELS2, 0)
	eng.ensureEquityLikeShareForMarketAndParty(t, marketID, voterWithELS2, 0.7)

	// ensure setting again the values have no effect
	eng.ensureNetworkParameter(t, netparams.GovernanceProposalUpdateMarketRequiredParticipation, "0")

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorMajorityThresholdNotReached)
	eng.expectVoteEvents(t)
	eng.expectGetMarketState(t, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterClosing)
}
