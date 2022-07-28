// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
