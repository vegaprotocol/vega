package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type OE interface {
	events.Event
	Order() *types.Order
}

type OrderStore interface {
	SaveBatch([]types.Order) error
}

type OrderEvent struct {
	*Base
	log   *logging.Logger
	store OrderStore
	buf   []types.Order
}

func NewOrderEvent(ctx context.Context, log *logging.Logger, store OrderStore) *OrderEvent {
	o := OrderEvent{
		Base:  NewBase(ctx, 10),
		log:   log,
		store: store,
		buf:   []types.Order{},
	}
	o.running = true
	go o.loop(o.ctx)
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
				o.Push(e)
			}
		}
	}
}

func (o *OrderEvent) Push(e events.Event) {
	switch te := e.(type) {
	case OE:
		o.write(te)
	case TimeEvent:
		o.flush()
	}
}

// this function will be replaced - this is where the events will be normalised for a market event plugin to use
func (o *OrderEvent) write(e OE) {
	o.buf = append(o.buf, *e.Order())
	o.log.Debug("ORDER EVENT",
		logging.String("trace-id", e.TraceID()),
		logging.String("type", e.Type().String()),
		logging.Order(*e.Order()),
	)
}

func (o *OrderEvent) flush() {
	b := o.buf
	o.buf = make([]types.Order, 0, cap(b))
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
