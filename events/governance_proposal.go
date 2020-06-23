package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Porposal struct {
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
