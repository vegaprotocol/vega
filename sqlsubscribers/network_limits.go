package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type NetworkLimitsEvent interface {
	events.Event
	NetworkLimits() *vega.NetworkLimits
}

type NetworkLimitStore interface {
	Add(context.Context, entities.NetworkLimits) error
}

type NetworkLimits struct {
	subscriber
	store NetworkLimitStore
	log   *logging.Logger
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

func (t *NetworkLimits) Types() []events.Type {
	return []events.Type{events.NetworkLimitsEvent}
}

func (nl *NetworkLimits) Push(ctx context.Context, evt events.Event) error {
	return nl.consume(ctx, evt.(NetworkLimitsEvent))
}

func (nl *NetworkLimits) consume(ctx context.Context, event NetworkLimitsEvent) error {
	protoLimits := event.NetworkLimits()
	limits := entities.NetworkLimitsFromProto(protoLimits)
	limits.VegaTime = nl.vegaTime

	return errors.Wrap(nl.store.Add(ctx, limits), "error adding network limits")
}
