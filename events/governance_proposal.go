package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Proposal struct {
	*Base
	p types.Proposal
}

func NewProposalEvent(ctx context.Context, p types.Proposal) *Proposal {
	return &Proposal{
		Base: newBase(ctx, ProposalEvent),
		p:    p,
	}
}

func (p *Proposal) Proposal() types.Proposal {
	return p.p
}

// ProposalID - for combined subscriber, communal interface
func (p *Proposal) ProposalID() string {
	return p.p.ID
}

func (p Proposal) IsParty(id string) bool {
	return (p.p.PartyID == id)
}

// PartyID - for combined subscriber, communal interface
func (p *Proposal) PartyID() string {
	return p.p.PartyID
}

func (p Proposal) Proto() types.Proposal {
	return p.p
}

func (p Proposal) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:   p.traceID,
		Type: p.et.ToProto(),
		Event: &types.BusEvent_Proposal{
			Proposal: &p.p,
		},
	}
}
