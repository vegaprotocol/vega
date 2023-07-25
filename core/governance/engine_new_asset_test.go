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

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalForNewAsset(t *testing.T) {
	t.Run("Submitting a proposal for new asset succeeds", testSubmittingProposalForNewAssetSucceeds)
	t.Run("Submitting a proposal for new asset with closing time before validation time fails", testSubmittingProposalForNewAssetWithClosingTimeBeforeValidationTimeFails)
	t.Run("Voting during validation of proposal for new asset succeeds", testVotingDuringValidationOfProposalForNewAssetSucceeds)
	t.Run("Rejects erc20 proposals for address already used", testRejectsERC20ProposalForAddressAlreadyUsed)
}

func testRejectsERC20ProposalForAddressAlreadyUsed(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewAsset(party.Id, eng.tsvc.GetTimeNow().Add(48*time.Hour))

	newAssetERC20 := newAssetTerms()
	newAssetERC20.NewAsset.Changes.Source = &types.AssetDetailsErc20{
		ERC20: &types.ERC20{
			ContractAddress:   "0x690B9A9E9aa1C9dB991C7721a92d351Db4FaC990",
			LifetimeLimit:     num.NewUint(1),
			WithdrawThreshold: num.NewUint(1),
		},
	}
	proposal.Terms.Change = newAssetERC20

	// setup
	eng.assets.EXPECT().ExistsForEthereumAddress("0x690B9A9E9aa1C9dB991C7721a92d351Db4FaC990").Times(1).Return(true)

	// setup
	eng.expectRejectedProposalEvent(t, party.Id, proposal.ID, types.ProposalErrorERC20AddressAlreadyInUse)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	require.EqualError(t, err, governance.ErrErc20AddressAlreadyInUse.Error())
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForNewAssetSucceeds(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewAsset(party.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour))

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
	eng := getTestEngine(t, time.Now())

	// given
	party := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewAsset(party, eng.tsvc.GetTimeNow().Add(48*time.Hour))
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
	eng := getTestEngine(t, time.Now())

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewAsset(proposer, eng.tsvc.GetTimeNow().Add(2*time.Hour))

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
	eng.expectGetMarketState(t, proposal.ID)

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

func TestNoVotesAnd0RequiredFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	ctx := context.Background()
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.GovernanceProposalAssetRequiredParticipation, "0")).Times(1)
	assert.NoError(t,
		eng.netp.Update(ctx,
			"governance.proposal.asset.requiredParticipation",
			"0",
		),
	)

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewAsset(proposer, eng.tsvc.GetTimeNow().Add(2*time.Hour))

	// setup
	var fcheck func(interface{}, bool)
	var rescheck validators.Resource
	eng.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, ref string, assetDetails *types.AssetDetails) (string, error) {
		return ref, nil
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

	// call success on the validation
	fcheck(rescheck, true)

	// then
	require.NoError(t, err)
	afterValidation := time.Unix(proposal.Terms.ValidationTimestamp, 0).Add(time.Second)

	// setup
	// eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)
	eng.expectGetMarketState(t, proposal.ID)

	// when
	eng.OnTick(context.Background(), afterValidation)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// expect
	eng.expectDeclinedProposalEvent(t, proposal.ID, types.ProposalErrorParticipationThresholdNotReached)
	// empty list of votes
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1)

	eng.assets.EXPECT().SetRejected(gomock.Any(), proposal.ID).Times(1)

	// when
	eng.OnTick(context.Background(), afterClosing)

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, _ := eng.OnTick(context.Background(), afterEnactment)

	// then
	require.Len(t, toBeEnacted, 0)
}
