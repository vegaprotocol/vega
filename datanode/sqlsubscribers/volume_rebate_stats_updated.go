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

	"github.com/pkg/errors"
)

type (
	VolumeRebateStatsUpdatedEvent interface {
		events.Event
		VolumeRebateStatsUpdated() *eventspb.VolumeRebateStatsUpdated
	}
	VolumeRebateStatsUpdatedStore interface {
		Add(context.Context, *entities.VolumeRebateStats) error
	}
	VolumeRebateStatsUpdated struct {
		subscriber
		store VolumeRebateStatsUpdatedStore
	}
)

func NewVolumeRebateStatsUpdated(store VolumeRebateStatsUpdatedStore) *VolumeRebateStatsUpdated {
	return &VolumeRebateStatsUpdated{
		store: store,
	}
}

func (pas *VolumeRebateStatsUpdated) Types() []events.Type {
	return []events.Type{
		events.VolumeRebateStatsUpdatedEvent,
	}
}

func (pas *VolumeRebateStatsUpdated) Push(ctx context.Context, evt events.Event) error {
	switch evt.Type() {
	case events.VolumeRebateStatsUpdatedEvent:
		return pas.consumeVolumeRebateStatsUpdatedEvent(ctx, evt.(VolumeRebateStatsUpdatedEvent))
	default:
		return nil
	}
}

func (pas *VolumeRebateStatsUpdated) consumeVolumeRebateStatsUpdatedEvent(ctx context.Context, evt VolumeRebateStatsUpdatedEvent) error {
	stats, err := entities.NewVolumeRebateStatsFromProto(evt.VolumeRebateStatsUpdated(), pas.vegaTime)
	if err != nil {
		return errors.Wrap(err, "could not convert volume rebate stats")
	}

	return errors.Wrap(pas.store.Add(ctx, stats), "could not add volume rebate stats to the store")
}

func (pas *VolumeRebateStatsUpdated) Name() string {
	return "VolumeRebateStatsUpdated"
}
