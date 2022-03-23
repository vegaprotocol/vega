package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type OrderEvent interface {
	events.Event
	Order() *vega.Order
}

type OrderStore interface {
	Add(context.Context, entities.Order) error
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

func (os *Order) Type() events.Type {
	return events.OrderEvent
}

func (os *Order) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		os.vegaTime = e.Time()
	case OrderEvent:
		os.consume(e)
	default:
		os.log.Panic("Unknown event type in order subscriber",
			logging.String("Type", e.Type().String()))
	}
}

func (os *Order) consume(oe OrderEvent) {
	protoOrder := oe.Order()

	order, err := entities.OrderFromProto(protoOrder)
	if err != nil {
		os.log.Errorf("deserializing order: %v", err)
		return
	}
	order.VegaTime = os.vegaTime
	err = os.store.Add(context.Background(), order)
	if err != nil {
		os.log.Errorf("adding order to database: %v", err)
		return
	}
}
