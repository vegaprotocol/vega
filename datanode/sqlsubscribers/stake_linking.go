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

type StakeLinkingEvent interface {
	events.Event
	StakeLinking() eventspb.StakeLinking
}

type StakeLinkingStore interface {
	Upsert(ctx context.Context, linking *entities.StakeLinking) error
}

type StakeLinking struct {
	subscriber
	store StakeLinkingStore
}

func NewStakeLinking(store StakeLinkingStore) *StakeLinking {
	return &StakeLinking{
		store: store,
	}
}

func (sl *StakeLinking) Types() []events.Type {
	return []events.Type{events.StakeLinkingEvent}
}

func (sl *StakeLinking) Push(ctx context.Context, evt events.Event) error {
	return sl.consume(ctx, evt.(StakeLinkingEvent))
}

func (sl StakeLinking) consume(ctx context.Context, event StakeLinkingEvent) error {
	stake := event.StakeLinking()
	entity, err := entities.StakeLinkingFromProto(&stake, entities.TxHash(event.TxHash()), sl.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting stake linking event to database entity failed")
	}

	return errors.Wrap(sl.store.Upsert(ctx, entity), "inserting stake linking event to SQL store failed")
}

func (sl *StakeLinking) Name() string {
	return "StakeLinking"
}
