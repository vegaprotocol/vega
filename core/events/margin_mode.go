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

type PartyMarginModeUpdated struct {
	*Base
	update *eventspb.PartyMarginModeUpdated
}

func (e PartyMarginModeUpdated) Proto() eventspb.PartyMarginModeUpdated {
	return *e.update
}

func (e *PartyMarginModeUpdated) PartyMarginModeUpdated() *eventspb.PartyMarginModeUpdated {
	return e.update
}

func (e PartyMarginModeUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(e.Base)
	busEvent.Event = &eventspb.BusEvent_PartyMarginModeUpdated{
		PartyMarginModeUpdated: e.update,
	}

	return busEvent
}

func NewPartyMarginModeUpdatedEvent(ctx context.Context, update *eventspb.PartyMarginModeUpdated) *PartyMarginModeUpdated {
	e := &PartyMarginModeUpdated{
		Base:   newBase(ctx, PartyMarginModeUpdatedEvent),
		update: update,
	}
	return e
}

func PartyMarginModeUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *PartyMarginModeUpdated {
	e := &PartyMarginModeUpdated{
		Base:   newBaseFromBusEvent(ctx, PartyMarginModeUpdatedEvent, be),
		update: be.GetPartyMarginModeUpdated(),
	}
	return e
}
