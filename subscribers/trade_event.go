package subscribers

import (
	"context"
	"sync"

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
	mu    sync.Mutex
	buf   []types.Trade
	store TradeStore
}

func NewTradeSub(ctx context.Context, store TradeStore, ack bool) *TradeSub {
	t := &TradeSub{
		Base:  NewBase(ctx, 10, ack),
		buf:   []types.Trade{},
		store: store,
	}
	if t.isRunning() {
		go t.loop(t.ctx)
	}
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

func (t *TradeSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	// acquire lock here, so a time event doesn't result in a partial flush
	t.mu.Lock()
	for _, e := range evts {
		switch te := e.(type) {
		case TE:
			t.buf = append(t.buf, te.Trade())
		case TimeEvent:
			t.flush()
		}
	}
	t.mu.Unlock()
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
