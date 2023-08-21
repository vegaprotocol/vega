package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ReferralProgramStarted struct {
	*Base
	e eventspb.ReferralProgramStarted
}

func (t ReferralProgramStarted) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralProgramStarted{
		ReferralProgramStarted: &t.e,
	}

	return busEvent
}

func NewReferralProgramStartedEvent(ctx context.Context, p *types.ReferralProgram) *ReferralProgramStarted {
	return &ReferralProgramStarted{
		Base: newBase(ctx, ReferralProgramStartedEvent),
		e: eventspb.ReferralProgramStarted{
			Program: p.IntoProto(),
		},
	}
}

func ReferralProgramStartedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralProgramStarted {
	return &ReferralProgramStarted{
		Base: newBaseFromBusEvent(ctx, ReferralProgramStartedEvent, be),
		e:    *be.GetReferralProgramStarted(),
	}
}

type ReferralProgramUpdated struct {
	*Base
	e eventspb.ReferralProgramUpdated
}

func (t ReferralProgramUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralProgramUpdated{
		ReferralProgramUpdated: &t.e,
	}

	return busEvent
}

func NewReferralProgramUpdatedEvent(ctx context.Context, p *types.ReferralProgram) *ReferralProgramUpdated {
	return &ReferralProgramUpdated{
		Base: newBase(ctx, ReferralProgramUpdatedEvent),
		e: eventspb.ReferralProgramUpdated{
			Program: p.IntoProto(),
		},
	}
}

func ReferralProgramUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralProgramUpdated {
	return &ReferralProgramUpdated{
		Base: newBaseFromBusEvent(ctx, ReferralProgramUpdatedEvent, be),
		e:    *be.GetReferralProgramUpdated(),
	}
}

type ReferralProgramEnded struct {
	*Base
	e eventspb.ReferralProgramEnded
}

func (t ReferralProgramEnded) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_ReferralProgramEnded{
		ReferralProgramEnded: &t.e,
	}

	return busEvent
}

func NewReferralProgramEndedEvent(ctx context.Context, version uint64, id string) *ReferralProgramEnded {
	return &ReferralProgramEnded{
		Base: newBase(ctx, ReferralProgramEndedEvent),
		e: eventspb.ReferralProgramEnded{
			Version: version,
			Id:      id,
		},
	}
}

func ReferralProgramEndedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ReferralProgramEnded {
	return &ReferralProgramEnded{
		Base: newBaseFromBusEvent(ctx, ReferralProgramEndedEvent, be),
		e:    *be.GetReferralProgramEnded(),
	}
}
