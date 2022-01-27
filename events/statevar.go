package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// StateVar is an event for tracking consensus in floating point state variables.
type StateVar struct {
	*Base
	ID      string
	EventID string
	State   string
}

func NewStateVarEvent(ctx context.Context, ID, eventID, state string) *StateVar {
	return &StateVar{
		Base:    newBase(ctx, StateVarEvent),
		ID:      ID,
		EventID: eventID,
		State:   state,
	}
}

func (sv StateVar) Proto() eventspb.StateVar {
	return eventspb.StateVar{
		Id:      sv.ID,
		EventId: sv.EventID,
		State:   sv.State,
	}
}

func (sv StateVar) StreamMessage() *eventspb.BusEvent {
	p := sv.Proto()
	busEvent := newBusEventFromBase(sv.Base)
	busEvent.Event = &eventspb.BusEvent_StateVar{
		StateVar: &p,
	}

	return busEvent
}

func StateVarEventFromStream(ctx context.Context, be *eventspb.BusEvent) *StateVar {
	event := be.GetStateVar()
	if event == nil {
		panic("failed to get state var event from event bus")
	}

	return &StateVar{
		Base:    newBaseFromBusEvent(ctx, StateVarEvent, be),
		ID:      event.GetId(),
		EventID: event.GetEventId(),
		State:   event.GetState(),
	}
}
