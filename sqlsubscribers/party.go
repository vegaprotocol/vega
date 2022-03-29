package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
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

func (ps *Party) Type() events.Type {
	return events.PartyEvent
}

func (ps *Party) Push(evt events.Event) {
	switch event := evt.(type) {
	case TimeUpdateEvent:
		ps.vegaTime = event.Time()
	case PartyEvent:
		ps.consume(event)
	default:
		ps.log.Panic("unknown event type in party subscriber",
			logging.String("Type", event.Type().String()))
	}
}

func (ps *Party) consume(event PartyEvent) {
	pp := event.Party()
	p := entities.PartyFromProto(&pp)
	vt := ps.vegaTime
	p.VegaTime = &vt

	if err := ps.store.Add(context.Background(), p); err != nil {
		ps.log.Error("error adding party", logging.Error(err))
	}
}
