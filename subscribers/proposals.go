package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type PropE interface {
	GovernanceEvent
	Proposal() types.Proposal
}

type ProposalSub struct {
	*Base
	all  []types.Proposal
	byID map[string]types.Proposal
}

func NewProposalSub(ctx context.Context) *ProposalSub {
	p := ProposalSub{
		Base: newBase(ctx, 10),
		all:  []types.Proposal{},
		byID: map[string]types.Proposal{},
	}
	p.running = true
	go p.loop(p.ctx)
	return &p
}

func (p *ProposalSub) loop(ctx context.Context) {
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

func (p *ProposalSub) Push(e events.Event) {
	switch et := e.(type) {
	case TimeEvent:
		p.Flush()
	case PropE:
		prop := et.Proposal()
		p.all = append(p.all, prop)
		p.byID[prop.ID] = prop
	}
}

func (p *ProposalSub) Flush() {
	p.all = make([]types.Proposal, 0, cap(p.all))
	p.byID = make(map[string]types.Proposal, len(p.byID))
}

func (p ProposalSub) Types() []events.Type {
	return []events.Type{
		events.ProposalEvent,
		events.TimeEvent,
	}
}
