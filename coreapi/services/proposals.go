package services

import (
	"context"
	"sync"

	vegapb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"
)

type proposalE interface {
	events.Event
	Proposal() vegapb.Proposal
}

type Proposals struct {
	*subscribers.Base
	ctx context.Context

	mu        sync.RWMutex
	proposals map[string]vegapb.Proposal
	// map of proposer -> set of proposal id
	proposalsPerProposer map[string]map[string]struct{}
	ch                   chan vegapb.Proposal
}

func NewProposals(ctx context.Context) (proposals *Proposals) {
	defer func() { go proposals.consume() }()
	return &Proposals{
		Base:                 subscribers.NewBase(ctx, 1000, true),
		ctx:                  ctx,
		proposals:            map[string]vegapb.Proposal{},
		proposalsPerProposer: map[string]map[string]struct{}{},
		ch:                   make(chan vegapb.Proposal, 100),
	}
}

func (p *Proposals) consume() {
	defer func() { close(p.ch) }()
	for {
		select {
		case <-p.Closed():
			return
		case prop, ok := <-p.ch:
			if !ok {
				// cleanup base
				p.Halt()
				// channel is closed
				return
			}
			p.mu.Lock()
			p.proposals[prop.Id] = prop
			proposals, ok := p.proposalsPerProposer[prop.PartyId]
			if !ok {
				proposals = map[string]struct{}{}
				p.proposalsPerProposer[prop.PartyId] = proposals
			}
			proposals[prop.Id] = struct{}{}
			p.mu.Unlock()
		}
	}
}

func (p *Proposals) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(proposalE); ok {
			p.ch <- ae.Proposal()
		}
	}
}

func (p *Proposals) List(proposal, party string) []*vegapb.Proposal {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(proposal) <= 0 && len(party) <= 0 {
		return p.getAllProposals()
	} else if len(party) > 0 {
		return p.getProposalsPerParty(proposal, party)
	}
	return p.getAllProposals()
}

func (p *Proposals) getProposalsPerParty(proposal, party string) []*vegapb.Proposal {
	out := []*vegapb.Proposal{}
	partyProposals, ok := p.proposalsPerProposer[party]
	if !ok {
		return out
	}

	if len(proposal) > 0 {
		_, ok := partyProposals[proposal]
		if ok {
			prop := p.proposals[proposal]
			out = append(out, &prop)
		}
		return out
	}

	for k, _ := range partyProposals {
		prop := p.proposals[k]
		out = append(out, &prop)
	}
	return out
}

func (p *Proposals) getProposalByID(proposal string) []*vegapb.Proposal {
	out := []*vegapb.Proposal{}
	asset, ok := p.proposals[proposal]
	if ok {
		out = append(out, &asset)
	}
	return out
}

func (p *Proposals) getAllProposals() []*vegapb.Proposal {
	out := make([]*vegapb.Proposal, 0, len(p.proposals))
	for _, v := range p.proposals {
		v := v
		out = append(out, &v)
	}
	return out
}

func (p *Proposals) Types() []events.Type {
	return []events.Type{
		events.ProposalEvent,
	}
}
