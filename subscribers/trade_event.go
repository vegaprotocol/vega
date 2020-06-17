package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type TE interface {
	events.Event
	Trade() types.Trade
}

type TradeStore interface {
	SaveBatch([]types.Trade) error
}

type trades struct {
	*Base
	buf   []types.Trade
	store TradeStore
}

func NewTradeSub(ctx context.Context, store TradeStore) *trades {
	t := &trades{
		Base:  newBase(ctx, 10),
		buf:   []types.Trade{},
		store: store,
	}
	t.running = true
	go t.loop(t.ctx)
	return t
}

func (t *trades) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			t.Halt()
			return
		case e := <-t.ch:
			if t.running {
				t.Push(e)
			}
		}
	}
}

func (t *trades) Push(e events.Event) {
	switch te := e.(type) {
	case TE:
		t.write(te)
	case TimeEvent:
		t.flush()
	}
}

// this function will be replaced - this is where the events will be normalised for a market event plugin to use
func (t *trades) write(e TE) {
	t.buf = append(o.buf, e.Trade())
}

func (t *trades) flush() {
	b := t.buf
	t.buf = make([]types.Trade, 0, cap(b))
	_ = t.store.SaveBatch(b)
}

func (t *trades) Types() []events.Type {
	return []events.Type{
		events.TradeEvent,
		events.TimeUpdate,
	}
}
