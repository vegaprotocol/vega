package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
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

func (p *PartySub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.Halt()
			return
		case e := <-p.ch:
			if p.isRunning() {
				p.Push(e...)
			}
		}
	}
}

func (p *PartySub) Push(evts ...events.Event) {
	for _, e := range evts {
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
}

func (*PartySub) Types() []events.Type {
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
