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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	FeeStatsEvent interface {
		events.Event
		FeeStats() *eventspb.FeeStats
	}

	ReferralFeeStatsStore interface {
		AddFeeStats(ctx context.Context, feeStats *entities.ReferralFeeStats) error
	}

	ReferralFeeStats struct {
		subscriber
		store ReferralFeeStatsStore
	}
)

func NewReferralFeeStats(store ReferralFeeStatsStore) *ReferralFeeStats {
	return &ReferralFeeStats{
		store: store,
	}
}

func (r *ReferralFeeStats) Types() []events.Type {
	return []events.Type{
		events.FeeStatsEvent,
	}
}

func (r *ReferralFeeStats) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case FeeStatsEvent:
		return r.consumeFeeStatsEvent(ctx, e)
	default:
		return nil
	}
}

func (r *ReferralFeeStats) consumeFeeStatsEvent(ctx context.Context, e FeeStatsEvent) error {
	return r.store.AddFeeStats(ctx, entities.ReferralFeeStatsFromProto(e.FeeStats(), r.vegaTime))
}
