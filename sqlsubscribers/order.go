package sqlsubscribers

import (
	"context"
	"time"

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
	Add(context.Context, entities.Order) error
	Flush(ctx context.Context) error
}

type Order struct {
	store    OrderStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewOrder(ctx context.Context, store OrderStore, blockStore BlockStore, log *logging.Logger) *Order {
	return &Order{
		store: store,
		log:   log,
	}
}

func (os *Order) Types() []events.Type {
	return []events.Type{events.OrderEvent}
}

func (os *Order) Push(evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		os.vegaTime = e.Time()
		return os.store.Flush(context.Background())
	case OrderEvent:
		return os.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
}

func (os *Order) consume(oe OrderEvent) error {
	protoOrder := oe.Order()

	order, err := entities.OrderFromProto(protoOrder)
	if err != nil {
		return errors.Wrap(err, "deserializing order")
	}
	order.VegaTime = os.vegaTime

	return errors.Wrap(os.store.Add(context.Background(), order), "adding order to database")
}
