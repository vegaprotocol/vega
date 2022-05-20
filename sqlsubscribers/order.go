package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type OrderEvent interface {
	events.Event
	Order() *vega.Order
}

type OrderStore interface {
	Add(entities.Order) error
	Flush(ctx context.Context) error
}

type Order struct {
	subscriber
	store OrderStore
	log   *logging.Logger
}

func NewOrder(store OrderStore, log *logging.Logger) *Order {
	return &Order{
		store: store,
		log:   log,
	}
}

func (os *Order) Types() []events.Type {
	return []events.Type{events.OrderEvent}
}

func (os *Order) Push(ctx context.Context, evt events.Event) error {
	return os.consume(evt.(OrderEvent), evt.Sequence())
}

func (os *Order) Flush(ctx context.Context) error {
	return os.store.Flush(ctx)
}

func (os *Order) consume(oe OrderEvent, seqNum uint64) error {
	protoOrder := oe.Order()

	order, err := entities.OrderFromProto(protoOrder, seqNum)
	if err != nil {
		return errors.Wrap(err, "deserializing order")
	}
	order.VegaTime = os.vegaTime

	return errors.Wrap(os.store.Add(order), "adding order to database")
}
