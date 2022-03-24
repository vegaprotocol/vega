package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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

func (n *NetworkParameter) Type() events.Type {
	return events.NetworkParameterEvent
}

func (n *NetworkParameter) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		n.vegaTime = event.Time()
	case NetworkParameterEvent:
		n.consume(event)
	default:
		n.log.Panic("unknown event type in network parameter subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (n *NetworkParameter) consume(event NetworkParameterEvent) {
	pnp := event.NetworkParameter()
	np, err := entities.NetworkParameterFromProto(&pnp)
	if err != nil {
		n.log.Error("unable to parse network parameter", logging.Error(err))
		return
	}
	np.VegaTime = n.vegaTime

	if err := n.store.Add(context.Background(), np); err != nil {
		n.log.Error("error adding networkParameter", logging.Error(err))
	}
}
