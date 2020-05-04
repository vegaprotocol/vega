package plugins_test

import (
	"context"
	"fmt"
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
		State:     types.Proposal_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewMarket{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-2",
		Reference: "prop-ref2",
		PartyID:   party,
		State:     types.Proposal_FAILED,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewAsset{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-3",
		Reference: "prop-ref2", // colliding reference
		PartyID:   party,
		State:     types.Proposal_REJECTED,
		Terms: &types.ProposalTerms{Change: &types.ProposalTerms_UpdateMarket{
			UpdateMarket: &types.UpdateMarket{},
		}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-4",
		Reference: "prop-ref4",
		PartyID:   party,
		State:     types.Proposal_PASSED,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        "prop-5",
		Reference: "prop-ref5",
		PartyID:   party,
		State:     types.Proposal_ENACTED,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}

	plugin.pCh <- proposals

	t.Run("proposals by party", func(t *testing.T) {
		loaded := plugin.GetProposalsByParty(party, nil)
		assert.Len(t, loaded, len(proposals))

		selector := types.Proposal_REJECTED
		loaded = plugin.GetProposalsByParty(party, &selector)
		assert.Len(t, loaded, 1)
		assert.Equal(t, *loaded[0].Proposal, proposals[2])
		assert.Len(t, loaded[0].Yes, 0)
		assert.Len(t, loaded[0].No, 0)

		loaded = plugin.GetProposalsByParty("not-a-party", nil)
		assert.Len(t, loaded, 0)
	})

	t.Run("proposal by id", func(t *testing.T) {
		loaded, err := plugin.GetProposalByID("prop-1")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, *loaded.Proposal, proposals[0])

		loaded, err = plugin.GetProposalByID("not-an-id")
		assert.Error(t, err)
		assert.Equal(t, err, plugins.ErrProposalNotFound)
		assert.Nil(t, loaded)
	})

	t.Run("proposal by reference", func(t *testing.T) {
		loaded, err := plugin.GetProposalByReference("prop-ref2")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, *loaded.Proposal, proposals[1],
			"picks the first submitted proposal with the matching reference")

		loaded, err = plugin.GetProposalByReference("not-a-ref")
		assert.Error(t, err)
		assert.Equal(t, err, plugins.ErrProposalNotFound)
		assert.Nil(t, loaded)
	})
	t.Run("new market proposals", func(t *testing.T) {
		loaded := plugin.GetNewMarketProposals(nil)
		assert.Len(t, loaded, 1)
		assert.NotNil(t, loaded)
	})
	t.Run("new asset proposals", func(t *testing.T) {
		loaded := plugin.GetNewAssetProposals(nil)
		assert.Len(t, loaded, 1)
		assert.NotNil(t, loaded)
	})
	t.Run("update market proposals", func(t *testing.T) {
		loaded := plugin.GetUpdateMarketProposals("", nil)
		assert.Len(t, loaded, 1)
		assert.NotNil(t, loaded)
	})
	t.Run("network parameters proposals", func(t *testing.T) {
		loaded := plugin.GetNetworkParametersProposals(nil)
		assert.Len(t, loaded, 2)
		assert.NotNil(t, loaded)
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
		Value:      types.Vote_YES,
	}, {
		PartyID:    party1,
		ProposalID: proposal1ID,
		Value:      types.Vote_NO,
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
		State:     types.Proposal_OPEN,
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
		Value:      types.Vote_YES,
	}, {
		PartyID:    party1,
		ProposalID: proposal2ID,
		Value:      types.Vote_NO,
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
		Value:      types.Vote_YES,
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

func TestStreamSubscriptions(t *testing.T) {
	t.Run("test that dangling votes do not produce gov data", danglingVoteImpactOnProposals)
	t.Run("test general governance stream", generalGovernanceSubs)
}

func danglingVoteImpactOnProposals(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	chG, idxG := plugin.SubscribeAll()
	chParty, idxParty := plugin.SubscribePartyProposals("partyX")

	plugin.vCh <- []types.Vote{{
		PartyID:    "some-party",
		ProposalID: "non-existent-proposal1",
		Value:      types.Vote_YES,
	}, {
		PartyID:    "some-party",
		ProposalID: "non-existent-proposal2",
		Value:      types.Vote_YES,
	}, {
		PartyID:    "some-party",
		ProposalID: "non-existent-proposal3",
		Value:      types.Vote_YES,
	}}
	for i := 0; i < 100; i++ { // polling for 1s to make sure nothing is omitted
		select {
		case danglingVotes := <-chG:
			assert.Fail(t, "received dangling votes governance data on general", danglingVotes)
		default:
		}
		time.Sleep(time.Millisecond * 10)
	}
	select {
	case danglingVotes := <-chParty:
		assert.Fail(t, "received dangling votes governance data on party", danglingVotes)
	default:
	}
	plugin.UnsubscribeAll(idxG)
	plugin.UnsubscribeAll(idxParty)

	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}

// the function restarts plugin to reduce likelihood of side-effects
func generalGovernanceSubs(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	ch1, idx1 := plugin.SubscribeAll()
	ch2, idx2 := plugin.SubscribeAll()
	proposal := types.Proposal{
		ID:        "proposal1",
		PartyID:   "some-party",
		State:     types.Proposal_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewAsset{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}
	plugin.pCh <- []types.Proposal{proposal}

	received1 := <-ch1
	assert.Len(t, received1, 1)
	assert.Equal(t, proposal, *received1[0].Proposal)
	assert.Nil(t, received1[0].Yes)
	assert.Nil(t, received1[0].No)

	received2 := <-ch2
	assert.Len(t, received2, len(received1))
	assert.Equal(t, received1, received2)

	plugin.UnsubscribeAll(idx2)

	props := make([]types.Proposal, 100)
	for i := 0; i < 100; i++ {
		props[i] = types.Proposal{
			ID:      "prop-" + fmt.Sprintf("%3d", i),
			PartyID: "spammer",
			State:   types.Proposal_OPEN,
			Terms: &types.ProposalTerms{Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{},
			}},
			Timestamp: time.Now().Add(3600 * time.Second).Unix(),
		}
	}
	plugin.pCh <- props
	received := <-ch1
	assert.Len(t, received, len(props))
	for i, g := range received {
		assert.Equal(t, props[i], *g.Proposal)
		assert.Nil(t, g.Yes)
		assert.Nil(t, g.No)
	}
	plugin.UnsubscribeAll(idx1)

	plugin.pCh <- []types.Proposal{{
		ID:        "proposal-post-close",
		PartyID:   "some-other-party",
		State:     types.Proposal_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewMarket{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}

	// poll for 300ms give plugin to make sure the proposal
	// isn't skipped due to concurrency issues
	for i := 0; i < 100; i++ {
		select {
		case data := <-ch1:
			assert.Empty(t, data, "received data after closing channel 1")
		case data := <-ch2:
			assert.Empty(t, data, "received data after closing channel 2")
		default:
		}
		time.Sleep(time.Millisecond * 3)
	}

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
