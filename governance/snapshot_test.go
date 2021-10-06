package governance_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	vgproto "code.vegaprotocol.io/protos/vega"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGovernanceSnapshotProposalReject(t *testing.T) {
	activeKey := (&types.PayloadGovernanceActive{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// get snapshot hash for active proposals
	emptyHash, err := eng.GetHash(activeKey)
	require.Nil(t, err)

	// Submit a proposal
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newOpenProposal(party.Id, time.Now())
	eng.expectAnyAssetTimes(2)
	eng.expectSendOpenProposalEvent(t, party, proposal)

	toSubmit, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)
	assert.NoError(t, err)

	// get snapshot hash for active proposals
	h1, err := eng.GetHash(activeKey)
	require.Nil(t, err)

	// Reject proposal
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), vgproto.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET, errors.New("failure"))
	assert.NoError(t, err)

	// Check its changed now proposal has been rejected
	h2, err := eng.GetHash(activeKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))

	// Check the hash is the same before we submitted the proposal
	require.True(t, bytes.Equal(emptyHash, h2))

}

func TestGovernanceSnapshotProposalEnacted(t *testing.T) {
	activeKey := (&types.PayloadGovernanceActive{}).Key()
	enactedKey := (&types.PayloadGovernanceEnacted{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// get snapshot hashes
	emptyActive, err := eng.GetHash(activeKey)
	require.Nil(t, err)
	emptyEnacted, err := eng.GetHash(enactedKey)
	require.Nil(t, err)

	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())

	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(9))
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// make proposal
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)
	assert.NoError(t, err)

	// vote for it
	eng.expectSendVoteEvent(t, voter1, proposal)
	err = eng.AddVote(context.Background(),
		types.VoteSubmission{Value: vgproto.Vote_VALUE_YES, ProposalID: proposal.ID}, voter1.Id)
	assert.NoError(t, err)

	// chain update
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, vgproto.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {

		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "7", v.TotalGovernanceTokenBalance())
	})

	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	eng.OnChainTimeUpdate(context.Background(), afterEnactment)

	// check snapshot hashes (should have no active proposals and one enacted proposal)
	activeHash, err := eng.GetHash(activeKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(emptyActive, activeHash)) // active proposal should be gone now its enacted

	enactedHash, err := eng.GetHash(enactedKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(emptyEnacted, enactedHash))

}

func TestGovernanceSnapshotRoundTrip(t *testing.T) {
	activeKey := (&types.PayloadGovernanceActive{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// initial state
	h, err := eng.GetHash(activeKey)
	require.Nil(t, err)

	proposer := eng.newValidParty("proposer", 1)
	//voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	//voter2 := eng.newValidPartyTimes("voter2", 1, 0)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())
	proposal2 := eng.newOpenProposal(proposer.Id, time.Now())
	ctx := context.Background()

	// setup
	//eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
	//	Return(num.NewUint(9))
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	_, err = eng.SubmitProposal(ctx, *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)
	assert.Nil(t, err)

	_, err = eng.SubmitProposal(ctx, *types.ProposalSubmissionFromProposal(&proposal2), proposal2.ID, proposer.Id)
	assert.Nil(t, err)

	h2, err := eng.GetHash(activeKey)
	require.Nil(t, err)
	assert.False(t, bytes.Equal(h, h2))

	engg := getTestEngine(t)
	defer engg.ctrl.Finish()

	state, err := eng.GetState(activeKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	err = engg.LoadState(
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	h3, err := engg.GetHash(activeKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(h2, h3))

}
