package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type EpochUpdateEvent interface {
	events.Event
	Proto() eventspb.EpochEvent
}

type EpochStore interface {
	Add(context.Context, entities.Epoch) error
}

type Epoch struct {
	store    EpochStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewEpoch(
	store EpochStore,
	log *logging.Logger,
) *Epoch {
	t := &Epoch{
		store: store,
		log:   log,
	}
	return t
}

func (es *Epoch) Types() []events.Type {
	return []events.Type{events.EpochUpdate}
}

func (es *Epoch) Push(evt events.Event) error {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		es.vegaTime = event.Time()
	case EpochUpdateEvent:
		return es.consume(event)
	default:
		return errors.Errorf("unknown event type %s", evt.Type().String())
	}

	return nil
}

func (es *Epoch) consume(event EpochUpdateEvent) error {
	epochUpdateEvent := event.Proto()
	epoch := entities.EpochFromProto(epochUpdateEvent)
	epoch.VegaTime = es.vegaTime

	return errors.Wrap(es.store.Add(context.Background(), epoch), "error adding epoch update")
}
