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
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/governance/mocks"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errNoBalanceForParty = errors.New("no balance for party")

type tstEngine struct {
	*governance.Engine
	ctrl            *gomock.Controller
	accounts        *mocks.MockStakingAccounts
	tsvc            *mocks.MockTimeService
	broker          *bmocks.MockBroker
	witness         *mocks.MockWitness
	markets         *mocks.MockMarkets
	assets          *mocks.MockAssets
	netp            *netparams.Store
	proposalCounter uint                          // to streamline proposal generation
	tokenBal        map[string]uint64             // party > balance
	els             map[string]map[string]float64 // market > party > ELS
}

func TestSubmitProposals(t *testing.T) {
	t.Run("Submitting a proposal with closing time too soon fails", testSubmittingProposalWithClosingTimeTooSoonFails)
	t.Run("Submitting a proposal with internal time termination with closing time too soon fails", testSubmittingProposalWithInternalTimeTerminationWithClosingTimeTooSoonFails)
	t.Run("Submitting a proposal with closing time too late fails", testSubmittingProposalWithClosingTimeTooLateFails)
	t.Run("Submitting a proposal with internal time termination with closing time too late fails", testSubmittingProposalWithInternalTimeTerminationWithClosingTimeTooLateFails)
	t.Run("Submitting a proposal with enactment time too soon fails", testSubmittingProposalWithEnactmentTimeTooSoonFails)
	t.Run("Submitting a proposal with enactment time too soon fails", testSubmittingProposalWithEnactmentTimeTooSoonFails)
	t.Run("Submitting a proposal with enactment time too late fails", testSubmittingProposalWithEnactmentTimeTooLateFails)
	t.Run("Submitting a proposal with non-existing account fails", testSubmittingProposalWithNonExistingAccountFails)
	t.Run("Submitting a proposal with internal time termination with non-existing account fails", testSubmittingProposalWithInternalTimeTerminationWithNonExistingAccountFails)
	t.Run("Submitting a proposal without enough stake fails", testSubmittingProposalWithoutEnoughStakeFails)
	t.Run("Submitting a proposal with internal time termination without enough stake fails", testSubmittingProposalWithInternalTimeTerminationWithoutEnoughStakeFails)

	t.Run("Submitting a time-triggered proposal for new market with termination time before enactment time fails", testSumittingTimeTriggeredProposalNewMarketTerminationBeforeEnactmentFails)

	t.Run("Voting on non-existing proposal fails", testVotingOnNonExistingProposalFails)
	t.Run("Voting with non-existing account fails", testVotingWithNonExistingAccountFails)
	t.Run("Voting without token fails", testVotingWithoutTokenFails)

	t.Run("Test multiple proposal lifecycle", testMultipleProposalsLifecycle)
	t.Run("Withdrawing vote assets removes vote from proposal state calculation", testWithdrawingVoteAssetRemovesVoteFromProposalStateCalculation)

	t.Run("Updating voters key on votes succeeds", testUpdatingVotersKeyOnVotesSucceeds)
	t.Run("Updating voters key on votes with internal time termination succeeds", testUpdatingVotersKeyOnVotesWithInternalTimeTerminationSucceeds)

	t.Run("Computing the governance state hash is deterministic", testComputingGovernanceStateHashIsDeterministic)
}

func testUpdatingVotersKeyOnVotesSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)

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
	voter1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 1)

	// expect
	eng.expectVoteEvent(t, voter1, proposal.ID)

	// when
	err = eng.addYesVote(t, voter1, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	voter2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter2, 1)

	// expect
	eng.expectVoteEvent(t, voter2, proposal.ID)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	newVoter1ID := vgrand.RandomStr(5)

	// expect
	eng.expectVoteEvent(t, newVoter1ID, proposal.ID)

	// then
	eng.ValidatorKeyChanged(context.Background(), voter1, newVoter1ID)

	// given
	newVoter2ID := vgrand.RandomStr(5)

	// setup
	eng.expectVoteEvent(t, newVoter2ID, proposal.ID)

	// then
	eng.ValidatorKeyChanged(context.Background(), voter2, newVoter2ID)
}

func testUpdatingVotersKeyOnVotesWithInternalTimeTerminationSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, false)

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
	voter1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 1)

	// expect
	eng.expectVoteEvent(t, voter1, proposal.ID)

	// when
	err = eng.addYesVote(t, voter1, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	voter2 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter2, 1)

	// expect
	eng.expectVoteEvent(t, voter2, proposal.ID)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	newVoter1ID := vgrand.RandomStr(5)

	// expect
	eng.expectVoteEvent(t, newVoter1ID, proposal.ID)

	// then
	eng.ValidatorKeyChanged(context.Background(), voter1, newVoter1ID)

	// given
	newVoter2ID := vgrand.RandomStr(5)

	// setup
	eng.expectVoteEvent(t, newVoter2ID, proposal.ID)

	// then
	eng.ValidatorKeyChanged(context.Background(), voter2, newVoter2ID)
}

func testSubmittingProposalWithNonExistingAccountFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	now := eng.tsvc.GetTimeNow()

	tcs := []struct {
		name     string
		proposal types.Proposal
	}{
		{
			name:     "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, true),
		}, {
			name:     "For market update",
			proposal: eng.newProposalForMarketUpdate("", party, now, nil, nil, true),
		}, {
			name:     "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		}, {
			name:     "Freeform",
			proposal: eng.newFreeformProposal(party, now),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// setup
			eng.ensureAllAssetEnabled(tt)
			eng.ensureNoAccountForParty(tt, party)
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorInsufficientTokens)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.EqualError(tt, err, errNoBalanceForParty.Error())
		})
	}
}

func testSubmittingProposalWithInternalTimeTerminationWithNonExistingAccountFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	now := eng.tsvc.GetTimeNow()

	tcs := []struct {
		name     string
		proposal types.Proposal
	}{
		{
			name:     "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, false),
		}, {
			name:     "For market update",
			proposal: eng.newProposalForMarketUpdate("", party, now, nil, nil, false),
		}, {
			name:     "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		}, {
			name:     "Freeform",
			proposal: eng.newFreeformProposal(party, now),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// setup
			eng.ensureAllAssetEnabled(tt)
			eng.ensureNoAccountForParty(tt, party)
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorInsufficientTokens)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.EqualError(tt, err, errNoBalanceForParty.Error())
		})
	}
}

func testSubmittingProposalWithoutEnoughStakeFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	now := eng.tsvc.GetTimeNow()

	tcs := []struct {
		name                    string
		minProposerBalanceParam string
		proposal                types.Proposal
	}{
		{
			name:                    "For new market",
			minProposerBalanceParam: netparams.GovernanceProposalMarketMinProposerBalance,
			proposal:                eng.newProposalForNewMarket(party, now, nil, nil, true),
		}, {
			name:                    "For market update",
			minProposerBalanceParam: netparams.GovernanceProposalUpdateMarketMinProposerBalance,
			proposal:                eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, true),
		}, {
			name:                    "For new asset",
			minProposerBalanceParam: netparams.GovernanceProposalAssetMinProposerBalance,
			proposal:                eng.newProposalForNewAsset(party, now),
		}, {
			name:                    "Freeform",
			minProposerBalanceParam: netparams.GovernanceProposalFreeformMinProposerBalance,
			proposal:                eng.newFreeformProposal(party, now),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// setup
			eng.ensureTokenBalanceForParty(tt, party, 10)
			eng.ensureNetworkParameter(tt, tc.minProposerBalanceParam, "10000")
			eng.ensureAllAssetEnabled(tt)
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorInsufficientTokens)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(t, err.Error(), "proposer have insufficient governance token, expected >=")
		})
	}
}

func testSubmittingProposalWithInternalTimeTerminationWithoutEnoughStakeFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	now := eng.tsvc.GetTimeNow()

	tcs := []struct {
		name                    string
		minProposerBalanceParam string
		proposal                types.Proposal
	}{
		{
			name:                    "For new market",
			minProposerBalanceParam: netparams.GovernanceProposalMarketMinProposerBalance,
			proposal:                eng.newProposalForNewMarket(party, now, nil, nil, false),
		}, {
			name:                    "For market update",
			minProposerBalanceParam: netparams.GovernanceProposalUpdateMarketMinProposerBalance,
			proposal:                eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, false),
		}, {
			name:                    "For new asset",
			minProposerBalanceParam: netparams.GovernanceProposalAssetMinProposerBalance,
			proposal:                eng.newProposalForNewAsset(party, now),
		}, {
			name:                    "Freeform",
			minProposerBalanceParam: netparams.GovernanceProposalFreeformMinProposerBalance,
			proposal:                eng.newFreeformProposal(party, now),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// setup
			eng.ensureTokenBalanceForParty(tt, party, 10)
			eng.ensureNetworkParameter(tt, tc.minProposerBalanceParam, "10000")
			eng.ensureAllAssetEnabled(tt)
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorInsufficientTokens)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(t, err.Error(), "proposer have insufficient governance token, expected >=")
		})
	}
}

func testSubmittingProposalWithClosingTimeTooSoonFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := eng.tsvc.GetTimeNow()
	party := vgrand.RandomStr(5)

	cases := []struct {
		msg      string
		proposal types.Proposal
	}{
		{
			msg:      "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, true),
		}, {
			msg:      "For market update",
			proposal: eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, true),
		}, {
			msg:      "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// given
			tc.proposal.Terms.ClosingTimestamp = now.Unix()

			// setup
			eng.ensureAllAssetEnabled(tt)

			// expect
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorCloseTimeTooSoon)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal closing time too soon, expected >")
		})
	}
}

func testSubmittingProposalWithInternalTimeTerminationWithClosingTimeTooSoonFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := eng.tsvc.GetTimeNow()
	party := vgrand.RandomStr(5)

	cases := []struct {
		msg      string
		proposal types.Proposal
	}{
		{
			msg:      "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, false),
		}, {
			msg:      "For market update",
			proposal: eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, false),
		}, {
			msg:      "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// given
			tc.proposal.Terms.ClosingTimestamp = now.Unix()

			// setup
			eng.ensureAllAssetEnabled(tt)

			// expect
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorCloseTimeTooSoon)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal closing time too soon, expected >")
		})
	}
}

func testSubmittingProposalWithClosingTimeTooLateFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := eng.tsvc.GetTimeNow()
	party := vgrand.RandomStr(5)

	cases := []struct {
		msg      string
		proposal types.Proposal
	}{
		{
			msg:      "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, true),
		}, {
			msg:      "For market update",
			proposal: eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, true),
		}, {
			msg:      "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// given
			tc.proposal.Terms.ClosingTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()

			// setup
			eng.ensureAllAssetEnabled(tt)

			// expect
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorCloseTimeTooLate)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal closing time too late, expected <")
		})
	}
}

func testSubmittingProposalWithInternalTimeTerminationWithClosingTimeTooLateFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := eng.tsvc.GetTimeNow()
	party := vgrand.RandomStr(5)

	cases := []struct {
		msg      string
		proposal types.Proposal
	}{
		{
			msg:      "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, false),
		}, {
			msg:      "For market update",
			proposal: eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, false),
		}, {
			msg:      "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// given
			tc.proposal.Terms.ClosingTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()

			// setup
			eng.ensureAllAssetEnabled(tt)

			// expect
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorCloseTimeTooLate)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal closing time too late, expected <")
		})
	}
}

func testSumittingTimeTriggeredProposalNewMarketTerminationBeforeEnactmentFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	proposer := vgrand.RandomStr(5)
	now := eng.tsvc.GetTimeNow()

	// Make a proporsal with termination time before enactment time
	// Enactment time for new market is now + 96 hours
	termTimeBeforeEnact := now.Add(1 * 48 * time.Hour).Add(47 * time.Hour).Add(15 * time.Minute)

	filter, binding := produceTimeTriggeredDataSourceSpec(termTimeBeforeEnact)
	proposal1 := eng.newProposalForNewMarket(proposer, now, filter, binding, true)

	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureAllAssetEnabled(t)

	eng.expectRejectedProposalEvent(t, proposer, proposal1.ID, types.ProposalErrorInvalidFutureProduct)
	_, err := eng.submitProposal(t, proposal1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data source spec termination time before enactment")

	// Make another proposal with termination time same as enactment time
	// Enactment time for new market is now + 96 hours
	termTimeEqualEnact := now.Add(2 * 48 * time.Hour)
	filter, binding = produceTimeTriggeredDataSourceSpec(termTimeEqualEnact)
	proposal2 := eng.newProposalForNewMarket(proposer, now, filter, binding, true)

	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureAllAssetEnabled(t)
	eng.expectRejectedProposalEvent(t, proposer, proposal2.ID, types.ProposalErrorInvalidFutureProduct)

	_, err = eng.submitProposal(t, proposal2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data source spec termination time before enactment")
}

func testSubmittingProposalWithEnactmentTimeTooSoonFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := eng.tsvc.GetTimeNow()
	party := vgrand.RandomStr(5)

	cases := []struct {
		msg      string
		proposal types.Proposal
	}{
		{
			msg:      "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, true),
		}, {
			msg:      "For market update",
			proposal: eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, true),
		}, {
			msg:      "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// given
			tc.proposal.Terms.EnactmentTimestamp = now.Unix()

			// setup
			eng.ensureAllAssetEnabled(tt)
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorEnactTimeTooSoon)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal enactment time too soon, expected >")
		})
	}
}

func testSubmittingProposalWithEnactmentTimeTooLateFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := eng.tsvc.GetTimeNow()
	party := vgrand.RandomStr(5)

	cases := []struct {
		msg      string
		proposal types.Proposal
	}{
		{
			msg:      "For new market",
			proposal: eng.newProposalForNewMarket(party, now, nil, nil, true),
		}, {
			msg:      "For market update",
			proposal: eng.newProposalForMarketUpdate("market-1", party, now, nil, nil, true),
		}, {
			msg:      "For new asset",
			proposal: eng.newProposalForNewAsset(party, now),
		},
	}

	for _, tc := range cases {
		t.Run(tc.msg, func(tt *testing.T) {
			// given
			tc.proposal.Terms.EnactmentTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()

			// setup
			eng.ensureAllAssetEnabled(tt)

			// expect
			eng.expectRejectedProposalEvent(tt, party, tc.proposal.ID, types.ProposalErrorEnactTimeTooLate)

			// when
			_, err := eng.submitProposal(tt, tc.proposal)

			// then
			require.Error(tt, err)
			assert.Contains(tt, err.Error(), "proposal enactment time too late, expected <")
		})
	}
}

func testVotingOnNonExistingProposalFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	voter := vgrand.RandomStr(5)

	// setup
	eng.ensureAllAssetEnabled(t)

	// when
	err := eng.addYesVote(t, voter, vgrand.RandomStr(5))

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalDoesNotExist.Error())
}

func testVotingWithNonExistingAccountFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)

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
	voterWithoutAccount := "voter-no-account"

	// setup
	eng.ensureNoAccountForParty(t, voterWithoutAccount)

	// when
	err = eng.addYesVote(t, voterWithoutAccount, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, errNoBalanceForParty.Error())
}

func testVotingWithoutTokenFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := eng.newValidParty("proposer", 1)
	proposal := eng.newProposalForNewMarket(proposer.Id, eng.tsvc.GetTimeNow(), nil, nil, true)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voterWithEmptyAccount := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voterWithEmptyAccount, 0)

	// when
	err = eng.addYesVote(t, voterWithEmptyAccount, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrVoterInsufficientTokens.Error())
}

func testMultipleProposalsLifecycle(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	now := eng.tsvc.GetTimeNow()
	partyA := vgrand.RandomStr(5)
	partyB := vgrand.RandomStr(5)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().AnyTimes().Return(num.NewUint(300))
	eng.tokenBal[partyA] = 200
	eng.tokenBal[partyB] = 100

	const howMany = 10
	passed := map[string]*types.Proposal{}
	declined := map[string]*types.Proposal{}
	var afterClosing time.Time
	var afterEnactment time.Time

	for i := 0; i < howMany; i++ {
		toBePassed := eng.newProposalForNewMarket(partyA, now, nil, nil, true)
		eng.expectOpenProposalEvent(t, partyA, toBePassed.ID)
		_, err := eng.submitProposal(t, toBePassed)
		require.NoError(t, err)
		passed[toBePassed.ID] = &toBePassed

		toBeDeclined := eng.newProposalForNewMarket(partyB, now, nil, nil, true)
		eng.expectOpenProposalEvent(t, partyB, toBeDeclined.ID)
		_, err = eng.submitProposal(t, toBeDeclined)
		require.NoError(t, err)
		declined[toBeDeclined.ID] = &toBeDeclined

		if i == 0 {
			// all proposal terms are expected to be equal
			afterClosing = time.Unix(toBePassed.Terms.ClosingTimestamp, 0).Add(time.Second)
			afterEnactment = time.Unix(toBePassed.Terms.EnactmentTimestamp, 0).Add(time.Second)
		}
	}
	require.Len(t, passed, howMany)
	require.Len(t, declined, howMany)

	for id := range passed {
		eng.expectVoteEvent(t, partyA, id)
		err := eng.addYesVote(t, partyA, id)
		require.NoError(t, err)

		eng.expectVoteEvent(t, partyB, id)
		err = eng.addNoVote(t, partyB, id)
		require.NoError(t, err)
	}

	for id := range declined {
		eng.expectVoteEvent(t, partyA, id)
		err := eng.addNoVote(t, partyA, id)
		require.NoError(t, err)

		eng.expectVoteEvent(t, partyB, id)
		err = eng.addYesVote(t, partyB, id)
		require.NoError(t, err)
	}

	var howManyPassed, howManyDeclined int
	eng.markets.EXPECT().GetMarketState(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).Times(howMany * 2).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		require.True(t, ok)
		p := pe.Proposal()
		if p.State == types.ProposalStatePassed {
			_, found := passed[p.Id]
			assert.True(t, found, "passed proposal is in the passed collection")
			howManyPassed++
		} else if p.State == types.ProposalStateDeclined {
			_, found := declined[p.Id]
			assert.True(t, found, "declined proposal is in the declined collection")
			howManyDeclined++
		} else {
			assert.FailNow(t, "unexpected proposal state")
		}
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(howMany * 2)
	eng.OnTick(context.Background(), afterClosing)
	assert.Equal(t, howMany, howManyPassed)
	assert.Equal(t, howMany, howManyDeclined)

	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)
	require.Len(t, toBeEnacted, howMany)
	for i := 0; i < howMany; i++ {
		_, found := passed[toBeEnacted[i].Proposal().ID]
		assert.True(t, found)
	}
}

func testWithdrawingVoteAssetRemovesVoteFromProposalStateCalculation(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewMarket(proposer, eng.tsvc.GetTimeNow(), nil, nil, true)

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

	// expect
	eng.expectVoteEvent(t, voter, proposal.ID)

	// when
	err = eng.addNoVote(t, voter, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter, 0)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorParticipationThresholdNotReached)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "0", "0")

	// when
	eng.expectGetMarketState(t, proposal.ID)
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

func testComputingGovernanceStateHashIsDeterministic(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	require.Equal(t,
		"a1292c11ccdb876535c6699e8217e1a1294190d83e4233ecc490d32df17a4116",
		hex.EncodeToString(eng.Hash()),
		"hash is not deterministic",
	)

	// when
	proposer := vgrand.RandomStr(5)
	now := eng.tsvc.GetTimeNow()
	proposal := eng.newProposalForNewMarket(proposer, now, nil, nil, true)

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1)
	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)

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
	// test hash before enactment
	require.Equal(t,
		"d43f721a8e28c5bad0e78ab7052b8990be753044bb355056519fab76e8de50a7",
		hex.EncodeToString(eng.Hash()),
		"hash is not deterministic",
	)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")

	// when
	eng.expectGetMarketState(t, proposal.ID)
	eng.OnTick(context.Background(), afterClosing)

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	eng.expectGetMarketState(t, proposal.ID)
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	require.Len(t, toBeEnacted, 1)
	require.Equal(t,
		"fbf86f159b135501153cda0fc333751df764290a3ae61c3f45f19f9c19445563",
		hex.EncodeToString(eng.Hash()),
		"hash is not deterministic",
	)
}

func getTestEngine(t *testing.T) *tstEngine {
	t.Helper()

	cfg := governance.NewDefaultConfig()
	log := logging.NewTestLogger()

	ctrl := gomock.NewController(t)
	accounts := mocks.NewMockStakingAccounts(ctrl)
	markets := mocks.NewMockMarkets(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	witness := mocks.NewMockWitness(ctrl)

	// Set default network parameters
	netp := netparams.New(log, netparams.NewDefaultConfig(), broker)

	ctx := context.Background()

	ts.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(1659006535, 0)
		}).AnyTimes()

	broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.GovernanceProposalMarketMinVoterBalance, "1")).Times(1)
	require.NoError(t, netp.Update(ctx, netparams.GovernanceProposalMarketMinVoterBalance, "1"))

	broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.GovernanceProposalMarketRequiredParticipation, "0.5")).Times(1)
	require.NoError(t, netp.Update(ctx, netparams.GovernanceProposalMarketRequiredParticipation, "0.5"))

	broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.GovernanceProposalUpdateMarketMinProposerEquityLikeShare, "0.1")).Times(1)
	require.NoError(t, netp.Update(ctx, netparams.GovernanceProposalUpdateMarketMinProposerEquityLikeShare, "0.1"))

	// Initialise engine as validator
	eng := governance.NewEngine(log, cfg, accounts, ts, broker, assets, witness, markets, netp)
	require.NotNil(t, eng)

	tEng := &tstEngine{
		Engine:   eng,
		ctrl:     ctrl,
		accounts: accounts,
		markets:  markets,
		tsvc:     ts,
		broker:   broker,
		assets:   assets,
		witness:  witness,
		netp:     netp,
		tokenBal: map[string]uint64{},
		els:      map[string]map[string]float64{},
	}
	// ensure the balance is always returned as expected
	tEng.accounts.EXPECT().GetAvailableBalance(gomock.Any()).AnyTimes().DoAndReturn(func(p string) (*num.Uint, error) {
		b, ok := tEng.tokenBal[p]
		if !ok {
			return nil, errNoBalanceForParty
		}
		return num.NewUint(b), nil
	})
	return tEng
}

func newFreeformTerms() *types.ProposalTermsNewFreeform {
	return &types.ProposalTermsNewFreeform{
		NewFreeform: &types.NewFreeform{},
	}
}

func newAssetTerms() *types.ProposalTermsNewAsset {
	return &types.ProposalTermsNewAsset{
		NewAsset: &types.NewAsset{
			Changes: &types.AssetDetails{
				Name:     "token",
				Symbol:   "TKN",
				Decimals: 18,
				Quantum:  num.DecimalFromFloat(1),
				Source: &types.AssetDetailsBuiltinAsset{
					BuiltinAsset: &types.BuiltinAsset{
						MaxFaucetAmountMint: num.NewUint(1),
					},
				},
			},
		},
	}
}

func produceNonTimeTriggeredDataSourceSpec() (*types.DataSourceSpecFilter, *types.DataSourceSpecBindingForFuture) {
	return &types.DataSourceSpecFilter{
			Key: &types.DataSourceSpecPropertyKey{
				Name: "trading.terminated",
				Type: datapb.PropertyKey_TYPE_BOOLEAN,
			},
			Conditions: []*types.DataSourceSpecCondition{},
		},
		&types.DataSourceSpecBindingForFuture{
			SettlementDataProperty:     "prices.ETH.value",
			TradingTerminationProperty: "trading.terminated",
		}
}

func produceTimeTriggeredDataSourceSpec(termTimestamp time.Time) (*types.DataSourceSpecFilter, *types.DataSourceSpecBindingForFuture) {
	return &types.DataSourceSpecFilter{
			Key: &types.DataSourceSpecPropertyKey{
				Name: "vegaprotocol.builtin.timestamp",
				Type: datapb.PropertyKey_TYPE_TIMESTAMP,
			},
			Conditions: []*types.DataSourceSpecCondition{
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
					Value:    strconv.FormatInt(termTimestamp.Unix(), 10),
				},
			},
		},
		&types.DataSourceSpecBindingForFuture{
			SettlementDataProperty:     "prices.ETH.value",
			TradingTerminationProperty: "vegaprotocol.builtin.timestamp",
		}
}

func newNetParamTerms(key, value string) *types.ProposalTermsUpdateNetworkParameter {
	return &types.ProposalTermsUpdateNetworkParameter{
		UpdateNetworkParameter: &types.UpdateNetworkParameter{
			Changes: &types.NetworkParameter{
				Key:   key,
				Value: value,
			},
		},
	}
}

func newMarketTerms(termFilter *types.DataSourceSpecFilter, termBinding *types.DataSourceSpecBindingForFuture, termExt bool, successor *types.SuccessorConfig) *types.ProposalTermsNewMarket {
	var dt *types.DataSourceDefinition
	if termExt {
		if termFilter == nil {
			termFilter, termBinding = produceNonTimeTriggeredDataSourceSpec()
		}

		dt = types.NewDataSourceDefinition(vegapb.DataSourceDefinitionTypeExt).SetOracleConfig(
			&types.DataSourceDefinitionExternal{
				SourceType: &types.DataSourceDefinitionExternalOracle{
					Oracle: &types.DataSourceSpecConfiguration{
						Signers: []*types.Signer{types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey)},
						Filters: []*types.DataSourceSpecFilter{
							termFilter,
						},
					},
				},
			},
		)
	} else {
		tm := time.Now().Add(time.Hour * 24 * 365)
		if termFilter == nil {
			_, termBinding = produceTimeTriggeredDataSourceSpec(tm)
		}

		dt = types.NewDataSourceDefinition(vegapb.DataSourceDefinitionTypeInt).SetTimeTriggerConditionConfig(
			[]*types.DataSourceSpecCondition{
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
					Value:    fmt.Sprintf("%d", tm.UnixNano()),
				},
			},
		)
	}

	return &types.ProposalTermsNewMarket{
		NewMarket: &types.NewMarket{
			Changes: &types.NewMarketConfiguration{
				Instrument: &types.InstrumentConfiguration{
					Name: "June 2020 GBP vs VUSD future",
					Code: "CRYPTO:GBPVUSD/JUN20",
					Product: &types.InstrumentConfigurationFuture{
						Future: &types.FutureProduct{
							SettlementAsset: "VUSD",
							QuoteName:       "VUSD",
							DataSourceSpecForSettlementData: *types.NewDataSourceDefinition(
								vegapb.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&types.DataSourceDefinitionExternal{
									SourceType: &types.DataSourceDefinitionExternalOracle{
										Oracle: &types.DataSourceSpecConfiguration{
											Signers: []*types.Signer{types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey)},
											Filters: []*types.DataSourceSpecFilter{
												{
													Key: &types.DataSourceSpecPropertyKey{
														Name: "prices.ETH.value",
														Type: datapb.PropertyKey_TYPE_INTEGER,
													},
													Conditions: []*types.DataSourceSpecCondition{
														{
															Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															Value:    "0",
														},
													},
												},
											},
										},
									},
								},
							),
							DataSourceSpecForTradingTermination: *dt,
							DataSourceSpecBinding:               termBinding,
						},
					},
				},
				RiskParameters: &types.NewMarketConfigurationLogNormal{
					LogNormal: &types.LogNormalRiskModel{
						RiskAversionParameter: num.DecimalFromFloat(0.01),
						Tau:                   num.DecimalFromFloat(0.00011407711613050422),
						Params: &types.LogNormalModelParams{
							Mu:    num.DecimalZero(),
							R:     num.DecimalFromFloat(0.016),
							Sigma: num.DecimalFromFloat(0.09),
						},
					},
				},
				Metadata:                []string{"asset_class:fx/crypto", "product:futures"},
				DecimalPlaces:           0,
				LpPriceRange:            num.DecimalFromFloat(0.95),
				LinearSlippageFactor:    num.DecimalFromFloat(0.1),
				QuadraticSlippageFactor: num.DecimalFromFloat(0.1),
				Successor:               successor,
			},
		},
	}
}

func newSpotMarketTerms() *types.ProposalTermsNewSpotMarket {
	return &types.ProposalTermsNewSpotMarket{
		NewSpotMarket: &types.NewSpotMarket{
			Changes: &types.NewSpotMarketConfiguration{
				Instrument: &types.InstrumentConfiguration{
					Name: "BTC/USDT Spot",
					Code: "CRYPTO:BTCUSDT",
					Product: &types.InstrumentConfigurationSpot{
						Spot: &types.SpotProduct{
							Name:       "BTC/USDT",
							BaseAsset:  "BTC",
							QuoteAsset: "USDT",
						},
					},
				},
				RiskParameters: &types.NewMarketConfigurationLogNormal{
					LogNormal: &types.LogNormalRiskModel{
						RiskAversionParameter: num.DecimalFromFloat(0.01),
						Tau:                   num.DecimalFromFloat(0.00011407711613050422),
						Params: &types.LogNormalModelParams{
							Mu:    num.DecimalZero(),
							R:     num.DecimalFromFloat(0.016),
							Sigma: num.DecimalFromFloat(0.09),
						},
					},
				},
				Metadata:      []string{"asset_class:spot/crypto", "product:spot"},
				DecimalPlaces: 0,
			},
		},
	}
}

func updateMarketTerms(termFilter *types.DataSourceSpecFilter, termBinding *types.DataSourceSpecBindingForFuture, termExt bool) *types.ProposalTermsUpdateMarket {
	var dt *types.DataSourceDefinition
	if termExt {
		if termFilter == nil {
			termFilter = &types.DataSourceSpecFilter{
				Key: &types.DataSourceSpecPropertyKey{
					Name: "trading.terminated",
					Type: datapb.PropertyKey_TYPE_BOOLEAN,
				},
				Conditions: []*types.DataSourceSpecCondition{},
			}

			termBinding = &types.DataSourceSpecBindingForFuture{
				SettlementDataProperty:     "prices.ETH.value",
				TradingTerminationProperty: "trading.terminated",
			}
		}
		dt = types.NewDataSourceDefinition(vegapb.DataSourceDefinitionTypeExt).SetOracleConfig(
			&types.DataSourceDefinitionExternal{
				SourceType: &types.DataSourceDefinitionExternalOracle{
					Oracle: &types.DataSourceSpecConfiguration{
						Signers: []*types.Signer{types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey)},
						Filters: []*types.DataSourceSpecFilter{
							termFilter,
						},
					},
				},
			},
		)
	} else {
		tm := time.Now().Add(time.Hour * 24 * 365)
		if termFilter == nil {
			_, termBinding = produceTimeTriggeredDataSourceSpec(tm)
		}

		dt = types.NewDataSourceDefinition(vegapb.DataSourceDefinitionTypeInt).SetTimeTriggerConditionConfig(
			[]*types.DataSourceSpecCondition{
				{
					Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
					Value:    fmt.Sprintf("%d", tm.UnixNano()),
				},
			},
		)
	}

	return &types.ProposalTermsUpdateMarket{
		UpdateMarket: &types.UpdateMarket{
			MarketID: vgrand.RandomStr(5),
			Changes: &types.UpdateMarketConfiguration{
				Instrument: &types.UpdateInstrumentConfiguration{
					Code: "CRYPTO:GBPVUSD/JUN20",
					Product: &types.UpdateInstrumentConfigurationFuture{
						Future: &types.UpdateFutureProduct{
							QuoteName: "VUSD",
							DataSourceSpecForSettlementData: *types.NewDataSourceDefinition(
								vegapb.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&types.DataSourceDefinitionExternal{
									SourceType: &types.DataSourceDefinitionExternalOracle{
										Oracle: &types.DataSourceSpecConfiguration{
											Signers: []*types.Signer{types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey)},
											Filters: []*types.DataSourceSpecFilter{
												{
													Key: &types.DataSourceSpecPropertyKey{
														Name: "prices.ETH.value",
														Type: datapb.PropertyKey_TYPE_INTEGER,
													},
													Conditions: []*types.DataSourceSpecCondition{},
												},
											},
										},
									},
								},
							),
							DataSourceSpecForTradingTermination: *dt,
							DataSourceSpecBinding:               termBinding,
						},
					},
				},
				RiskParameters: &types.UpdateMarketConfigurationLogNormal{
					LogNormal: &types.LogNormalRiskModel{
						RiskAversionParameter: num.DecimalFromFloat(0.01),
						Tau:                   num.DecimalFromFloat(0.00011407711613050422),
						Params: &types.LogNormalModelParams{
							Mu:    num.DecimalZero(),
							R:     num.DecimalFromFloat(0.016),
							Sigma: num.DecimalFromFloat(0.09),
						},
					},
				},
				Metadata:                []string{"asset_class:fx/crypto", "product:futures"},
				LpPriceRange:            num.DecimalFromFloat(0.95),
				LinearSlippageFactor:    num.DecimalFromFloat(0.1),
				QuadraticSlippageFactor: num.DecimalFromFloat(0.1),
			},
		},
	}
}

func (e *tstEngine) submitProposal(t *testing.T, proposal types.Proposal) (*governance.ToSubmit, error) {
	t.Helper()
	return e.SubmitProposal(
		context.Background(),
		*types.ProposalSubmissionFromProposal(&proposal),
		proposal.ID,
		proposal.Party,
	)
}

func (e *tstEngine) addYesVote(t *testing.T, party, proposal string) error {
	t.Helper()
	return e.AddVote(context.Background(), types.VoteSubmission{
		ProposalID: proposal,
		Value:      types.VoteValueYes,
	}, party)
}

func (e *tstEngine) addNoVote(t *testing.T, party, proposal string) error {
	t.Helper()
	return e.AddVote(context.Background(), types.VoteSubmission{
		ProposalID: proposal,
		Value:      types.VoteValueNo,
	}, party)
}

func (e *tstEngine) newValidPartyTimes(partyID string, balance uint64, _ int) *types.Party {
	// is called with 0 times, which messes up the expected calls when adding votes
	e.tokenBal[partyID] = balance
	return &types.Party{Id: partyID}
}

func (e *tstEngine) newValidParty(partyID string, balance uint64) *types.Party {
	return e.newValidPartyTimes(partyID, balance, 1)
}

func (e *tstEngine) newProposalID() string {
	e.proposalCounter++
	return fmt.Sprintf("proposal-id-%d", e.proposalCounter)
}

func (e *tstEngine) newProposalForNewMarket(partyID string, now time.Time, termFilter *types.DataSourceSpecFilter, termBinding *types.DataSourceSpecBindingForFuture, termExt bool) types.Proposal {
	id := e.newProposalID()
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newMarketTerms(termFilter, termBinding, termExt, nil),
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}
}

func (e *tstEngine) newProposalForSuccessorMarket(partyID string, now time.Time, termFilter *types.DataSourceSpecFilter, termBinding *types.DataSourceSpecBindingForFuture, termExt bool, successor *types.SuccessorConfig) types.Proposal {
	id := e.newProposalID()
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newMarketTerms(termFilter, termBinding, termExt, successor),
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}
}

func (e *tstEngine) newProposalForNewSpotMarket(partyID string, now time.Time) types.Proposal {
	id := e.newProposalID()
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newSpotMarketTerms(),
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}
}

func (e *tstEngine) newProposalForMarketUpdate(marketID, partyID string, now time.Time, termFilter *types.DataSourceSpecFilter, termBinding *types.DataSourceSpecBindingForFuture, termExt bool) types.Proposal {
	id := e.newProposalID()
	prop := types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(96 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(4 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(2 * time.Hour).Unix(),
			Change:              updateMarketTerms(termFilter, termBinding, termExt),
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}
	switch p := prop.Terms.Change.(type) {
	case *types.ProposalTermsUpdateMarket:
		p.UpdateMarket.MarketID = marketID
	}
	return prop
}

func (e *tstEngine) newProposalForNewAsset(partyID string, now time.Time) types.Proposal {
	id := e.newProposalID()
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newAssetTerms(),
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}
}

func (e *tstEngine) newFreeformProposal(partyID string, now time.Time) types.Proposal {
	id := e.newProposalID()
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newFreeformTerms(),
		},
		Rationale: &types.ProposalRationale{
			Title:       "https://example.com",
			Description: "Test my freeform proposal",
		},
	}
}

func (e *tstEngine) expectTotalGovernanceTokenFromVoteEvents(t *testing.T, weight, balance string) {
	t.Helper()
	e.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		require.True(t, ok)
		assert.Equal(t, weight, v.TotalGovernanceTokenWeight())
		assert.Equal(t, balance, v.TotalGovernanceTokenBalance())
	})
}

func (e *tstEngine) expectVoteEvents(t *testing.T) {
	t.Helper()
	e.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		_, ok := evts[0].(*events.Vote)
		require.True(t, ok)
	})
}

func (e *tstEngine) expectPassedProposalEvent(t *testing.T, proposal string) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		require.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.ProposalStatePassed.String(), p.State.String())
		assert.Equal(t, proposal, p.Id)
	})
}

func (e *tstEngine) expectDeclinedProposalEvent(t *testing.T, proposal string, reason types.ProposalError) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		require.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.ProposalStateDeclined.String(), p.State.String())
		assert.Equal(t, proposal, p.Id)
		assert.Equal(t, reason.String(), p.Reason.String())
	})
}

func (e *tstEngine) expectOpenProposalEvent(t *testing.T, party, proposal string) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(ev events.Event) {
		pe, ok := ev.(*events.Proposal)
		require.True(t, ok)
		p := pe.Proposal()
		reason := types.ProposalErrorUnspecified
		if p.Reason != nil {
			reason = *p.Reason
		}
		errDetails := ""
		if p.ErrorDetails != nil {
			errDetails = *p.ErrorDetails
		}
		assert.Equal(t, types.ProposalStateOpen.String(), p.State.String(), fmt.Sprintf("reason: %v, details: %s", reason, errDetails))
		assert.Equal(t, party, p.PartyId)
		assert.Equal(t, proposal, p.Id)
	})
}

func (e *tstEngine) expectProposalWaitingForNodeVoteEvent(t *testing.T, party, proposal string) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(ev events.Event) {
		pe, ok := ev.(*events.Proposal)
		require.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.ProposalStateWaitingForNodeVote.String(), p.State.String())
		assert.Equal(t, party, p.PartyId)
		assert.Equal(t, proposal, p.Id)
	})
}

func (e *tstEngine) expectRejectedProposalEvent(t *testing.T, partyID, proposalID string, reason types.ProposalError) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		require.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proposalID, p.Id)
		assert.Equal(t, partyID, p.PartyId)
		assert.Equal(t, types.ProposalStateRejected.String(), p.State.String())
		assert.Equal(t, reason.String(), p.Reason.String())
	})
}

func (e *tstEngine) expectVoteEvent(t *testing.T, party, proposal string) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		require.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, proposal, vote.ProposalId)
		assert.Equal(t, party, vote.PartyId)
	})
}

func (e *tstEngine) expectRestoredProposals(t *testing.T, proposalIDs []string) {
	t.Helper()
	e.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(es []events.Event) {
		if len(es) == 0 {
			t.Errorf("expecting %d proposals to be restored, found %d", len(proposalIDs), len(es))
		}

		for i, e := range es {
			pe, ok := e.(*events.Proposal)
			require.True(t, ok)
			assert.Equal(t, proposalIDs[i], pe.ProposalID())
		}
	})
}

func (e *tstEngine) ensureStakingAssetTotalSupply(t *testing.T, supply uint64) {
	t.Helper()
	e.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).Return(num.NewUint(supply))
}

func (e *tstEngine) ensureTokenBalanceForParty(t *testing.T, party string, balance uint64) {
	t.Helper()
	e.tokenBal[party] = balance
}

func (e *tstEngine) ensureAllAssetEnabled(t *testing.T) {
	t.Helper()
	details := newAssetTerms()
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(id string) (*assets.Asset, error) {
		ret := assets.NewAsset(builtin.New(id, details.NewAsset.Changes))
		return ret, nil
	})
	e.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
}

func (e *tstEngine) ensureEquityLikeShareForMarketAndParty(t *testing.T, market, party string, share float64) {
	t.Helper()
	mels, ok := e.els[market]
	if !ok {
		mels = map[string]float64{}
	}
	mels[party] = share
	e.els[market] = mels
	e.markets.EXPECT().
		GetEquityLikeShareForMarketAndParty(market, party).
		AnyTimes().
		Return(num.DecimalFromFloat(share), true)
}

func (e *tstEngine) ensureGetMarket(t *testing.T, marketID string, market types.Market) {
	t.Helper()
	e.markets.EXPECT().
		GetMarket(marketID, gomock.Any()).
		Times(1).
		Return(market, true)
}

func (e *tstEngine) expectGetMarketState(t *testing.T, marketID string) {
	t.Helper()
	e.markets.EXPECT().GetMarketState(marketID).AnyTimes().Return(types.MarketStateActive, nil)
}

func (e *tstEngine) ensureNonExistingMarket(t *testing.T, market string) {
	t.Helper()
	e.markets.EXPECT().MarketExists(market).Times(1).Return(false)
}

func (e *tstEngine) ensureExistingMarket(t *testing.T, market string) {
	t.Helper()
	e.markets.EXPECT().MarketExists(market).Times(1).Return(true)
}

func (e *tstEngine) ensureNoAccountForParty(t *testing.T, partyID string) {
	t.Helper()
	delete(e.tokenBal, partyID)
}

func (e *tstEngine) ensureNetworkParameter(t *testing.T, key, value string) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	if err := e.netp.Update(context.Background(), key, value); err != nil {
		t.Fatalf("failed to set %s parameter: %v", key, err)
	}
}
