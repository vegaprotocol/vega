package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/pkg/errors"
)

var ErrNoSignaturesForID = errors.New("no signatures for id")

type NodeSignatureEvent interface {
	events.Event
	NodeSignature() commandspb.NodeSignature
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers NotaryStore
type NotaryStore interface {
	Add(context.Context, *entities.NodeSignature) error
}

type Notary struct {
	store NotaryStore
	log   *logging.Logger
}

func NewNotary(store NotaryStore, log *logging.Logger) *Notary {
	return &Notary{
		store: store,
		log:   log,
	}
}

func (w *Notary) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		return nil // not needed but the broker pushes time events to all subscribers
	case NodeSignatureEvent:
		return w.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type HERE %s", e.Type().String())
	}
}

func (w *Notary) consume(ctx context.Context, event NodeSignatureEvent) error {
	ns := event.NodeSignature()
	record, err := entities.NodeSignatureFromProto(&ns)
	if err != nil {
		return errors.Wrap(err, "converting node-signature proto to database entity failed")
	}

	return errors.Wrap(w.store.Add(ctx, record), "inserting node-signature to SQL store failed")
}

func (n *Notary) Types() []events.Type {
	return []events.Type{
		events.NodeSignatureEvent,
	}
}
