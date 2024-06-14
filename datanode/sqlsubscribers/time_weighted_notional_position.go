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
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	TimeWeightedNotionalPositionUpdatedEvent interface {
		events.Event
		TimeWeightedNotionalPositionUpdated() *eventspb.TimeWeightedNotionalPositionUpdated
	}

	TimeWeightedNotionalPositionStore interface {
		Upsert(ctx context.Context, twNotionalPos entities.TimeWeightedNotionalPosition) error
	}

	TimeWeightedNotionalPosition struct {
		subscriber
		store TimeWeightedNotionalPositionStore
	}
)

func NewTimeWeightedNotionalPosition(store TimeWeightedNotionalPositionStore) *TimeWeightedNotionalPosition {
	return &TimeWeightedNotionalPosition{
		store: store,
	}
}

func (tw *TimeWeightedNotionalPosition) Types() []events.Type {
	return []events.Type{
		events.TimeWeightedNotionalPositionUpdatedEvent,
	}
}

func (tw *TimeWeightedNotionalPosition) Push(ctx context.Context, e events.Event) error {
	switch t := e.(type) {
	case TimeWeightedNotionalPositionUpdatedEvent:
		pos, err := entities.TimeWeightedNotionalPositionFromProto(t.TimeWeightedNotionalPositionUpdated(), tw.vegaTime)
		if err != nil {
			return fmt.Errorf("error converting TimeWeightedNotionalPositionUpdatedEvent to TimeWeightedNotionalPosition: %w", err)
		}
		return tw.store.Upsert(ctx, *pos)
	default:
		return fmt.Errorf("unexpected event type: %T", e)
	}
}

func (tw *TimeWeightedNotionalPosition) Name() string {
	return "TimeWeightedNotionalPosition"
}
