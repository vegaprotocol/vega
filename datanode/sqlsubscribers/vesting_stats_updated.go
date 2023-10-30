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
