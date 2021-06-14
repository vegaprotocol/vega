package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	dtypes "code.vegaprotocol.io/vega/types"
)

type Vote struct {
	*Base
	v types.Vote
}

func NewVoteEvent(ctx context.Context, v *dtypes.Vote) *Vote {
	return &Vote{
		Base: newBase(ctx, VoteEvent),
		v:    *v.IntoProto(),
	}
}

// Vote get the vote object
func (v *Vote) Vote() types.Vote {
	return v.v
}

// ProposalID get the proposal ID, part of the interface for event subscribers
func (v *Vote) ProposalID() string {
	return v.v.ProposalId
}

// IsParty - used in event stream API filter
func (v Vote) IsParty(id string) bool {
	return v.v.PartyId == id
}

// PartyID - return the PartyID for subscribers' convenience
func (v *Vote) PartyID() string {
	return v.v.PartyId
}

// Value - return a Y/N value, makes subscribers easier to implement
func (v *Vote) Value() types.Vote_Value {
	return v.v.Value
}

// TotalGovernanceTokenBalance returns the total balance of token used for this
// vote
func (v *Vote) TotalGovernanceTokenBalance() uint64 {
	return v.v.TotalGovernanceTokenBalance
}

// TotalGovernanceTokenWeight returns the total weight of token used for this
// vote
func (v *Vote) TotalGovernanceTokenWeight() string {
	return v.v.TotalGovernanceTokenWeight
}

func (v Vote) Proto() types.Vote {
	return v.v
}

func (v Vote) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    v.eventID(),
		Block: v.TraceID(),
		Type:  v.et.ToProto(),
		Event: &eventspb.BusEvent_Vote{
			Vote: &v.v,
		},
	}
}
