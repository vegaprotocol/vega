package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	ReferralSetCreatedEvent interface {
		events.Event
		GetReferralSetCreated() *eventspb.ReferralSetCreated
	}

	RefereeJoinedReferralSetEvent interface {
		events.Event
		GetRefereeJoinedReferralSet() *eventspb.RefereeJoinedReferralSet
	}

	ReferralSetsStore interface {
		AddReferralSet(ctx context.Context, referralSet *entities.ReferralSet) error
		RefereeJoinedReferralSet(ctx context.Context, referee *entities.ReferralSetReferee) error
	}

	ReferralSets struct {
		subscriber
		store ReferralSetsStore
	}
)

func NewReferralSets(store ReferralSetsStore) *ReferralSets {
	return &ReferralSets{
		store: store,
	}
}

func (rs *ReferralSets) Types() []events.Type {
	return []events.Type{
		events.ReferralSetCreatedEvent,
		events.RefereeJoinedReferralSetEvent,
	}
}

func (rs *ReferralSets) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case ReferralSetCreatedEvent:
		return rs.consumeReferralSetCreatedEvent(ctx, e)
	case RefereeJoinedReferralSetEvent:
		return rs.consumeRefereeJoinedReferralSetEvent(ctx, e)
	default:
		return nil
	}
}

func (rs *ReferralSets) consumeReferralSetCreatedEvent(ctx context.Context, e ReferralSetCreatedEvent) error {
	referralSet := entities.ReferralSetFromProto(e.GetReferralSetCreated(), rs.vegaTime)
	return rs.store.AddReferralSet(ctx, referralSet)
}

func (rs *ReferralSets) consumeRefereeJoinedReferralSetEvent(ctx context.Context, e RefereeJoinedReferralSetEvent) error {
	referralSetReferee := entities.ReferralSetRefereeFromProto(e.GetRefereeJoinedReferralSet(), rs.vegaTime)
	return rs.store.RefereeJoinedReferralSet(ctx, referralSetReferee)
}
