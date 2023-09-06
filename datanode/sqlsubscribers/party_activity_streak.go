package sqlsubscribers

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
)

type (
	PartyActivityStreakEvent interface {
		events.Event
		PartyActivityStreak() *eventspb.PartyActivityStreak
	}
	PartyActivityStreakStore interface {
		Add(context.Context, *entities.PartyActivityStreak) error
	}
	PartyActivityStreak struct {
		subscriber
		store PartyActivityStreakStore
	}
)

func NewPartyActivityStreak(store PartyActivityStreakStore) *PartyActivityStreak {
	return &PartyActivityStreak{
		store: store,
	}
}

func (pas *PartyActivityStreak) Types() []events.Type {
	return []events.Type{
		events.PartyActivityStreakEvent,
	}
}

func (pas *PartyActivityStreak) Push(ctx context.Context, evt events.Event) error {
	switch evt.Type() {
	case events.PartyActivityStreakEvent:
		return pas.consumePartyActivityStreakEvent(ctx, evt.(PartyActivityStreakEvent))
	default:
		return nil
	}
}

func (pas *PartyActivityStreak) consumePartyActivityStreakEvent(ctx context.Context, evt PartyActivityStreakEvent) error {
	activityStreak := entities.NewPartyActivityStreakFromProto(evt.PartyActivityStreak(), entities.TxHash(evt.TxHash()), pas.vegaTime)

	return errors.Wrap(pas.store.Add(ctx, activityStreak), "adding party activity streak")
}
