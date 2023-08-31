package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ReferralSetCreated struct {
	*Base
	e eventspb.ReferralSetCreated
}

func (t ReferralSetCreated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralSetCreated{
		ReferralSetCreated: &t.e,
	}

	return busEvent
}

func NewReferralSetCreatedEvent(ctx context.Context, set *types.ReferralSet) *ReferralSetCreated {
	return &ReferralSetCreated{
		Base: newBase(ctx, ReferralSetCreatedEvent),
		e: eventspb.ReferralSetCreated{
			SetId:     string(set.ID),
			Referrer:  string(set.Referrer.PartyID),
			CreatedAt: set.CreatedAt.UnixNano(),
			UpdatedAt: set.CreatedAt.UnixNano(),
		},
	}
}

func ReferralSetCreatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralSetCreated {
	return &ReferralSetCreated{
		Base: newBaseFromBusEvent(ctx, ReferralSetCreatedEvent, be),
		e:    *be.GetReferralSetCreated(),
	}
}

type RefereeJoinedReferralSet struct {
	*Base
	e eventspb.RefereeJoinedReferralSet
}

func (t RefereeJoinedReferralSet) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_RefereeJoinedReferralSet{
		RefereeJoinedReferralSet: &t.e,
	}

	return busEvent
}

func NewRefereeJoinedReferralSetEvent(ctx context.Context, setID types.ReferralSetID, membership *types.Membership) *RefereeJoinedReferralSet {
	return &RefereeJoinedReferralSet{
		Base: newBase(ctx, RefereeJoinedReferralSetEvent),
		e: eventspb.RefereeJoinedReferralSet{
			SetId:    string(setID),
			Referee:  string(membership.PartyID),
			JoinedAt: membership.JoinedAt.UnixNano(),
			AtEpoch:  membership.StartedAtEpoch,
		},
	}
}

func RefereeJoinedReferralSetEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RefereeJoinedReferralSet {
	return &RefereeJoinedReferralSet{
		Base: newBaseFromBusEvent(ctx, RefereeJoinedReferralSetEvent, be),
		e:    *be.GetRefereeJoinedReferralSet(),
	}
}
