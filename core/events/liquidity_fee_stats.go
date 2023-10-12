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

type PaidLiquidityFeeStats struct {
	*Base
	fs *eventspb.PaidLiquidityFeeStats
}

func NewPaidLiquidityFeeStatsEvent(ctx context.Context, fs *eventspb.PaidLiquidityFeeStats) *PaidLiquidityFeeStats {
	stats := &PaidLiquidityFeeStats{
		Base: newBase(ctx, PaidLiquidityFeeStatsEvent),
		fs:   fs,
	}
	return stats
}

func (f *PaidLiquidityFeeStats) LiquidityFeeStats() *eventspb.PaidLiquidityFeeStats {
	return f.fs
}

func (f PaidLiquidityFeeStats) Proto() eventspb.PaidLiquidityFeeStats {
	return *f.fs
}

func (f PaidLiquidityFeeStats) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(f.Base)
	busEvent.Event = &eventspb.BusEvent_PaidLiquidityFeeStats{
		PaidLiquidityFeeStats: f.fs,
	}

	return busEvent
}

func PaidLiquidityFeeStatsEventFromStream(ctx context.Context, be *eventspb.BusEvent) *PaidLiquidityFeeStats {
	stats := &PaidLiquidityFeeStats{
		Base: newBaseFromBusEvent(ctx, PaidLiquidityFeeStatsEvent, be),
		fs:   be.GetPaidLiquidityFeeStats(),
	}
	return stats
}
