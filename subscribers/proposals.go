package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type ProposalType int

type PropE interface {
	GovernanceEvent
	Proposal() types.Proposal
}

const (
	NewMarketProposal ProposalType = iota
	NewAssetPropopsal
	UpdateMarketProposal
	UpdateNetworkProposal
)

type ProposalFilteredSub struct {
	*Base
	mu      sync.Mutex
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

// ProposalByReference - filter out proposals by reference
func ProposalByReference(ref string) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.Reference == ref {
			return true
		}
		return false
	}
}

func ProposalByChange(ptypes ...ProposalType) ProposalFilter {
	return func(p types.Proposal) bool {
		for _, t := range ptypes {
			switch t {
			case NewMarketProposal:
				if nm := p.Terms.GetNewMarket(); nm != nil {
					return true
				}
			case NewAssetPropopsal:
				if na := p.Terms.GetNewAsset(); na != nil {
					return true
				}
			case UpdateMarketProposal:
				if um := p.Terms.GetUpdateMarket(); um != nil {
					return true
				}
			case UpdateNetworkProposal:
				if un := p.Terms.GetUpdateNetwork(); un != nil {
					return true
				}
			}
		}
		return false
	}
}

func NewProposalFilteredSub(ctx context.Context, filters ...ProposalFilter) *ProposalFilteredSub {
	p := ProposalFilteredSub{
		Base:    NewBase(ctx, 10),
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
		p.mu.Lock()
		p.matched = append(p.matched, prop)
		p.mu.Unlock()
	}
}

func (p *ProposalFilteredSub) Flush() {
	p.mu.Lock()
	p.matched = make([]types.Proposal, 0, cap(p.matched))
	p.mu.Unlock()
}

func (p *ProposalFilteredSub) Types() []events.Type {
	return []events.Type{
		events.ProposalEvent,
		events.TimeUpdate,
	}
}
