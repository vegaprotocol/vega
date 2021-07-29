package subscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
)

// NEE - MarketUpdatedEvent
type MEE interface {
	Proto() types.Market
}

type MarketUpdated struct {
	*Base
	store MarketStore
	log   *logging.Logger
}

func NewMarketUpdatedSub(ctx context.Context, store MarketStore, log *logging.Logger, ack bool) *MarketUpdated {
	m := &MarketUpdated{
		Base:  NewBase(ctx, 1, ack),
		store: store,
		log:   log,
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
		switch et := e.(type) {
		case MEE:
			batch = append(batch, et.Proto())
		default:
			m.log.Panic("Unknown event type in market updated subscriber", logging.String("Type", et.Type().String()))
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
