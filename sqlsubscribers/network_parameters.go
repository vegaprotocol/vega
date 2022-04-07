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

type NetworkParameterEvent interface {
	events.Event
	NetworkParameter() vega.NetworkParameter
}

type NetworkParameterStore interface {
	Add(context.Context, entities.NetworkParameter) error
}

type NetworkParameter struct {
	store    NetworkParameterStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewNetworkParameter(
	store NetworkParameterStore,
	log *logging.Logger,
) *NetworkParameter {
	np := &NetworkParameter{
		store: store,
		log:   log,
	}
	return np
}

func (n *NetworkParameter) Types() []events.Type {
	return []events.Type{events.NetworkParameterEvent}
}

func (n *NetworkParameter) Push(ctx context.Context, evt events.Event) error {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		n.vegaTime = event.Time()
	case NetworkParameterEvent:
		return n.consume(ctx, event)
	default:
		return errors.Errorf("unknown event type %s", event.Type().String())
	}

	return nil
}

func (n *NetworkParameter) consume(ctx context.Context, event NetworkParameterEvent) error {
	pnp := event.NetworkParameter()
	np, err := entities.NetworkParameterFromProto(&pnp)
	if err != nil {
		return errors.Wrap(err, "unable to parse network parameter")
	}
	np.VegaTime = n.vegaTime

	return errors.Wrap(n.store.Add(ctx, np), "error adding networkParameter")
}
