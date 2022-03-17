package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
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

func (es *Epoch) Type() events.Type {
	return events.EpochUpdate
}

func (es *Epoch) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		es.vegaTime = event.Time()
	case EpochUpdateEvent:
		es.consume(event)
	default:
		es.log.Panic("Unknown event type in epoch update subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (es *Epoch) consume(event EpochUpdateEvent) {
	es.log.Debug("Epoch: ", logging.Int64("block", event.BlockNr()))
	epochUpdateEvent := event.Proto()
	epoch := entities.EpochFromProto(epochUpdateEvent)
	epoch.VegaTime = es.vegaTime

	if err := es.store.Add(context.Background(), epoch); err != nil {
		es.log.Error("Error adding epoch update", logging.Error(err))
	}
}
