package plugins_test

import (
	"fmt"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestStreamSubscriptions(t *testing.T) {
	// the functions restart plugin to reduce likelihood of side-effects

	t.Run("test that dangling votes do not produce gov data", testDanglingVoteImpactOnProposals)
	t.Run("test reading closed subscription", testReadClosedSubs)

	t.Run("test general governance stream", testGeneralGovernanceSubs)
	t.Run("test party proposals stream", testPartyProposalsSubs)
	t.Run("test party votes stream", testPartyVotesSubs)
	t.Run("test proposal votes stream", testProposalVotesSubs)
}

func testDanglingVoteImpactOnProposals(t *testing.T) {
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
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    "some-party",
		ProposalID: "non-existent-proposal2",
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    "some-party",
		ProposalID: "non-existent-proposal3",
		Value:      types.Vote_VALUE_YES,
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

func testReadClosedSubs(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	ch1, idx1 := plugin.SubscribeAll()
	ch2, idx2 := plugin.SubscribeAll()
	ch3, idx3 := plugin.SubscribePartyProposals("partyX")
	ch4, idx4 := plugin.SubscribePartyVotes("partyX")
	ch5, idx5 := plugin.SubscribeProposalVotes("proposal-1")
	plugin.UnsubscribeAll(idx1)
	plugin.UnsubscribeAll(idx2)
	plugin.UnsubscribePartyProposals("partyX", idx3)
	plugin.UnsubscribePartyVotes("partyX", idx4)
	plugin.UnsubscribeProposalVotes("proposal-1", idx5)

	plugin.pCh <- []types.Proposal{{
		ID:        "proposal-post-close",
		PartyID:   "some-other-party",
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_NewMarket{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}
	plugin.vCh <- []types.Vote{{
		PartyID:    "some-other-party",
		ProposalID: "proposal-post-close",
		Value:      types.Vote_VALUE_YES,
	}}

	// poll for 300ms give plugin to make sure the channel is not reused
	// and proposal isn't skipped due to concurrency
	for i := 0; i < 100; i++ {
		select {
		case data := <-ch1:
			assert.Empty(t, data, "received data after closing channel 1")
		case data := <-ch2:
			assert.Empty(t, data, "received data after closing channel 2")
		case data := <-ch3:
			assert.Empty(t, data, "received data after closing channel 3")
		case data := <-ch4:
			assert.Empty(t, data, "received data after closing channel 4")
		case data := <-ch5:
			assert.Empty(t, data, "received data after closing channel 5")
		default:
		}
		time.Sleep(time.Millisecond * 3)
	}

	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}

func testGeneralGovernanceSubs(t *testing.T) {
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
		State:     types.Proposal_STATE_OPEN,
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
			State:   types.Proposal_STATE_OPEN,
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

	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}

func testPartyProposalsSubs(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	partyA, partyB := "partyA", "partyB"
	chA, idxA := plugin.SubscribePartyProposals(partyA)
	chB, idxB := plugin.SubscribePartyProposals(partyB)

	proposal1, proposal2 := "proposal1", "proposal2"

	plugin.vCh <- []types.Vote{{
		PartyID:    partyA,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyA,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyB,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyB,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}}
	plugin.pCh <- []types.Proposal{{
		ID:        proposal1,
		PartyID:   partyA,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        proposal2,
		PartyID:   partyB,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}
	receivedA := <-chA
	assert.Len(t, receivedA, 1)
	assert.Equal(t, proposal1, receivedA[0].Proposal.ID)
	assert.Equal(t, partyA, receivedA[0].Proposal.PartyID)
	assert.Len(t, receivedA[0].Yes, 2)
	assert.Len(t, receivedA[0].No, 0)

	receivedB := <-chB
	assert.Len(t, receivedB, 1)
	assert.Equal(t, proposal2, receivedB[0].Proposal.ID)
	assert.Equal(t, partyB, receivedB[0].Proposal.PartyID)
	assert.Len(t, receivedB[0].Yes, 0)
	assert.Len(t, receivedB[0].No, 2)

	plugin.UnsubscribePartyProposals(partyA, idxA)
	plugin.UnsubscribePartyProposals(partyB, idxB)

	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}

func testPartyVotesSubs(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()

	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	partyA, partyB := "partyA", "partyB"
	chA, idxA := plugin.SubscribePartyVotes(partyA)
	chB, idxB := plugin.SubscribePartyVotes(partyB)

	proposal1, proposal2 := "proposal1", "proposal2"

	plugin.vCh <- []types.Vote{{
		PartyID:    partyA,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyA,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyB,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyB,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_NO,
	}}
	receivedA := <-chA
	assert.Len(t, receivedA, 2)
	assert.Equal(t, partyA, receivedA[0].PartyID)
	assert.Equal(t, partyA, receivedA[1].PartyID)
	assert.Equal(t, proposal1, receivedA[0].ProposalID)
	assert.Equal(t, proposal2, receivedA[1].ProposalID)

	receivedB := <-chB
	assert.Len(t, receivedB, 2)
	assert.Equal(t, partyB, receivedB[0].PartyID)
	assert.Equal(t, partyB, receivedB[1].PartyID)
	assert.Equal(t, proposal2, receivedB[0].ProposalID)
	assert.Equal(t, proposal1, receivedB[1].ProposalID)

	plugin.pCh <- []types.Proposal{{
		ID:        proposal1,
		PartyID:   partyA,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        proposal2,
		PartyID:   partyB,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}

	// inverting votes
	plugin.vCh <- []types.Vote{{
		PartyID:    partyA,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyA,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyB,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyB,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}}
	receivedA = <-chA
	assert.Len(t, receivedA, 2)
	assert.Equal(t, partyA, receivedA[0].PartyID)
	assert.Equal(t, partyA, receivedA[1].PartyID)
	assert.Equal(t, proposal1, receivedA[0].ProposalID)
	assert.Equal(t, proposal2, receivedA[1].ProposalID)

	receivedB = <-chB
	assert.Len(t, receivedB, 2)
	assert.Equal(t, partyB, receivedB[0].PartyID)
	assert.Equal(t, partyB, receivedB[1].PartyID)
	assert.Equal(t, proposal2, receivedB[0].ProposalID)
	assert.Equal(t, proposal1, receivedB[1].ProposalID)

	// duplicate votes
	plugin.vCh <- []types.Vote{{
		PartyID:    partyA,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyA,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyB,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyB,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}}
	receivedA = <-chA
	assert.Len(t, receivedA, 2)
	assert.Equal(t, partyA, receivedA[0].PartyID)
	assert.Equal(t, partyA, receivedA[1].PartyID)
	assert.Equal(t, proposal1, receivedA[0].ProposalID)
	assert.Equal(t, proposal2, receivedA[1].ProposalID)

	receivedB = <-chB
	assert.Len(t, receivedB, 2)
	assert.Equal(t, partyB, receivedB[0].PartyID)
	assert.Equal(t, partyB, receivedB[1].PartyID)
	assert.Equal(t, proposal2, receivedB[0].ProposalID)
	assert.Equal(t, proposal1, receivedB[1].ProposalID)

	plugin.UnsubscribePartyVotes(partyA, idxA)
	plugin.UnsubscribePartyVotes(partyB, idxB)

	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}

func testProposalVotesSubs(t *testing.T) {
	plugin := getTestGovernance(t)
	defer plugin.Finish()
	plugin.pBuf.EXPECT().Subscribe().Times(1).Return(plugin.pCh, 1)
	plugin.vBuf.EXPECT().Subscribe().Times(1).Return(plugin.vCh, 1)
	plugin.Start(plugin.ctx)

	proposal1, proposal2 := "proposal1", "proposal2"
	ch1, idx1 := plugin.SubscribeProposalVotes(proposal1)
	ch2, idx2 := plugin.SubscribeProposalVotes(proposal2)

	partyA, partyB := "partyA", "partyB"
	plugin.vCh <- []types.Vote{{
		PartyID:    partyA,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyA,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyB,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyB,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_NO,
	}}
	received1 := <-ch1
	assert.Len(t, received1, 2)
	assert.Equal(t, proposal1, received1[0].ProposalID)
	assert.Equal(t, proposal1, received1[1].ProposalID)
	assert.Equal(t, partyA, received1[0].PartyID)
	assert.Equal(t, partyB, received1[1].PartyID)

	received2 := <-ch2
	assert.Len(t, received2, 2)
	assert.Equal(t, proposal2, received2[0].ProposalID)
	assert.Equal(t, proposal2, received2[1].ProposalID)
	assert.Equal(t, partyA, received2[0].PartyID)
	assert.Equal(t, partyB, received2[1].PartyID)

	plugin.pCh <- []types.Proposal{{
		ID:        proposal1,
		PartyID:   partyA,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}, {
		ID:        proposal2,
		PartyID:   partyB,
		State:     types.Proposal_STATE_OPEN,
		Terms:     &types.ProposalTerms{Change: &types.ProposalTerms_UpdateNetwork{}},
		Timestamp: time.Now().Add(3600 * time.Second).Unix(),
	}}

	// inverting votes
	plugin.vCh <- []types.Vote{{
		PartyID:    partyA,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyA,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyB,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyB,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}}
	received1 = <-ch1
	assert.Len(t, received1, 2)
	assert.Equal(t, proposal1, received1[0].ProposalID)
	assert.Equal(t, proposal1, received1[1].ProposalID)
	assert.Equal(t, partyA, received1[0].PartyID)
	assert.Equal(t, partyB, received1[1].PartyID)

	received2 = <-ch2
	assert.Len(t, received2, 2)
	assert.Equal(t, proposal2, received2[0].ProposalID)
	assert.Equal(t, proposal2, received2[1].ProposalID)
	assert.Equal(t, partyA, received2[0].PartyID)
	assert.Equal(t, partyB, received2[1].PartyID)

	// duplicate votes
	plugin.vCh <- []types.Vote{{
		PartyID:    partyA,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyA,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_YES,
	}, {
		PartyID:    partyB,
		ProposalID: proposal2,
		Value:      types.Vote_VALUE_NO,
	}, {
		PartyID:    partyB,
		ProposalID: proposal1,
		Value:      types.Vote_VALUE_YES,
	}}
	received1 = <-ch1
	assert.Len(t, received1, 2)
	assert.Equal(t, proposal1, received1[0].ProposalID)
	assert.Equal(t, proposal1, received1[1].ProposalID)
	assert.Equal(t, partyA, received1[0].PartyID)
	assert.Equal(t, partyB, received1[1].PartyID)

	received2 = <-ch2
	assert.Len(t, received2, 2)
	assert.Equal(t, proposal2, received2[0].ProposalID)
	assert.Equal(t, proposal2, received2[1].ProposalID)
	assert.Equal(t, partyA, received2[0].PartyID)
	assert.Equal(t, partyB, received2[1].PartyID)

	plugin.UnsubscribeProposalVotes(proposal1, idx1)
	plugin.UnsubscribeProposalVotes(proposal2, idx2)

	plugin.pBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.vBuf.EXPECT().Unsubscribe(1).Times(1)
	plugin.Stop()
}
