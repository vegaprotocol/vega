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
	PaidLiquidityFeesStatsEvent interface {
		events.Event
		GetPaidLiquidityFeesStats() *eventspb.PaidLiquidityFeesStats
	}

	PaidLiquidityFeesStatsStore interface {
		Add(ctx context.Context, stats *entities.PaidLiquidityFeesStats) error
	}

	PaidLiquidityFeesStats struct {
		subscriber
		store PaidLiquidityFeesStatsStore
	}
)

func NewPaidLiquidityFeesStats(store PaidLiquidityFeesStatsStore) *PaidLiquidityFeesStats {
	return &PaidLiquidityFeesStats{
		store: store,
	}
}

func (r *PaidLiquidityFeesStats) Types() []events.Type {
	return []events.Type{
		events.PaidLiquidityFeesStatsEvent,
	}
}

func (r *PaidLiquidityFeesStats) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case PaidLiquidityFeesStatsEvent:
		return r.consumeFeeStatsEvent(ctx, e)
	default:
		return nil
	}
}

func (r *PaidLiquidityFeesStats) consumeFeeStatsEvent(ctx context.Context, e PaidLiquidityFeesStatsEvent) error {
	return r.store.Add(ctx, entities.PaidLiquidityFeesStatsFromProto(e.GetPaidLiquidityFeesStats()))
}
