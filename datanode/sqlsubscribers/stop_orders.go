package sqlsubscribers

import (
	"context"

	pbevents "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
)

type (
	StopOrderEvent interface {
		events.Event
		StopOrder() *pbevents.StopOrderEvent
	}

	StopOrderStore interface {
		Add(entities.StopOrder) error
		Flush(ctx context.Context) error
	}

	StopOrder struct {
		subscriber
		store StopOrderStore
	}
)

func NewStopOrder(store StopOrderStore) *StopOrder {
	return &StopOrder{
		store: store,
	}
}

func (so *StopOrder) Types() []events.Type {
	return []events.Type{
		events.StopOrderEvent,
	}
}

func (so *StopOrder) Push(ctx context.Context, evt events.Event) error {
	return so.consume(evt.(StopOrderEvent), evt.Sequence())
}

func (so *StopOrder) Flush(ctx context.Context) error {
	return so.store.Flush(ctx)
}

func (so *StopOrder) consume(evt StopOrderEvent, seqNum uint64) error {
	protoOrder := evt.StopOrder()
	stop, err := entities.StopOrderFromProto(protoOrder, so.vegaTime, seqNum, entities.TxHash(evt.TxHash()))
	if err != nil {
		return errors.Wrap(err, "deserializing stop order")
	}
	return errors.Wrap(so.store.Add(stop), "adding stop order")
}
