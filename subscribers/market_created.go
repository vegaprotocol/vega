package subscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
)

// NME - NewMarketEvent
type NME interface {
	Market() types.Market
}

type MarketStore interface {
	SaveBatch(markets []types.Market) error
}

type Market struct {
	*Base
	store MarketStore
	log   *logging.Logger
}

func NewMarketSub(ctx context.Context, store MarketStore, log *logging.Logger, ack bool) *Market {
	m := &Market{
		Base:  NewBase(ctx, 1, ack),
		store: store,
		log:   log,
	}
	if m.isRunning() {
		go m.loop(m.ctx)
	}
	return m
}

func (m *Market) loop(ctx context.Context) {
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

func (m *Market) Push(evts ...events.Event) {
	batch := make([]types.Market, 0, len(evts))
	for _, e := range evts {
		switch et := e.(type) {
		case NME:
			batch = append(batch, et.Market())
		default:
			m.log.Panic("Unknown event type in market subscriber", logging.String("Type", et.Type().String()))
		}
	}
	if len(batch) > 0 {
		_ = m.store.SaveBatch(batch)
	}
}

func (m *Market) Types() []events.Type {
	return []events.Type{
		events.MarketCreatedEvent,
	}
}
