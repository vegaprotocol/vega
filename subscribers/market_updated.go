package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

// NEE - MarketUpdatedEvent
type MEE interface {
	Market() types.Market
}

type MarketUpdated struct {
	*Base
	store MarketStore
}

func NewMarketUpdatedSub(ctx context.Context, store MarketStore, ack bool) *MarketUpdated {
	m := &MarketUpdated{
		Base:  NewBase(ctx, 1, ack),
		store: store,
	}
	if m.isRunning() {
		go m.loop(m.ctx)
	}
	return m
}

func (m *MarketUpdated) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.Halt()
			return
		case e := <-m.ch:
			if m.isRunning() {
				m.Push(e...)
			}
		}
	}
}

func (m *MarketUpdated) Push(evts ...events.Event) {
	batch := make([]types.Market, 0, len(evts))
	for _, e := range evts {
		if te, ok := e.(MEE); ok {
			batch = append(batch, te.Market())
		}
	}
	if len(batch) > 0 {
		_ = m.store.SaveBatch(batch)
	}
}

func (m *MarketUpdated) Types() []events.Type {
	return []events.Type{
		events.MarketUpdatedEvent,
	}
}
