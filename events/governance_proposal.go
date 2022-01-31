package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
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

// ProposalID - for combined subscriber, communal interface.
func (p *Proposal) ProposalID() string {
	return p.p.Id
}

func (p Proposal) IsParty(id string) bool {
	return p.p.PartyId == id
}

// PartyID - for combined subscriber, communal interface.
func (p *Proposal) PartyID() string {
	return p.p.PartyId
}

func (p Proposal) Proto() proto.Proposal {
	return p.p
}

func (p Proposal) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_Proposal{
		Proposal: &p.p,
	}
	return busEvent
}

func ProposalEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Proposal {
	return &Proposal{
		Base: newBaseFromBusEvent(ctx, ProposalEvent, be),
		p:    *be.GetProposal(),
	}
}
