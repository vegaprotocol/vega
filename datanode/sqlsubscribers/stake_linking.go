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

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
