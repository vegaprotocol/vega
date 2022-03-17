package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type DelegationBalanceEvent interface {
	events.Event
	Proto() eventspb.DelegationBalanceEvent
}

type DelegationStore interface {
	Add(context.Context, entities.Delegation) error
}

type Delegation struct {
	store    DelegationStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewDelegation(
	store DelegationStore,
	log *logging.Logger,
) *Delegation {
	t := &Delegation{
		store: store,
		log:   log,
	}
	return t
}

func (ds *Delegation) Type() events.Type {
	return events.DelegationBalanceEvent
}

func (ds *Delegation) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		ds.vegaTime = event.Time()
	case DelegationBalanceEvent:
		ds.consume(event)
	default:
		ds.log.Panic("Unknown event type in delegation subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (ds *Delegation) consume(event DelegationBalanceEvent) {
	protoDBE := event.Proto()
	delegation, err := entities.DelegationFromProto(&protoDBE)
	if err != nil {
		ds.log.Error("unable to parse delegation", logging.Error(err))
	}

	delegation.VegaTime = ds.vegaTime

	if err := ds.store.Add(context.Background(), delegation); err != nil {
		ds.log.Error("Error adding delegation", logging.Error(err))
	}
}
