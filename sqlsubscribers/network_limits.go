package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type NetworkLimitsEvent interface {
	events.Event
	NetworkLimits() *vega.NetworkLimits
}

type NetworkLimitStore interface {
	Add(context.Context, entities.NetworkLimits) error
}

type NetworkLimits struct {
	store    NetworkLimitStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewNetworkLimitSub(
	ctx context.Context,
	store NetworkLimitStore,
	log *logging.Logger,
) *NetworkLimits {
	t := &NetworkLimits{
		store: store,
		log:   log,
	}
	return t
}

func (t *NetworkLimits) Type() events.Type {
	return events.NetworkLimitsEvent
}

func (nl *NetworkLimits) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		nl.vegaTime = event.Time()
	case NetworkLimitsEvent:
		nl.consume(event)
	default:
		nl.log.Panic("Unknown event type in time subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (nl *NetworkLimits) consume(event NetworkLimitsEvent) {
	protoLimits := event.NetworkLimits()
	limits := entities.NetworkLimitsFromProto(protoLimits)
	limits.VegaTime = nl.vegaTime
	err := nl.store.Add(context.Background(), limits)
	if err != nil {
		nl.log.Error("Error adding network limits",
			logging.Error(err))
	}
}
