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

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalForNewAsset(t *testing.T) {
	t.Run("Submitting a proposal for new asset succeeds", testSubmittingProposalForNewAssetSucceeds)
	t.Run("Submitting a proposal for new asset with closing time before validation time fails", testSubmittingProposalForNewAssetWithClosingTimeBeforeValidationTimeFails)

	t.Run("Voting during validation of proposal for new asset succeeds", testVotingDuringValidationOfProposalForNewAssetSucceeds)
}

func testSubmittingProposalForNewAssetSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewAsset(party.Id, eng.tsvc.GetTimeNow())

	// setup
	eng.assets.EXPECT().NewAsset(gomock.Any(), proposal.ID, gomock.Any()).Times(1).Return(proposal.ID, nil)
	eng.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)

	// expect
	eng.expectProposalWaitingForNodeVoteEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	assert.False(t, toSubmit.IsNewMarket())
	require.Nil(t, toSubmit.NewMarket())
}

func testSubmittingProposalForNewAssetWithClosingTimeBeforeValidationTimeFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewAsset(party, eng.tsvc.GetTimeNow())
	proposal.Terms.ValidationTimestamp = proposal.Terms.ClosingTimestamp + 10

	// setup
	eng.expectRejectedProposalEvent(t, party, proposal.ID, types.ProposalErrorIncompatibleTimestamps)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "proposal closing time cannot be before validation time, expected >")
}

func testVotingDuringValidationOfProposalForNewAssetSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewAsset(proposer, eng.tsvc.GetTimeNow())

	// setup
	var bAsset *assets.Asset
	var fcheck func(interface{}, bool)
	var rescheck validators.Resource
	eng.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, ref string, assetDetails *types.AssetDetails) (string, error) {
		bAsset = assets.NewAsset(builtin.New(ref, assetDetails))
		return ref, nil
	})
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).DoAndReturn(func(id string) (*assets.Asset, error) {
		return bAsset, nil
	})
	eng.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Do(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		fcheck = f
		rescheck = r
		return nil
	})
	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 1)

	// expect
	eng.expectProposalWaitingForNodeVoteEvent(t, proposer, proposal.ID)

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

	// call success on the validation
	fcheck(rescheck, true)

	// then
	require.NoError(t, err)
	afterValidation := time.Unix(proposal.Terms.ValidationTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterValidation)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")
	eng.assets.EXPECT().SetPendingListing(gomock.Any(), proposal.ID).Times(1)

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
