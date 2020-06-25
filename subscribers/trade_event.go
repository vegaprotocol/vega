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

type TradeSub struct {
	*Base
	buf   []types.Trade
	store TradeStore
}

func NewTradeSub(ctx context.Context, store TradeStore) *TradeSub {
	t := &TradeSub{
		Base:  NewBase(ctx, 10),
		buf:   []types.Trade{},
		store: store,
	}
	t.running = true
	go t.loop(t.ctx)
	return t
}

func (t *TradeSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			t.Halt()
			return
		case e := <-t.ch:
			if t.isRunning() {
				t.Push(e)
			}
		}
	}
}

func (t *TradeSub) Push(e events.Event) {
	switch te := e.(type) {
	case TE:
		t.write(te)
	case TimeEvent:
		t.flush()
	}
}

// this function will be replaced - this is where the events will be normalised for a market event plugin to use
func (t *TradeSub) write(e TE) {
	t.buf = append(t.buf, e.Trade())
}

func (t *TradeSub) flush() {
	b := t.buf
	t.buf = make([]types.Trade, 0, cap(b))
	_ = t.store.SaveBatch(b)
}

func (t *TradeSub) Types() []events.Type {
	return []events.Type{
		events.TradeEvent,
		events.TimeUpdate,
	}
}
