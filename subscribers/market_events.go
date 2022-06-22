// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package subscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
)

type ME interface {
	events.Event
	MarketEvent() string
}

type MarketEvent struct {
	*Base
	cfg Config
	log *logging.Logger
}

func NewMarketEvent(ctx context.Context, cfg Config, log *logging.Logger, ack bool) *MarketEvent {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.MarketEventLogLevel.Level)
	m := &MarketEvent{
		Base: NewBase(ctx, 10, ack), // the size of the buffer can be tweaked, maybe use config?
		log:  log,
		cfg:  cfg,
	}
	if m.isRunning() {
		go m.loop()
	}
	return m
}

func (m *MarketEvent) loop() {
	for {
		select {
		case <-m.ctx.Done():
			m.Halt()
			return
		case e := <-m.ch:
			if m.isRunning() {
				m.Push(e...)
			}
		}
	}
}

func (m *MarketEvent) Push(evts ...events.Event) {
	for _, e := range evts {
		me, ok := e.(ME)
		if !ok {
			return
		}
		m.write(me)
	}
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
