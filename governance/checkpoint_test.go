package governance_test

import (
	"context"
	"testing"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpoint(t *testing.T) {
	t.Run("Basic test -> get checkpoints at various points in time, load checkpoint", testCheckpointSuccess)
}

func testCheckpointSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	voter2 := eng.newValidPartyTimes("voter2", 1, 0)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())
	ctx := context.Background()

	// setup
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(9))
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(ctx, *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter1, proposal)

	// then
	err = eng.AddVote(ctx, types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter1.Id)

	// then
	assert.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "7", v.TotalGovernanceTokenBalance())
	})

	// checkpoint should be empty at this point
	data, err := eng.Checkpoint()
	require.NoError(t, err)
	require.Empty(t, data)

	// when
	eng.OnChainTimeUpdate(ctx, afterClosing)

	// the proposal should already be in the snapshot
	data, err = eng.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// when
	err = eng.AddVote(ctx, types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter2.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalPassed.Error())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, closed := eng.OnChainTimeUpdate(ctx, afterEnactment)

	// then
	require.NotEmpty(t, toBeEnacted)
	require.Empty(t, closed)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// Now take the snapshot
	data, err = eng.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	eng2 := getTestEngine(t)
	defer eng2.ctrl.Finish()

	// Load checkpoint
	require.NoError(t, eng2.Load(ctx, data))

	enact, noClose := eng2.OnChainTimeUpdate(ctx, afterEnactment)
	require.Empty(t, noClose)
	require.NotEmpty(t, enact)

	data = append(data, []byte("foo")...)
	require.Error(t, eng2.Load(ctx, data))
}
