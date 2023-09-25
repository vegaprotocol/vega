// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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
