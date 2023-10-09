// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package services

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/subscribers"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
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
	} else if len(proposal) > 0 {
		return p.getProposalByID(proposal)
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

	for k := range partyProposals {
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
