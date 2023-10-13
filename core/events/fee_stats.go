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

type FeesStats struct {
	*Base
	fs *eventspb.FeesStats
}

func NewFeesStatsEvent(ctx context.Context, fs *eventspb.FeesStats) *FeesStats {
	order := &FeesStats{
		Base: newBase(ctx, FeesStatsEvent),
		fs:   fs,
	}
	return order
}

func (f *FeesStats) FeesStats() *eventspb.FeesStats {
	return f.fs
}

func (f FeesStats) Proto() eventspb.FeesStats {
	return *f.fs
}

func (f FeesStats) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(f.Base)
	busEvent.Event = &eventspb.BusEvent_FeesStats{
		FeesStats: f.fs,
	}

	return busEvent
}

func FeesStatsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *FeesStats {
	order := &FeesStats{
		Base: newBaseFromBusEvent(ctx, FeesStatsEvent, be),
		fs:   be.GetFeesStats(),
	}
	return order
}
