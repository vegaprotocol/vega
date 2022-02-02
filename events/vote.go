package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type Vote struct {
	*Base
	v proto.Vote
}

func NewVoteEvent(ctx context.Context, v types.Vote) *Vote {
	return &Vote{
		Base: newBase(ctx, VoteEvent),
		v:    *v.IntoProto(),
	}
}

// Vote get the vote object.
func (v *Vote) Vote() proto.Vote {
	return v.v
}

// ProposalID get the proposal ID, part of the interface for event subscribers.
func (v *Vote) ProposalID() string {
	return v.v.ProposalId
}

// IsParty - used in event stream API filter.
func (v Vote) IsParty(id string) bool {
	return v.v.PartyId == id
}

// PartyID - return the PartyID for subscribers' convenience.
func (v *Vote) PartyID() string {
	return v.v.PartyId
}

// Value - return a Y/N value, makes subscribers easier to implement.
func (v *Vote) Value() proto.Vote_Value {
	return v.v.Value
}

// TotalGovernanceTokenBalance returns the total balance of token used for this
// vote.
func (v *Vote) TotalGovernanceTokenBalance() string {
	return v.v.TotalGovernanceTokenBalance
}

// TotalGovernanceTokenWeight returns the total weight of token used for this
// vote.
func (v *Vote) TotalGovernanceTokenWeight() string {
	return v.v.TotalGovernanceTokenWeight
}

func (v Vote) Proto() proto.Vote {
	return v.v
}

func (v Vote) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(v.Base)
	busEvent.Event = &eventspb.BusEvent_Vote{
		Vote: &v.v,
	}

	return busEvent
}

func VoteEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Vote {
	return &Vote{
		Base: newBaseFromBusEvent(ctx, VoteEvent, be),
		v:    *be.GetVote(),
	}
}
