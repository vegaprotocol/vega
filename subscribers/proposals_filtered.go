package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type ProposalFilteredSub struct {
	*Base
	filters []ProposalFilter
	matched []types.Proposal
}

// ByProposalID - filter proposal events by proposal ID
func ProposalByID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.ID == id {
			return true
		}
		return false
	}
}

// ProposalByPartyID - filter proposals submitted by given party
func ProposalByPartyID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.PartyID == id {
			return true
		}
		return false
	}
}

// ProposalByState - filter proposals by state
func ProposalByState(s types.Proposal_State) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.State == s {
			return true
		}
		return false
	}
}

func NewProposalFilteredSub(ctx context.Context, filters ...ProposalFilter) *ProposalFilteredSub {
	p := ProposalFilteredSub{
		Base:    newBase(ctx, 10),
		filters: filters,
		matched: []types.Proposal{},
	}
	p.running = true
	go p.loop(p.ctx)
	return &p
}

func (p *ProposalFilteredSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.Halt()
			return
		case e := <-p.ch:
			if p.isRunning() {
				p.Push(e)
			}
		}
	}
}

func (p *ProposalFilteredSub) Push(e events.Event) {
	switch et := e.(type) {
	case TimeEvent:
		p.Flush()
	case PropE:
		prop := et.Proposal()
		for _, f := range p.filters {
			if !f(prop) {
				return
			}
		}
		p.matched = append(p.matched, prop)
	}
}

func (p *ProposalFilteredSub) Flush() {
	p.matched = make([]types.Proposal, 0, cap(p.matched))
}

func (p ProposalFilteredSub) Types() []events.Type {
	return []events.Type{
		events.ProposalEvent,
		events.TimeUpdate,
	}
}
