package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
)

type Storage interface {
	SaveBatch([]proto.MarketData)
}

type MDE interface {
	events.Event
	MarketData() proto.MarketData
}

type MarketDataSub struct {
	*Base
	mu    sync.Mutex
	buf   []proto.MarketData
	store Storage
}

func NewMarketDataSub(ctx context.Context, store Storage, ack bool) *MarketDataSub {
	md := &MarketDataSub{
		Base:  NewBase(ctx, 10, ack),
		buf:   []proto.MarketData{},
		store: store,
	}
	if md.isRunning() {
		go md.loop(md.ctx)
	}
	return md
}

func (m *MarketDataSub) loop(ctx context.Context) {
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

func (m *MarketDataSub) Push(evts ...events.Event) {
	for _, e := range evts {
		switch te := e.(type) {
		case MDE:
			md := te.MarketData()
			m.mu.Lock()
			m.buf = append(m.buf, md)
			m.mu.Unlock()
		case TimeEvent:
			m.flush()
		}
	}
}

func (m *MarketDataSub) flush() {
	m.mu.Lock()
	if len(m.buf) == 0 {
		m.mu.Unlock()
		return
	}
	data := m.buf
	m.buf = make([]proto.MarketData, 0, cap(data))
	m.mu.Unlock()
	m.store.SaveBatch(data)
}

func (m *MarketDataSub) Types() []events.Type {
	return []events.Type{
		events.MarketDataEvent,
		events.TimeUpdate,
	}
}
