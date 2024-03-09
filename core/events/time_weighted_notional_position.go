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

type TimeWeightedNotionalPositionUpdated struct {
	*Base
	e eventspb.TimeWeightedNotionalPositionUpdated
}

func (tw *TimeWeightedNotionalPositionUpdated) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(tw.Base)
	busEvent.Event = &eventspb.BusEvent_TimeWeightedNotionalPositionUpdated{
		TimeWeightedNotionalPositionUpdated: &tw.e,
	}

	return busEvent
}

func (tw *TimeWeightedNotionalPositionUpdated) TimeWeightedNotionalPositionUpdated() *eventspb.TimeWeightedNotionalPositionUpdated {
	return &tw.e
}

func NewTimeWeightedNotionalPositionUpdated(ctx context.Context, asset, party, notionalPosition string) *TimeWeightedNotionalPositionUpdated {
	e := eventspb.TimeWeightedNotionalPositionUpdated{
		Asset:                        asset,
		Party:                        party,
		TimeWeightedNotionalPosition: notionalPosition,
	}

	return &TimeWeightedNotionalPositionUpdated{
		Base: newBase(ctx, TimeWeightedNotionalPositionUpdatedEvent),
		e:    e,
	}
}

func TimeWeightedNotionalPositionUpdatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TimeWeightedNotionalPositionUpdated {
	return &TimeWeightedNotionalPositionUpdated{
		Base: newBaseFromBusEvent(ctx, TimeWeightedNotionalPositionUpdatedEvent, be),
		e:    *be.GetTimeWeightedNotionalPositionUpdated(),
	}
}
