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

func TestProposals(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	party := "prop-party"
	proposals := []types.Proposal{{
		ID:        "prop-1",
		Reference: "prop-ref1",
		PartyID:   party,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewMarket{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-2",
		Reference: "prop-ref2",
		PartyID:   party,
		State:     types.Proposal_STATE_FAILED,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewAsset{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-3",
		Reference: "prop-ref2", // colliding reference
		PartyID:   party,
		State:     types.Proposal_STATE_REJECTED,
		Terms: &types.ProposalTerms{Change: &types.ProposalTerms_UpdateMarket{
			UpdateMarket: &types.UpdateMarket{},
		}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-4",
		Reference: "prop-ref4",
		PartyID:   party,
		State:     types.Proposal_STATE_PASSED,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-5",
		Reference: "prop-ref5",
		PartyID:   party,
		State:     types.Proposal_STATE_ENACTED,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}

	plugin.pCh <- proposals

	t.Run("proposals by party", func(t *testing.T) {
		loaded := plugin.GetProposalsByParty(party, nil)
		assert.Len(t, loaded, len(proposals))

		selector := types.Proposal_STATE_REJECTED
		loaded = plugin.GetProposalsByParty(party, &selector)
		assert.Len(t, loaded, 1)
		assert.Equal(t, proposals[2], *loaded[0].Proposal)
		assert.Len(t, loaded[0].Yes, 0)
		assert.Len(t, loaded[0].No, 0)

		loaded = plugin.GetProposalsByParty("not-a-party", nil)
		assert.Len(t, loaded, 0)
	})

	t.Run("proposal by id", func(t *testing.T) {
		loaded, err := plugin.GetProposalByID(proposals[0].ID)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, proposals[0], *loaded.Proposal)

		loaded, err = plugin.GetProposalByID("not-an-id")
		assert.Error(t, err)
		assert.Equal(t, err, plugins.ErrProposalNotFound)
		assert.Nil(t, loaded)
	})

	t.Run("proposal by reference", func(t *testing.T) {
		ambiguousRef := proposals[1].Reference
		loaded, err := plugin.GetProposalByReference(ambiguousRef)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, ambiguousRef, loaded.Proposal.Reference,
			"valid but random proposal selected if reference is ambiguous")

		loaded, err = plugin.GetProposalByReference("not-a-ref")
		assert.Error(t, err)
		assert.Equal(t, err, plugins.ErrProposalNotFound)
		assert.Nil(t, loaded)
	})
	t.Run("new market proposals", func(t *testing.T) {
		loaded := plugin.GetNewMarketProposals(nil)
		assert.Len(t, loaded, 1)
		assert.NotNil(t, loaded[0])
		assert.NotNil(t, loaded[0].Proposal)
	})
	t.Run("new asset proposals", func(t *testing.T) {
		loaded := plugin.GetNewAssetProposals(nil)
		assert.Len(t, loaded, 1)
		assert.NotNil(t, loaded[0])
		assert.NotNil(t, loaded[0].Proposal)
	})
	t.Run("update market proposals", func(t *testing.T) {
		validMarket := "" //TODO: replace this with a valid market once supported
		loaded := plugin.GetUpdateMarketProposals(validMarket, nil)
		assert.Len(t, loaded, 1)
		assert.NotNil(t, loaded[0])
		assert.NotNil(t, loaded[0].Proposal)
	})
	t.Run("network parameters proposals", func(t *testing.T) {
		loaded := plugin.GetNetworkParametersProposals(nil)
		assert.Len(t, loaded, 2)
		assert.NotNil(t, loaded[0])
		assert.NotNil(t, loaded[0].Proposal)
		assert.NotNil(t, loaded[1])
		assert.NotNil(t, loaded[1].Proposal)
	})

	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}

func TestVotes(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	party1 := "party1"
	wait4Party1Votes, party1VotesSub := plugin.SubscribePartyVotes(party1)

	proposal1ID := "prop-1"
	votes1 := []types.Vote{{
		PartyID:    party1,
		ProposalID: proposal1ID,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    party1,
		ProposalID: proposal1ID,
		Value:      types.Vote_VALUE_NO,
	}}
	plugin.vCh <- votes1
	<-wait4Party1Votes

	t.Run("dangling votes must not appear in governance data", func(t *testing.T) {
		loaded := plugin.GetProposals(nil)
		assert.Empty(t, loaded)
	})

	t.Run("dangling votes by party", func(t *testing.T) {
		loaded := plugin.GetVotesByParty(party1)
		assert.Len(t, loaded, 2)
		assert.Equal(t, party1, loaded[0].PartyID)
		assert.Equal(t, votes1[0], *loaded[0])
		assert.Equal(t, votes1[1], *loaded[1])
	})

	plugin.pCh <- []types.Proposal{{
		ID:        proposal1ID,
		PartyID:   party1,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewMarket{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}
	wait4Proposals, propSub := plugin.SubscribeAll()
	<-wait4Proposals
	plugin.UnsubscribeAll(propSub)

	t.Run("no dangling votes since proposal shows up", func(t *testing.T) {
		props := plugin.GetProposals(nil)
		assert.Len(t, props, 1)
		assert.NotNil(t, props[0])
		assert.Empty(t, props[0].Yes, "previous vote is ignored")
		assert.Len(t, props[0].No, 1, "overriding vote is counted")

		votes := plugin.GetVotesByParty(party1)
		assert.Len(t, votes, 2, "despite only 1 one on proposal, total votes cast is 2")
	})

	proposal2ID := "prop-2"
	party2 := "party2"
	plugin.vCh <- []types.Vote{{
		PartyID:    party2,
		ProposalID: proposal1ID,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    party1,
		ProposalID: proposal2ID,
		Value:      types.Vote_VALUE_NO,
	}}
	<-wait4Party1Votes
	plugin.UnsubscribePartyVotes(party1, party1VotesSub)

	t.Run("regular + dangling votes", func(t *testing.T) {
		votes := plugin.GetVotesByParty(party1)
		assert.Len(t, votes, 3, "track all votes, even dangling and overriden ones")

		prop, err := plugin.GetProposalByID(proposal2ID)
		assert.Error(t, err, "dangling vote should not create proposals")
		assert.Nil(t, prop)

		prop, err = plugin.GetProposalByID(proposal1ID)
		assert.NoError(t, err)
		assert.NotNil(t, prop)
		assert.Len(t, prop.Yes, 1)
		assert.Equal(t, prop.Yes[0].PartyID, party2)
		assert.Len(t, prop.No, 1)
		assert.Equal(t, prop.No[0].PartyID, party1)
	})

	party3 := "party3"
	wait4Party3Votes, party3VotesSub := plugin.SubscribePartyVotes(party3)
	plugin.vCh <- []types.Vote{{
		PartyID:    party3,
		ProposalID: proposal1ID,
		Value:      types.Vote_VALUE_YES,
	}}
	<-wait4Party3Votes
	plugin.UnsubscribePartyVotes(party3, party3VotesSub)

	t.Run("normal boring vote", func(t *testing.T) {
		loaded := plugin.GetVotesByParty(party3)
		assert.Len(t, loaded, 1)
		assert.Equal(t, party3, loaded[0].PartyID)

		prop, err := plugin.GetProposalByID(proposal1ID)
		assert.NoError(t, err)
		assert.NotNil(t, prop)
		assert.Len(t, prop.Yes, 2)
	})
	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}

func testNewProposalChangingVoteSuccess(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	proposal := types.Proposal{
		ID:        "prop-ID",
		Reference: "prop-ref",
		PartyID:   "prop-party-ID",
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}
	vote := types.Vote{
		ProposalID: proposal.ID,
		PartyID:    "vote-party-ID",
		Value:      types.Vote_VALUE_YES,
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
	vote.Value = types.Vote_VALUE_NO
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
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewAsset{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}
	vote := types.Vote{
		ProposalID: proposal.ID,
		PartyID:    "vote-party-ID",
		Value:      types.Vote_VALUE_YES,
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
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewMarket{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}
	vote := types.Vote{
		ProposalID: proposal.ID,
		PartyID:    "vote-party-ID",
		Value:      types.Vote_VALUE_YES,
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
	state := types.Proposal_STATE_OPEN
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
