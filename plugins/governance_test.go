package plugins_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/plugins/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type governanceTst struct {
	*plugins.Governance
	ctrl  *gomock.Controller
	pBuf  *mocks.MockPropBuffer
	vBuf  *mocks.MockVoteBuffer
	pCh   chan []types.Proposal
	vCh   chan []types.Vote
	ctx   context.Context
	cfunc context.CancelFunc
}

func getTestGovernance(t *testing.T) *governanceTst {
	ctrl := gomock.NewController(t)
	vBuf := mocks.NewMockVoteBuffer(ctrl)
	pBuf := mocks.NewMockPropBuffer(ctrl)
	ctx, cfunc := context.WithCancel(context.Background())
	return &governanceTst{
		Governance: plugins.NewGovernance(pBuf, vBuf),
		ctrl:       ctrl,
		pBuf:       pBuf,
		vBuf:       vBuf,
		pCh:        make(chan []types.Proposal),
		vCh:        make(chan []types.Vote),
		ctx:        ctx,
		cfunc:      cfunc,
	}
}

func (t *governanceTst) Finish() {
	t.cfunc()
	close(t.vCh)
	close(t.pCh)
	t.ctrl.Finish()
}

func TestStartStopGovernance(t *testing.T) {
	t.Run("start and stop plugin manually", testStartStopManual)
	t.Run("start and stop plugin through context", testStartStopContext)
}

func TestProposalWithVotes(t *testing.T) {
	t.Run("new proposal, then a single vote - success", testNewProposalThenVoteSuccess)
	t.Run("new proposal, first a single vote - success", testNewProposalFirstVoteSuccess)
	t.Run("new proposal, changing votes - success", testNewProposalChangingVoteSuccess)
}

func testNewProposalChangingVoteSuccess(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	proposal := types.Proposal{
		ID:        "prop-ID",
		Reference: "prop-ref",
		PartyID:   "prop-party-ID",
		State:     types.Proposal_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}
	vote := types.Vote{
		ProposalID: proposal.ID,
		PartyID:    "vote-party-ID",
		Value:      types.Vote_YES,
	}
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)
	// first the vote event is sent
	plugin.vCh <- []types.Vote{vote}
	plugin.vCh <- []types.Vote{}
	// then the proposal event
	plugin.pCh <- []types.Proposal{proposal}
	plugin.pCh <- []types.Proposal{}
	// By ID -> we get the proposal
	p, err := plugin.GetProposalByID(proposal.ID)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, proposal, *p.Proposal)
	assert.NotEmpty(t, p.Yes)
	assert.Equal(t, 1, len(p.Yes))
	assert.Empty(t, p.No) // no votes against were cast yet

	// same party now votes no
	vote.Value = types.Vote_NO
	plugin.vCh <- []types.Vote{vote}
	plugin.vCh <- []types.Vote{}
	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	// stop the plugin here already, we've gotten all the data needed for the test
	plugin.Stop()
	p, err = plugin.GetProposalByID(proposal.ID)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, proposal, *p.Proposal)
	// updated value is counted
	assert.NotEmpty(t, p.No)
	assert.Equal(t, 1, len(p.No))
	assert.Empty(t, p.Yes) // old vote is gone
}

func testNewProposalFirstVoteSuccess(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	proposal := types.Proposal{
		ID:        "prop-ID",
		Reference: "prop-ref",
		PartyID:   "prop-party-ID",
		State:     types.Proposal_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewAsset{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}
	vote := types.Vote{
		ProposalID: proposal.ID,
		PartyID:    "vote-party-ID",
		Value:      types.Vote_YES,
	}
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	// ensure the proposal doesn't exist yet:
	_, err := plugin.GetProposalByID(proposal.ID)
	assert.Error(t, err)
	assert.Equal(t, plugins.ErrProposalNotFound, err)
	_, err = plugin.GetProposalByReference(proposal.Reference)
	assert.Error(t, err)
	assert.Equal(t, plugins.ErrProposalNotFound, err)

	// first the vote event is sent
	plugin.vCh <- []types.Vote{vote}
	plugin.vCh <- []types.Vote{}
	// then the proposal event
	plugin.pCh <- []types.Proposal{proposal}
	plugin.pCh <- []types.Proposal{}
	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	// stop the plugin here already, we've gotten all the data needed for the test
	plugin.Stop()
	// By ID -> we get the proposal
	p, err := plugin.GetProposalByID(proposal.ID)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, proposal, *p.Proposal)
	assert.NotEmpty(t, p.Yes)
}

func testNewProposalThenVoteSuccess(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	proposal := types.Proposal{
		ID:        "prop-ID",
		Reference: "prop-ref",
		PartyID:   "prop-party-ID",
		State:     types.Proposal_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewMarket{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}
	vote := types.Vote{
		ProposalID: proposal.ID,
		PartyID:    "vote-party-ID",
		Value:      types.Vote_YES,
	}
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)
	plugin.pCh <- []types.Proposal{proposal}
	plugin.pCh <- []types.Proposal{}
	plugin.vCh <- []types.Vote{vote}
	plugin.vCh <- []types.Vote{}
	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	// stop the plugin here already, we've gotten all the data needed for the test
	plugin.Stop()
	// By ID -> we get the proposal
	p, err := plugin.GetProposalByID(proposal.ID)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, proposal, *p.Proposal)
	assert.NotEmpty(t, p.Yes)

	// by reference -> same result
	pRef, err := plugin.GetProposalByReference(proposal.Reference)
	assert.NoError(t, err)
	assert.Equal(t, proposal, *pRef.Proposal)

	// proposal is open, should get it from the open proposals
	state := types.Proposal_OPEN
	open := plugin.GetProposals(&state)
	assert.NotEmpty(t, open)
	assert.Equal(t, 1, len(open))
	assert.Equal(t, proposal, *open[0].Proposal)

	all := plugin.GetProposals(nil)
	assert.NotEmpty(t, all)
	assert.Equal(t, proposal, *all[0].Proposal)
}

func testStartStopManual(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	ctx := context.Background()
	vCh := make(chan []types.Vote)
	pCh := make(chan []types.Proposal)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(vCh, 1)
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(pCh, 1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1).Do(func(_ int) {
		close(vCh)
	})
	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1).Do(func(_ int) {
		close(pCh)
	})
	plugin.Start(ctx)
	plugin.Stop()
}

func testStartStopContext(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	ctx, cfunc := context.WithCancel(context.Background())
	vCh := make(chan []types.Vote)
	pCh := make(chan []types.Proposal)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(vCh, 1)
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(pCh, 1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1).Do(func(_ int) {
		close(vCh)
	})
	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1).Do(func(_ int) {
		close(pCh)
	})
	plugin.Start(ctx)
	cfunc()
	// read all 3 channels (the data channels should be closed)
	// this ensures all unsubscribe calls have been made
	<-ctx.Done()
	<-vCh
	<-pCh
}
