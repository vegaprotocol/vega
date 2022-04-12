package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"github.com/pkg/errors"
)

type PartyEvent interface {
	events.Event
	Party() types.Party
}

type PartyStore interface {
	Add(context.Context, entities.Party) error
}

type Party struct {
	store    PartyStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewParty(
	store PartyStore,
	log *logging.Logger,
) *Party {
	ps := &Party{
		store: store,
		log:   log,
	}
	return ps
}

func (ps *Party) Types() []events.Type {
	return []events.Type{events.PartyEvent}
}

func (ps *Party) Push(evt events.Event) error {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		ps.vegaTime = event.Time()
	case PartyEvent:
		return ps.consume(event)
	default:
		return errors.Errorf("unknown event type %s", event.Type().String())
	}

	return nil
}

func (ps *Party) consume(event PartyEvent) error {
	pp := event.Party()
	p := entities.PartyFromProto(&pp)
	vt := ps.vegaTime
	p.VegaTime = &vt

	return errors.Wrap(ps.store.Add(context.Background(), p), "error adding party:%w")
}
