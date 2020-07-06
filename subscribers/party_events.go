package subscribers

import (
	"context"
	"sync"

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
	mu    sync.Mutex
	store PartyStore
	buf   []types.Party
}

func NewPartySub(ctx context.Context, store PartyStore, ack bool) *PartySub {
	a := &PartySub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		buf:   []types.Party{},
	}
	if a.isRunning() {
		go a.loop(a.ctx)
	}
	return a
}

func (a *PartySub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			a.Halt()
			return
		case e := <-a.ch:
			if a.isRunning() {
				a.Push(e)
			}
		}
	}
}

func (p *PartySub) Push(e events.Event) {
	switch et := e.(type) {
	case PE:
		party := et.Party()
		p.mu.Lock()
		p.buf = append(p.buf, party)
		p.mu.Unlock()
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
	p.mu.Lock()
	cpy := p.buf
	p.buf = make([]types.Party, 0, cap(cpy))
	p.mu.Unlock()
	_ = p.store.SaveBatch(cpy)
}
