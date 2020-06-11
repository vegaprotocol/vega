package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type PE interface {
	events.Event
	Party() types.Party
}

type PartyStore interface {
	SaveBatch(order []types.Party) error
}

type PartySub struct {
	*Base
	store PartyStore
	buf   []types.Party
}

func NewPartySub(ctx context.Context, store PartyStore) *PartySub {
	a := &PartySub{
		Base:  newBase(ctx, 10),
		store: store,
		buf:   []types.Party{},
	}
	a.running = true
	go a.loop(a.ctx)
	return a
}

func (a *PartySub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			a.Halt()
			return
		case e := <-a.ch:
			if a.running {
				a.Push(e)
			}
		}
	}
}

func (p *PartySub) Push(e events.Event) {
	switch et := e.(type) {
	case PE:
		party := et.Party()
		p.buf = append(p.buf, party)
	case TimeEvent:
		p.flush()
	}
}

func (p *PartySub) Types() []events.Type {
	return []events.Type{
		events.PartyEvent,
		events.TimeUpdate,
	}
}

func (p *PartySub) flush() {
	cpy := p.buf
	p.buf = make([]types.Party, 0, cap(cpy))
	_ = p.store.SaveBatch(cpy)
}
