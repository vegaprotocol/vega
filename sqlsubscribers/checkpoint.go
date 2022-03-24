package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
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

func (n *Checkpoint) Type() events.Type {
	return events.CheckpointEvent
}

func (n *Checkpoint) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		n.vegaTime = event.Time()
	case CheckpointEvent:
		n.consume(event)
	default:
		n.log.Panic("unknown event type in checkpoint subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (n *Checkpoint) consume(event CheckpointEvent) {
	pnp := event.Proto()
	np, err := entities.CheckpointFromProto(&pnp)
	if err != nil {
		n.log.Error("unable to parse checkpoint", logging.Error(err))
		return
	}
	np.VegaTime = n.vegaTime

	if err := n.store.Add(context.Background(), np); err != nil {
		n.log.Error("error adding checkpoint", logging.Error(err))
	}
}
