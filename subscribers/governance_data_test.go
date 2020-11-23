package subscribers_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/stretchr/testify/assert"
)

func TestFilterMany(t *testing.T) {
	t.Run("Filter proposals by status", testFilterByState)
	t.Run("Filter proposals by party", testFilterByParty)
	t.Run("No filter - unique votes", testNoFilterVotes)
}

func TestFilterOne(t *testing.T) {
	t.Run("Get proposal by ID returns last version", testGetByID)
}

func testGetByID(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	sub := subscribers.NewGovernanceDataSub(ctx, true)
	ids := []string{
		"prop1",
		"prop2",
	}
	lastState := types.Proposal_STATE_FAILED
	for _, id := range ids {
		sub.Push(events.NewProposalEvent(ctx, types.Proposal{
			PartyID: "party",
			ID:      id,
			State:   types.Proposal_STATE_OPEN,
		}))
		sub.Push(events.NewProposalEvent(ctx, types.Proposal{
			PartyID: "party",
			ID:      id,
			State:   lastState,
		}))
	}
	for _, id := range ids {
		data := sub.Filter(false, subscribers.ProposalByID(id))
		assert.Equal(t, 1, len(data))
		assert.Equal(t, id, data[0].Proposal.ID)
		assert.Equal(t, lastState, data[0].Proposal.State)
	}
}

func testFilterByState(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	sub := subscribers.NewGovernanceDataSub(ctx, true)
	party := "test-party"
	states := []types.Proposal_State{
		types.Proposal_STATE_OPEN,
		types.Proposal_STATE_DECLINED,
		types.Proposal_STATE_FAILED,
		types.Proposal_STATE_OPEN,
		types.Proposal_STATE_PASSED,
		types.Proposal_STATE_ENACTED,
		types.Proposal_STATE_REJECTED,
		types.Proposal_STATE_REJECTED,
	}
	expNr := map[types.Proposal_State]int{
		types.Proposal_STATE_OPEN:     2,
		types.Proposal_STATE_DECLINED: 1,
		types.Proposal_STATE_FAILED:   1,
		types.Proposal_STATE_PASSED:   1,
		types.Proposal_STATE_ENACTED:  1,
		types.Proposal_STATE_REJECTED: 2,
	}
	for i, s := range states {
		prop := types.Proposal{
			PartyID: party,
			ID:      fmt.Sprintf("test-prop-%d", i),
			State:   s,
		}
		sub.Push(events.NewProposalEvent(ctx, prop))
	}
	for s, exp := range expNr {
		filter := subscribers.ProposalByState(s)
		data := sub.Filter(false, filter)
		assert.Equal(t, len(data), exp)
	}
}

func testFilterByParty(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	sub := subscribers.NewGovernanceDataSub(ctx, true)
	assert.Empty(t, sub.Filter(false))
	party := "test-party"
	ids := []string{
		"prop-1",
		"prop-2",
		"prop-3",
	}
	for _, id := range ids {
		prop := types.Proposal{
			PartyID: party,
			ID:      id,
			State:   types.Proposal_STATE_OPEN,
		}
		sub.Push(events.NewProposalEvent(ctx, prop))
	}
	sub.Push(events.NewProposalEvent(ctx, types.Proposal{
		PartyID: "some-other-party",
		ID:      "foobar",
		State:   types.Proposal_STATE_OPEN,
	}))
	data := sub.Filter(false, subscribers.ProposalByPartyID(party))
	assert.Equal(t, len(ids), len(data))
	other := sub.Filter(false, subscribers.ProposalByPartyID("some-other-party"))
	assert.Equal(t, 1, len(other))
	defer cfunc()
}

func testNoFilterVotes(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	sub := subscribers.NewGovernanceDataSub(ctx, true)
	parties := []string{
		"party1",
		"party2",
		"party3",
	}
	props := []string{
		"prop1",
		"prop2",
		"prop3",
	}
	// last vote will always be yes
	for i, p := range parties {
		sub.Push(events.NewProposalEvent(ctx, types.Proposal{
			PartyID: p,
			ID:      props[i],
			State:   types.Proposal_STATE_OPEN,
		}))
	}
	for _, p := range parties {
		for i := range props {
			sub.Push(events.NewVoteEvent(ctx, types.Vote{
				ProposalID: props[i],
				PartyID:    p,
				Value:      types.Vote_VALUE_NO,
			}))
			sub.Push(events.NewVoteEvent(ctx, types.Vote{
				ProposalID: props[i],
				PartyID:    p,
				Value:      types.Vote_VALUE_YES,
			}))
			if i > 1 {
				sub.Push(events.NewVoteEvent(ctx, types.Vote{
					ProposalID: props[i],
					PartyID:    p,
					Value:      types.Vote_VALUE_YES,
				}))
				sub.Push(events.NewVoteEvent(ctx, types.Vote{
					ProposalID: props[i],
					PartyID:    p,
					Value:      types.Vote_VALUE_NO,
				}))
			}
		}
	}
	raw := sub.Filter(false)
	// votes were No -> Yes (and last case another YES -> NO)
	for _, d := range raw {
		assert.Equal(t, len(d.Yes), len(d.No))
	}
	last := sub.Filter(true)
	for _, d := range last {
		assert.NotEqual(t, len(d.Yes), len(d.No))
	}
	assert.Equal(t, len(parties), len(raw))
	assert.Equal(t, len(parties), len(last))
	// no := last[len(last)-1]
	special := sub.Filter(true, subscribers.ProposalByID(props[len(props)-1]))
	no := special[0]
	assert.Empty(t, no.Yes)
	assert.NotEmpty(t, no.No)
}
