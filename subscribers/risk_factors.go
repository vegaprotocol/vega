package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type RF interface {
	events.Event
	RiskFactor() types.RiskFactor
}

type RFStore interface {
	SaveRiskFactorBatch(batch []types.RiskFactor)
}

type RiskFactorSub struct {
	*Base
	store RFStore
	mu    sync.Mutex
	buf   map[string]types.RiskFactor
}

func NewRiskFactorSub(ctx context.Context, store RFStore, ack bool) *RiskFactorSub {
	m := RiskFactorSub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		buf:   map[string]types.RiskFactor{},
	}
	if m.isRunning() {
		go m.loop(m.ctx)
	}
	return &m
}

func (m *RiskFactorSub) loop(ctx context.Context) {
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

func (m *RiskFactorSub) Push(evts ...events.Event) {
	for _, e := range evts {
		switch te := e.(type) {
		case RF:
			rf := te.RiskFactor()
			m.mu.Lock()
			m.buf[rf.Market] = rf
			m.mu.Unlock()
		case TimeEvent:
			m.flush()
		}
	}
}

func (m *RiskFactorSub) flush() {
	m.mu.Lock()
	buf := m.buf
	m.buf = map[string]types.RiskFactor{}
	m.mu.Unlock()
	batch := make([]types.RiskFactor, 0, len(buf))
	for _, rf := range buf {
		batch = append(batch, rf)
	}
	m.store.SaveRiskFactorBatch(batch)
}

func (*RiskFactorSub) Types() []events.Type {
	return []events.Type{
		events.RiskFactorEvent,
		events.TimeUpdate,
	}
}
