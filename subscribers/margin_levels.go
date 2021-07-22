package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/data-node/proto/vega"
)

type MLE interface {
	MarginLevels() types.MarginLevels
}

type Store interface {
	SaveMarginLevelsBatch(batch []types.MarginLevels)
}

type MarginLevelSub struct {
	*Base
	store Store
	mu    sync.Mutex
	buf   map[string]map[string]types.MarginLevels
	log   *logging.Logger
}

func NewMarginLevelSub(ctx context.Context, store Store, log *logging.Logger, ack bool) *MarginLevelSub {
	m := MarginLevelSub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		buf:   map[string]map[string]types.MarginLevels{},
		log:   log,
	}
	if m.isRunning() {
		go m.loop(m.ctx)
	}
	return &m
}

func (m *MarginLevelSub) loop(ctx context.Context) {
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

func (m *MarginLevelSub) Push(evts ...events.Event) {
	for _, e := range evts {
		switch et := e.(type) {
		case MLE:
			ml := et.MarginLevels()
			m.mu.Lock()
			if _, ok := m.buf[ml.PartyId]; !ok {
				m.buf[ml.PartyId] = map[string]types.MarginLevels{}
			}
			m.buf[ml.PartyId][ml.MarketId] = ml
			m.mu.Unlock()
		case TimeEvent:
			m.flush()
		default:
			m.log.Panic("Unknown event type in margin level subscriber", logging.String("Type", et.Type().String()))
		}
	}
}

func (m *MarginLevelSub) flush() {
	m.log.Error("Margin level flush")
	m.mu.Lock()
	buf := m.buf
	m.buf = map[string]map[string]types.MarginLevels{}
	m.mu.Unlock()
	batch := make([]types.MarginLevels, 0, len(buf))
	for _, mm := range buf {
		for _, ml := range mm {
			batch = append(batch, ml)
		}
	}
	m.store.SaveMarginLevelsBatch(batch)
}

func (*MarginLevelSub) Types() []events.Type {
	return []events.Type{
		events.MarginLevelsEvent,
		events.TimeUpdate,
	}
}
