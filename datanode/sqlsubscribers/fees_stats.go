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
	FeesStatsEvent interface {
		events.Event
		FeesStats() *eventspb.FeesStats
	}

	FeesStatsStore interface {
		AddFeesStats(ctx context.Context, FeesStats *entities.FeesStats) error
	}

	FeesStats struct {
		subscriber
		store FeesStatsStore
	}
)

func NewFeesStats(store FeesStatsStore) *FeesStats {
	return &FeesStats{
		store: store,
	}
}

func (r *FeesStats) Types() []events.Type {
	return []events.Type{
		events.FeesStatsEvent,
	}
}

func (r *FeesStats) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case FeesStatsEvent:
		return r.consumeFeesStatsEvent(ctx, e)
	default:
		return nil
	}
}

func (r *FeesStats) consumeFeesStatsEvent(ctx context.Context, e FeesStatsEvent) error {
	return r.store.AddFeesStats(ctx, entities.FeesStatsFromProto(e.FeesStats(), r.vegaTime))
}

func (r *FeesStats) Name() string {
	return "FeesStats"
}
