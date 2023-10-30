// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
