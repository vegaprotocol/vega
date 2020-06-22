package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
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
	buf   map[string]map[string]types.MarginLevels
}

func NewMarginLevelSub(ctx context.Context, store Store) *MarginLevelSub {
	m := MarginLevelSub{
		Base:  newBase(ctx, 10),
		store: store,
		buf:   map[string]map[string]types.MarginLevels{},
	}
	m.running = true
	go m.loop(m.ctx)
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
				m.Push(e)
			}
		}
	}
}

func (m *MarginLevelSub) Push(e events.Event) {
	switch te := e.(type) {
	case MLE:
		ml := te.MarginLevels()
		if _, ok := m.buf[ml.PartyID]; !ok {
			m.buf[ml.PartyID] = map[string]types.MarginLevels{}
		}
		m.buf[ml.PartyID][ml.MarketID] = ml
	case TimeEvent:
		m.flush()
	}
}

func (m *MarginLevelSub) flush() {
	buf := m.buf
	m.buf = map[string]map[string]types.MarginLevels{}
	batch := make([]types.MarginLevels, 0, len(buf))
	for _, mm := range buf {
		for _, ml := range mm {
			batch = append(batch, ml)
		}
	}
	m.store.SaveMarginLevelsBatch(batch)
}

func (t *MarginLevelSub) Types() []events.Type {
	return []events.Type{
		events.MarginLevelsEvent,
		events.TimeUpdate,
	}
}
