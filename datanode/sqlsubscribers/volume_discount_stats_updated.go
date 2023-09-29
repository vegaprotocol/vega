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

package sqlsubscribers

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
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
