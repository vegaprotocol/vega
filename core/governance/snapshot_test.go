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
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	activeKey         = (&types.PayloadGovernanceActive{}).Key()
	enactedKey        = (&types.PayloadGovernanceEnacted{}).Key()
	nodeValidationKey = (&types.PayloadGovernanceNode{}).Key()
)

func TestGovernanceSnapshotProposalReject(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// get snapshot hash for active proposals
	emptyState, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	// Submit a proposal
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewMarket(party.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), nil, nil, true)
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	toSubmit, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)
	require.NoError(t, err)

	// get snapshot hash for active proposals
	s1, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	// Reject proposal
	eng.expectRejectedProposalEvent(t, party.Id, proposal.ID, types.ProposalErrorCouldNotInstantiateMarket)
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), types.ProposalErrorCouldNotInstantiateMarket, assert.AnError)
	require.NoError(t, err)

	// Check its changed now proposal has been rejected
	s2, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))

	// Check the hash is the same before we submitted the proposal
	require.True(t, bytes.Equal(emptyState, s2))
}

func TestGovernanceSnapshotProposalEnacted(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// get snapshot hashes
	emptyActive, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	emptyEnacted, _, err := eng.GetState(enactedKey)
	require.Nil(t, err)

	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	proposal := eng.newProposalForNewMarket(proposer.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), nil, nil, true)

	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, proposal.ID)

	// make proposal
	_, err = eng.submitProposal(t, proposal)

	require.NoError(t, err)

	eng.GetState(activeKey) // we call get state to get change back to false

	// vote for it
	eng.expectVoteEvent(t, voter1.Id, proposal.ID)
	err = eng.addYesVote(t, voter1.Id, proposal.ID)
	require.NoError(t, err)

	eng.GetState(activeKey) // we call get state to get change back to false

	// chain update
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")

	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.expectGetMarketState(t, proposal.ID)
	eng.OnTick(context.Background(), afterClosing)

	eng.GetState(activeKey) // we call get state to get change back to false

	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	eng.OnTick(context.Background(), afterEnactment)

	eng.GetState(activeKey) // we call get state to get change back to false

	// check snapshot hashes (should have no active proposals and one enacted proposal)
	activeHash, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(emptyActive, activeHash)) // active proposal should be gone now its enacted

	enactedHash, _, err := eng.GetState(enactedKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(emptyEnacted, enactedHash))
}

func TestGovernanceSnapshotWithInternalTimeTerminationProposalEnacted(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// get snapshot hashes
	emptyActive, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	emptyEnacted, _, err := eng.GetState(enactedKey)
	require.Nil(t, err)

	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	proposal := eng.newProposalForNewMarket(proposer.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), nil, nil, false)

	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, proposal.ID)

	// make proposal
	_, err = eng.submitProposal(t, proposal)

	require.NoError(t, err)

	eng.GetState(activeKey) // we call get state to get change back to false

	// vote for it
	eng.expectVoteEvent(t, voter1.Id, proposal.ID)
	err = eng.addYesVote(t, voter1.Id, proposal.ID)
	require.NoError(t, err)

	eng.GetState(activeKey) // we call get state to get change back to false

	// chain update
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")

	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.expectGetMarketState(t, proposal.ID)
	eng.OnTick(context.Background(), afterClosing)

	eng.GetState(activeKey) // we call get state to get change back to false

	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	eng.OnTick(context.Background(), afterEnactment)

	eng.GetState(activeKey) // we call get state to get change back to false

	// check snapshot hashes (should have no active proposals and one enacted proposal)
	activeHash, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(emptyActive, activeHash)) // active proposal should be gone now its enacted

	enactedHash, _, err := eng.GetState(enactedKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(emptyEnacted, enactedHash))
}

func TestGovernanceSnapshotNodeProposal(t *testing.T) {
	eng := getTestEngine(t, time.Now())
	defer eng.ctrl.Finish()

	// get snapshot state for active proposals
	emptyState, _, err := eng.GetState(nodeValidationKey)
	require.Nil(t, err)

	// Submit a proposal
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForNewAsset(party.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour))

	eng.expectProposalWaitingForNodeVoteEvent(t, party.Id, proposal.ID)
	eng.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes()
	eng.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	// submit new asset proposal
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)
	require.Nil(t, err)

	// vote on it even though its in waiting-for-node-vote-state
	voter1 := vgrand.RandomStr(5)
	eng.ensureTokenBalanceForParty(t, voter1, 1)
	eng.expectVoteEvent(t, voter1, proposal.ID)
	err = eng.addYesVote(t, voter1, proposal.ID)
	require.NoError(t, err)

	// get snapshot state for node proposals and hope its changed
	s1, _, err := eng.GetState(nodeValidationKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(emptyState, s1))

	// Get snapshot payload
	state, _, err := eng.GetState(nodeValidationKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapEng := getTestEngine(t, time.Now())
	defer snapEng.ctrl.Finish()

	snapEng.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	snapEng.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(1)

	// Load snapshot into a new engine
	snapEng.broker.EXPECT().Send(gomock.Any()).Times(1)
	_, err = snapEng.LoadState(
		context.Background(),
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	s2, _, err := snapEng.GetState(nodeValidationKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(s1, s2))

	// check the vote still exists
	err = proto.Unmarshal(s2, snap)
	require.Nil(t, err)
	pp := types.PayloadFromProto(snap)
	dd := pp.Data.(*types.PayloadGovernanceNode)
	assert.Equal(t, 1, len(dd.GovernanceNode.ProposalData[0].Yes))
}

func TestGovernanceSnapshotRoundTrip(t *testing.T) {
	activeKey := (&types.PayloadGovernanceActive{}).Key()
	eng := getTestEngine(t, time.Now())
	defer eng.ctrl.Finish()

	// initial state
	emptyState, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	proposer := eng.newValidParty("proposer", 1)
	proposal := eng.newProposalForNewMarket(proposer.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), nil, nil, true)
	ctx := context.Background()

	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, proposal.ID)

	_, err = eng.SubmitProposal(ctx, *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)
	assert.Nil(t, err)

	s1, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	assert.False(t, bytes.Equal(emptyState, s1))

	// given
	voter1 := vgrand.RandomStr(5)
	eng.ensureTokenBalanceForParty(t, voter1, 1)
	eng.expectVoteEvent(t, voter1, proposal.ID)
	err = eng.addYesVote(t, voter1, proposal.ID)
	require.NoError(t, err)
	s2, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	assert.False(t, bytes.Equal(s1, s2))

	snapEng := getTestEngine(t, time.Now())
	defer snapEng.ctrl.Finish()

	state, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapEng.broker.EXPECT().SendBatch(gomock.Any()).Times(2)
	_, err = snapEng.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)

	s3, _, err := snapEng.GetState(activeKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(s2, s3))
}

func TestGovernanceWithInternalTimeTerminationSnapshotRoundTrip(t *testing.T) {
	activeKey := (&types.PayloadGovernanceActive{}).Key()
	eng := getTestEngine(t, time.Now())
	defer eng.ctrl.Finish()

	// initial state
	emptyState, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	proposer := eng.newValidParty("proposer", 1)
	proposal := eng.newProposalForNewMarket(proposer.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), nil, nil, false)
	ctx := context.Background()

	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, proposal.ID)

	_, err = eng.SubmitProposal(ctx, *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)
	assert.Nil(t, err)

	s1, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	assert.False(t, bytes.Equal(emptyState, s1))

	// given
	voter1 := vgrand.RandomStr(5)
	eng.ensureTokenBalanceForParty(t, voter1, 1)
	eng.expectVoteEvent(t, voter1, proposal.ID)
	err = eng.addYesVote(t, voter1, proposal.ID)
	require.NoError(t, err)
	s2, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	assert.False(t, bytes.Equal(s1, s2))

	snapEng := getTestEngine(t, time.Now())
	defer snapEng.ctrl.Finish()

	state, _, err := eng.GetState(activeKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	snapEng.broker.EXPECT().SendBatch(gomock.Any()).Times(2)
	_, err = snapEng.LoadState(ctx, types.PayloadFromProto(snap))
	require.Nil(t, err)

	s3, _, err := snapEng.GetState(activeKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(s2, s3))
}

func TestGovernanceSnapshotEmpty(t *testing.T) {
	activeKey := (&types.PayloadGovernanceActive{}).Key()
	eng := getTestEngine(t, time.Now())
	defer eng.ctrl.Finish()

	s, _, err := eng.GetState(activeKey)
	require.Nil(t, err)
	require.NotNil(t, s)

	s, _, err = eng.GetState(enactedKey)
	require.Nil(t, err)
	require.NotNil(t, s)

	s, _, err = eng.GetState(nodeValidationKey)
	require.Nil(t, err)
	require.NotNil(t, s)
}
