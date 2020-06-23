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

// PartyID - for combined subscriber, communal interface
func (p *Proposal) PartyID() string {
	return p.p.PartyID
}
