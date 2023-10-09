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

type FeeStats struct {
	*Base
	fs *eventspb.FeeStats
}

func NewFeeStatsEvent(ctx context.Context, fs *eventspb.FeeStats) *FeeStats {
	order := &FeeStats{
		Base: newBase(ctx, FeeStatsEvent),
		fs:   fs,
	}
	return order
}

func (f *FeeStats) FeeStats() *eventspb.FeeStats {
	return f.fs
}

func (f FeeStats) Proto() eventspb.FeeStats {
	return *f.fs
}

func (f FeeStats) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(f.Base)
	busEvent.Event = &eventspb.BusEvent_FeeStats{
		FeeStats: f.fs,
	}

	return busEvent
}

func FeeStatsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *FeeStats {
	order := &FeeStats{
		Base: newBaseFromBusEvent(ctx, FeeStatsEvent, be),
		fs:   be.GetFeeStats(),
	}
	return order
}
