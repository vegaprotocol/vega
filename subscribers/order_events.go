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
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type OE interface {
	events.Event
	GetOrder() *types.Order
	VegaTime() time.Time
}

type OrderStore interface {
	SaveBatch([]types.Order) error
}

type OrderEvent struct {
	*Base
	mu    sync.Mutex
	cfg   Config
	log   *logging.Logger
	store OrderStore
	buf   []types.Order
}

func NewOrderEvent(ctx context.Context, cfg Config, log *logging.Logger, store OrderStore, ack bool) *OrderEvent {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.OrderEventLogLevel.Level)

	o := OrderEvent{
		Base:  NewBase(ctx, 10, ack),
		log:   log,
		store: store,
		buf:   []types.Order{},
		cfg:   cfg,
	}
	if o.isRunning() {
		go o.loop(o.ctx)
	}
	return &o
}

func (o *OrderEvent) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			o.Halt()
			return
		case e := <-o.ch:
			if o.isRunning() {
				o.Push(e...)
			}
		}
	}
}

func (o *OrderEvent) Push(evts ...events.Event) {
	for _, e := range evts {
		switch et := e.(type) {
		case OE:
			o.write(et)
		case TimeEvent:
			o.flush()
		default:
			o.log.Panic("Unknown event type in order subscriber", logging.String("Type", et.Type().String()))
		}
	}
}

// this function will be replaced - this is where the events will be normalised for a market event plugin to use
func (o *OrderEvent) write(e OE) {
	o.mu.Lock()
	o.buf = append(o.buf, *e.GetOrder())
	o.mu.Unlock()
	if o.log.GetLevel() <= logging.DebugLevel {
		o.log.Debug("ORDER EVENT",
			logging.String("trace-id", e.TraceID()),
			logging.String("type", e.Type().String()),
		)
	}
}

func (o *OrderEvent) flush() {
	o.mu.Lock()
	b := o.buf
	o.buf = make([]types.Order, 0, cap(b))
	o.mu.Unlock()
	if err := o.store.SaveBatch(b); err != nil {
		o.log.Error(
			"Failed to store batch of orders",
			logging.Error(err),
		)
	}
}

func (o *OrderEvent) Types() []events.Type {
	return []events.Type{
		events.OrderEvent,
		events.TimeUpdate,
	}
}
