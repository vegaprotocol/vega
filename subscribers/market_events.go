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
	*Base
	log *logging.Logger
}

func NewMarketEvent(ctx context.Context, log *logging.Logger) *MarketEvent {
	m := &MarketEvent{
		Base: newBase(ctx, 10), // the size of the buffer can be tweaked, maybe use config?
		log:  log,
	}
	m.running = true
	go m.loop()
	return m
}

func (m *MarketEvent) loop() {
	for {
		select {
		case <-m.ctx.Done():
			m.Halt()
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
