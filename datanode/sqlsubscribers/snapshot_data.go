// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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
