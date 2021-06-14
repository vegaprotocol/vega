package subscribers_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/types"

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
	lastState := proto.Proposal_STATE_FAILED
	for _, id := range ids {
		sub.Push(events.NewProposalEvent(ctx, proto.Proposal{
			PartyId: "party",
			Id:      id,
			State:   proto.Proposal_STATE_OPEN,
		}))
		sub.Push(events.NewProposalEvent(ctx, proto.Proposal{
			PartyId: "party",
			Id:      id,
			State:   lastState,
		}))
	}
	for _, id := range ids {
		data := sub.Filter(false, subscribers.ProposalByID(id))
		assert.Equal(t, 1, len(data))
		assert.Equal(t, id, data[0].Proposal.Id)
		assert.Equal(t, lastState, data[0].Proposal.State)
	}
}

func testFilterByState(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	sub := subscribers.NewGovernanceDataSub(ctx, true)
	party := "test-party"
	states := []proto.Proposal_State{
		proto.Proposal_STATE_OPEN,
		proto.Proposal_STATE_DECLINED,
		proto.Proposal_STATE_FAILED,
		proto.Proposal_STATE_OPEN,
		proto.Proposal_STATE_PASSED,
		proto.Proposal_STATE_ENACTED,
		proto.Proposal_STATE_REJECTED,
		proto.Proposal_STATE_REJECTED,
	}
	expNr := map[proto.Proposal_State]int{
		proto.Proposal_STATE_OPEN:     2,
		proto.Proposal_STATE_DECLINED: 1,
		proto.Proposal_STATE_FAILED:   1,
		proto.Proposal_STATE_PASSED:   1,
		proto.Proposal_STATE_ENACTED:  1,
		proto.Proposal_STATE_REJECTED: 2,
	}
	for i, s := range states {
		prop := proto.Proposal{
			PartyId: party,
			Id:      fmt.Sprintf("test-prop-%d", i),
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
		prop := proto.Proposal{
			PartyId: party,
			Id:      id,
			State:   proto.Proposal_STATE_OPEN,
		}
		sub.Push(events.NewProposalEvent(ctx, prop))
	}
	sub.Push(events.NewProposalEvent(ctx, proto.Proposal{
		PartyId: "some-other-party",
		Id:      "foobar",
		State:   proto.Proposal_STATE_OPEN,
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
		sub.Push(events.NewProposalEvent(ctx, proto.Proposal{
			PartyId: p,
			Id:      props[i],
			State:   proto.Proposal_STATE_OPEN,
		}))
	}
	for _, p := range parties {
		for i := range props {
			sub.Push(events.NewVoteEvent(ctx, types.Vote{
				ProposalID: props[i],
				PartyID:    p,
				Value:      proto.Vote_VALUE_NO,
			}))
			sub.Push(events.NewVoteEvent(ctx, types.Vote{
				ProposalID: props[i],
				PartyID:    p,
				Value:      proto.Vote_VALUE_YES,
			}))
			if i > 1 {
				sub.Push(events.NewVoteEvent(ctx, types.Vote{
					ProposalID: props[i],
					PartyID:    p,
					Value:      proto.Vote_VALUE_YES,
				}))
				sub.Push(events.NewVoteEvent(ctx, types.Vote{
					ProposalID: props[i],
					PartyID:    p,
					Value:      proto.Vote_VALUE_NO,
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
