package events

import (
	"context"

	proto "code.vegaprotocol.io/data-node/proto/vega"
	eventspb "code.vegaprotocol.io/data-node/proto/vega/events/v1"
	"code.vegaprotocol.io/data-node/types"
)

type Proposal struct {
	*Base
	p proto.Proposal
}

func NewProposalEvent(ctx context.Context, p types.Proposal) *Proposal {
	return &Proposal{
		Base: newBase(ctx, ProposalEvent),
		p:    *p.IntoProto(),
	}
}

func (p *Proposal) Proposal() proto.Proposal {
	return p.p
}

// ProposalID - for combined subscriber, communal interface
func (p *Proposal) ProposalID() string {
	return p.p.Id
}

func (p Proposal) IsParty(id string) bool {
	return p.p.PartyId == id
}

// PartyID - for combined subscriber, communal interface
func (p *Proposal) PartyID() string {
	return p.p.PartyId
}

func (p Proposal) Proto() proto.Proposal {
	return p.p
}

func (p Proposal) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    p.eventID(),
		Block: p.TraceID(),
		Type:  p.et.ToProto(),
		Event: &eventspb.BusEvent_Proposal{
			Proposal: &p.p,
		},
	}
}
