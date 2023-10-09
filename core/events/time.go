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
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// Time event indicating a change in block time (ie time update).
type Time struct {
	*Base
	blockTime time.Time
}

// NewTime returns a new time Update event.
func NewTime(ctx context.Context, t time.Time) *Time {
	return &Time{
		Base:      newBase(ctx, TimeUpdate),
		blockTime: t,
	}
}

// Time returns the new blocktime.
func (t Time) Time() time.Time {
	return t.blockTime
}

func (t Time) Proto() eventspb.TimeUpdate {
	return eventspb.TimeUpdate{
		Timestamp: t.blockTime.UnixNano(),
	}
}

func (t Time) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TimeUpdate{
		TimeUpdate: &p,
	}

	return busEvent
}

func TimeEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Time {
	return &Time{
		Base:      newBaseFromBusEvent(ctx, TimeUpdate, be),
		blockTime: time.Unix(0, be.GetTimeUpdate().Timestamp),
	}
}
