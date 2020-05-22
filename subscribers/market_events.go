package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
)

type ME interface {
	events.Event
	MarketEvent() string
}

type MarketEvent struct {
	ctx     context.Context
	sCh     chan struct{}
	ch      chan events.Event
	running bool
	log     *logging.Logger
}

func NewMarketEvent(ctx context.Context, log *logging.Logger) *MarketEvent {
	m := &MarketEvent{
		ctx:     ctx,
		sCh:     make(chan struct{}),
		ch:      make(chan events.Event, 10), // the size of the buffer can be tweaked, maybe use config?
		running: true,
		log:     log,
	}
	go m.loop()
	return m
}

func (m *MarketEvent) loop() {
	// add this call to at least close the pause channel
	// a destructor would've been nice to remove the data channel
	// but without explicit de-register calls, we can't
	defer func() {
		m.Pause()
		close(m.ch)
	}()
	for {
		select {
		case <-m.ctx.Done():
			return
		case e := <-m.ch:
			if m.running {
				if me, ok := e.(ME); ok {
					m.write(me)
				}
			}
		}
	}
}

func (m *MarketEvent) Pause() {
	if m.running {
		m.running = false
		close(m.sCh)
	}
}

func (m *MarketEvent) Resume() {
	if !m.running {
		m.sCh = make(chan struct{})
		m.running = true
	}
}

func (m *MarketEvent) Skip() <-chan struct{} {
	return m.sCh
}

func (m *MarketEvent) Closed() <-chan struct{} {
	return m.ctx.Done()
}

func (m *MarketEvent) C() chan<- events.Event {
	return m.ch
}

func (m *MarketEvent) Push(e events.Event) {
	me, ok := e.(ME)
	if !ok {
		return
	}
	m.write(me)
}

// this function will be replaced - this is where the events will be normalised for a market event plugin to use
func (m *MarketEvent) write(e ME) {
	m.log.Debug("MARKET EVENT",
		logging.String("trace-id", e.TraceID()),
		logging.String("type", e.Type().String()),
		logging.String("event", e.MarketEvent()),
	)
}

func (m *MarketEvent) Types() []events.Type {
	return events.MarketEvents()
}
