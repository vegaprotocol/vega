package sqlsubscribers

import (
	"context"

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
	subscriber
	store EpochStore
	log   *logging.Logger
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

func (es *Epoch) Push(ctx context.Context, evt events.Event) error {
	return es.consume(ctx, evt.(EpochUpdateEvent))
}

func (es *Epoch) consume(ctx context.Context, event EpochUpdateEvent) error {
	epochUpdateEvent := event.Proto()
	epoch := entities.EpochFromProto(epochUpdateEvent)
	epoch.VegaTime = es.vegaTime

	return errors.Wrap(es.store.Add(ctx, epoch), "error adding epoch update")
}
