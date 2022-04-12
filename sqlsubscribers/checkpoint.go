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

type CheckpointEvent interface {
	events.Event
	Proto() eventspb.CheckpointEvent
}

type CheckpointStore interface {
	Add(context.Context, entities.Checkpoint) error
}

type Checkpoint struct {
	store    CheckpointStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewCheckpoint(
	store CheckpointStore,
	log *logging.Logger,
) *Checkpoint {
	np := &Checkpoint{
		store: store,
		log:   log,
	}
	return np
}

func (n *Checkpoint) Types() []events.Type {
	return []events.Type{events.CheckpointEvent}
}

func (n *Checkpoint) Push(evt events.Event) error {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		n.vegaTime = event.Time()
	case CheckpointEvent:
		return n.consume(event)
	default:
		return errors.Errorf("unknown event type %s", event.Type().String())
	}

	return nil
}

func (n *Checkpoint) consume(event CheckpointEvent) error {
	pnp := event.Proto()
	np, err := entities.CheckpointFromProto(&pnp)
	if err != nil {
		return errors.Wrap(err, "unable to parse checkpoint")
	}
	np.VegaTime = n.vegaTime

	if err := n.store.Add(context.Background(), np); err != nil {
		return errors.Wrap(err, "error adding checkpoint")
	}

	return nil
}
