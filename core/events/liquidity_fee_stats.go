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

type LiquidityFeeStats struct {
	*Base
	fs *eventspb.LiquidityFeeStats
}

func NewLiquidityFeeStatsEvent(ctx context.Context, fs *eventspb.LiquidityFeeStats) *LiquidityFeeStats {
	stats := &LiquidityFeeStats{
		Base: newBase(ctx, LiquidityFeeStatsEvent),
		fs:   fs,
	}
	return stats
}

func (f *LiquidityFeeStats) LiquidityFeeStats() *eventspb.LiquidityFeeStats {
	return f.fs
}

func (f LiquidityFeeStats) Proto() eventspb.LiquidityFeeStats {
	return *f.fs
}

func (f LiquidityFeeStats) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(f.Base)
	busEvent.Event = &eventspb.BusEvent_LiquidityFeeStats{
		LiquidityFeeStats: f.fs,
	}

	return busEvent
}

func LiquidityFeeStatsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LiquidityFeeStats {
	stats := &LiquidityFeeStats{
		Base: newBaseFromBusEvent(ctx, LiquidityFeeStatsEvent, be),
		fs:   be.GetLiquidityFeeStats(),
	}
	return stats
}
