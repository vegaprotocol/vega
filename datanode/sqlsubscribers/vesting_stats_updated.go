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
	VestingStatsUpdatedEvent interface {
		events.Event
		VestingStatsUpdated() *eventspb.VestingStatsUpdated
	}
	VestingStatsUpdatedStore interface {
		Add(context.Context, *entities.VestingStatsUpdated) error
	}
	VestingStatsUpdated struct {
		subscriber
		store VestingStatsUpdatedStore
	}
)

func NewVestingStatsUpdated(store VestingStatsUpdatedStore) *VestingStatsUpdated {
	return &VestingStatsUpdated{
		store: store,
	}
}

func (pas *VestingStatsUpdated) Types() []events.Type {
	return []events.Type{
		events.VestingStatsUpdatedEvent,
	}
}

func (pas *VestingStatsUpdated) Push(ctx context.Context, evt events.Event) error {
	switch evt.Type() {
	case events.VestingStatsUpdatedEvent:
		return pas.consumeVestingStatsUpdatedEvent(ctx, evt.(VestingStatsUpdatedEvent))
	default:
		return nil
	}
}

func (pas *VestingStatsUpdated) consumeVestingStatsUpdatedEvent(ctx context.Context, evt VestingStatsUpdatedEvent) error {
	stats, err := entities.NewVestingStatsFromProto(evt.VestingStatsUpdated(), pas.vegaTime)
	if err != nil {
		return errors.Wrap(err, "could not convert vesting stats")
	}

	return errors.Wrap(pas.store.Add(ctx, stats), "could not add vesting stats to the store")
}
