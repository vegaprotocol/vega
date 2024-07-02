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
	VolumeDiscountStatsUpdatedEvent interface {
		events.Event
		VolumeDiscountStatsUpdated() *eventspb.VolumeDiscountStatsUpdated
	}
	VolumeDiscountStatsUpdatedStore interface {
		Add(context.Context, *entities.VolumeDiscountStats) error
	}
	VolumeDiscountStatsUpdated struct {
		subscriber
		store VolumeDiscountStatsUpdatedStore
	}
)

func NewVolumeDiscountStatsUpdated(store VolumeDiscountStatsUpdatedStore) *VolumeDiscountStatsUpdated {
	return &VolumeDiscountStatsUpdated{
		store: store,
	}
}

func (pas *VolumeDiscountStatsUpdated) Types() []events.Type {
	return []events.Type{
		events.VolumeDiscountStatsUpdatedEvent,
	}
}

func (pas *VolumeDiscountStatsUpdated) Push(ctx context.Context, evt events.Event) error {
	switch evt.Type() {
	case events.VolumeDiscountStatsUpdatedEvent:
		return pas.consumeVolumeDiscountStatsUpdatedEvent(ctx, evt.(VolumeDiscountStatsUpdatedEvent))
	default:
		return nil
	}
}

func (pas *VolumeDiscountStatsUpdated) consumeVolumeDiscountStatsUpdatedEvent(ctx context.Context, evt VolumeDiscountStatsUpdatedEvent) error {
	stats, err := entities.NewVolumeDiscountStatsFromProto(evt.VolumeDiscountStatsUpdated(), pas.vegaTime)
	if err != nil {
		return errors.Wrap(err, "could not convert volume discount stats")
	}

	return errors.Wrap(pas.store.Add(ctx, stats), "could not add volume discount stats to the store")
}

func (pas *VolumeDiscountStatsUpdated) Name() string {
	return "VolumeDiscountStatsUpdated"
}
