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

type PaidLiquidityFeesStats struct {
	*Base
	fs *eventspb.PaidLiquidityFeesStats
}

func NewPaidLiquidityFeesStatsEvent(ctx context.Context, fs *eventspb.PaidLiquidityFeesStats) *PaidLiquidityFeesStats {
	stats := &PaidLiquidityFeesStats{
		Base: newBase(ctx, PaidLiquidityFeesStatsEvent),
		fs:   fs,
	}
	return stats
}

func (f *PaidLiquidityFeesStats) LiquidityFeesStats() *eventspb.PaidLiquidityFeesStats {
	return f.fs
}

func (f PaidLiquidityFeesStats) Proto() eventspb.PaidLiquidityFeesStats {
	return *f.fs
}

func (f PaidLiquidityFeesStats) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(f.Base)
	busEvent.Event = &eventspb.BusEvent_PaidLiquidityFeesStats{
		PaidLiquidityFeesStats: f.fs,
	}

	return busEvent
}

func PaidLiquidityFeesStatsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *PaidLiquidityFeesStats {
	stats := &PaidLiquidityFeesStats{
		Base: newBaseFromBusEvent(ctx, PaidLiquidityFeesStatsEvent, be),
		fs:   be.GetPaidLiquidityFeesStats(),
	}
	return stats
}
