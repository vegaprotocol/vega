package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type Vote struct {
	*Base
	v types.Vote
}

func NewVoteEvent(ctx context.Context, v types.Vote) *Vote {
	return &Vote{
		Base: newBase(ctx, VoteEvent),
		v:    v,
	}
}

// Vote get the vote object
func (v *Vote) Vote() types.Vote {
	return v.v
}

// ProposalID get the proposal ID, part of the interface for event subscribers
func (v *Vote) ProposalID() string {
	return v.v.ProposalID
}

// IsParty - used in event stream API filter
func (v Vote) IsParty(id string) bool {
	return (v.v.PartyID == id)
}

// PartyID - return the PartyID for subscribers' convenience
func (v *Vote) PartyID() string {
	return v.v.PartyID
}

// Value - return a Y/N value, makes subscribers easier to implement
func (v *Vote) Value() types.Vote_Value {
	return v.v.Value
}

func (v Vote) Proto() types.Vote {
	return v.v
}

func (v Vote) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:    v.eventID(),
		Block: v.TraceID(),
		Type:  v.et.ToProto(),
		Event: &types.BusEvent_Vote{
			Vote: &v.v,
		},
	}
}
