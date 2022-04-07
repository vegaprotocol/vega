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

func (ds *Delegation) Types() []events.Type {
	return []events.Type{events.DelegationBalanceEvent}
}

func (ds *Delegation) Push(ctx context.Context, e events.Event) error {
	switch event := e.(type) {
	case TimeUpdateEvent:
		ds.vegaTime = event.Time()
	case DelegationBalanceEvent:
		return ds.consume(ctx, event)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (ds *Delegation) consume(ctx context.Context, event DelegationBalanceEvent) error {
	protoDBE := event.Proto()
	delegation, err := entities.DelegationFromProto(&protoDBE)
	if err != nil {
		return errors.Wrap(err, "unable to parse delegation")
	}

	delegation.VegaTime = ds.vegaTime

	if err := ds.store.Add(ctx, delegation); err != nil {
		return errors.Wrap(err, "error adding delegation")
	}

	return nil
}
