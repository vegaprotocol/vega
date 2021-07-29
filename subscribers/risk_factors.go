package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
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
	log   *logging.Logger
}

func NewRiskFactorSub(ctx context.Context, store RFStore, log *logging.Logger, ack bool) *RiskFactorSub {
	m := RiskFactorSub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		buf:   map[string]types.RiskFactor{},
		log:   log,
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
		switch et := e.(type) {
		case RF:
			rf := et.RiskFactor()
			m.mu.Lock()
			m.buf[rf.Market] = rf
			m.mu.Unlock()
		case TimeEvent:
			m.flush()
		default:
			m.log.Panic("Unknown event type in risk factor subscriber", logging.String("Type", et.Type().String()))
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
