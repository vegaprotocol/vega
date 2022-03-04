package governance_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/governance"
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
	proposal := eng.newProposalForNewMarket(proposer.Id, time.Now())
	ctx := context.Background()

	// setup
	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// setup
	eng.expectVoteEvent(t, voter1.Id, proposal.ID)

	// then
	err = eng.addYesVote(t, voter1.Id, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")

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
	err = eng.addNoVote(t, voter2.Id, proposal.ID)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotOpenForVotes.Error())

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

	eng2.broker.EXPECT().SendBatch(gomock.Any()).Times(1)

	// Load checkpoint
	require.NoError(t, eng2.Load(ctx, data))

	enact, noClose := eng2.OnChainTimeUpdate(ctx, afterEnactment)
	require.Empty(t, noClose)
	require.NotEmpty(t, enact)

	data = append(data, []byte("foo")...)
	require.Error(t, eng2.Load(ctx, data))
}
