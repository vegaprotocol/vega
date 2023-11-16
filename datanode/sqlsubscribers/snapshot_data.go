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

type CoreSnapshotEvent interface {
	events.Event
	SnapshotTakenEvent() eventspb.CoreSnapshotData
}

type snapAdder interface {
	AddSnapshot(context.Context, entities.CoreSnapshotData) error
}

type SnapshotData struct {
	subscriber
	store snapAdder
}

func NewSnapshotData(store snapAdder) *SnapshotData {
	return &SnapshotData{
		store: store,
	}
}

func (s *SnapshotData) Types() []events.Type {
	return []events.Type{events.CoreSnapshotEvent}
}

func (s *SnapshotData) Push(ctx context.Context, evt events.Event) error {
	return s.consume(ctx, evt.(CoreSnapshotEvent))
}

func (s *SnapshotData) consume(ctx context.Context, event CoreSnapshotEvent) error {
	sProto := event.SnapshotTakenEvent()
	snap := entities.CoreSnapshotDataFromProto(&sProto, entities.TxHash(event.TxHash()), s.vegaTime)

	if err := s.store.AddSnapshot(ctx, snap); err != nil {
		return fmt.Errorf("error adding core snapshot data: %w", err)
	}

	return nil
}
