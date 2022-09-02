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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type MarginLevelsEvent interface {
	events.Event
	MarginLevels() vega.MarginLevels
}

type MarginLevelsStore interface {
	Add(entities.MarginLevels) error
	Flush(context.Context) error
}

type MarginLevels struct {
	subscriber
	store             MarginLevelsStore
	accountSource     AccountSource
	log               *logging.Logger
	eventDeduplicator *eventDeduplicator[marginLevelEventKey, MarginLevelsEvent]
}

type marginLevelEventKey struct {
	PartyID  string
	MarketID string
	AssetID  string
}

func NewMarginLevels(store MarginLevelsStore, accountSource AccountSource, log *logging.Logger) *MarginLevels {
	getID := func(ctx context.Context, evt MarginLevelsEvent) marginLevelEventKey {
		ml := evt.MarginLevels()
		return marginLevelEventKey{ml.PartyId, ml.MarketId, ml.Asset}
	}

	compareEvents := func(e1 MarginLevelsEvent, e2 MarginLevelsEvent) bool {
		ml1 := e1.MarginLevels()
		ml2 := e2.MarginLevels()
		return proto.Equal(&ml1, &ml2)
	}

	return &MarginLevels{
		store:             store,
		accountSource:     accountSource,
		log:               log,
		eventDeduplicator: NewEventDeduplicator(getID, compareEvents),
	}
}

func (ml *MarginLevels) Types() []events.Type {
	return []events.Type{events.MarginLevelsEvent}
}

func (ml *MarginLevels) Flush(ctx context.Context) error {
	err := ml.flush(ctx)
	if err != nil {
		return errors.Wrap(err, "flushing margin levels")
	}

	return nil
}

func (ml *MarginLevels) Push(ctx context.Context, evt events.Event) error {
	return ml.consume(ctx, evt.(MarginLevelsEvent))
}

func (ml *MarginLevels) flush(ctx context.Context) error {
	updates := ml.eventDeduplicator.Flush()
	for _, update := range updates {
		proto := update.MarginLevels()
		entity, err := entities.MarginLevelsFromProto(ctx, &proto, ml.accountSource, entities.TxHash(update.TxHash()), ml.vegaTime)
		if err != nil {
			return errors.Wrap(err, "converting margin level to database entity failed")
		}
		err = ml.store.Add(entity)
		if err != nil {
			return errors.Wrap(err, "add margin level to store")
		}
	}

	err := ml.store.Flush(ctx)

	return errors.Wrap(err, "flushing margin levels")
}

func (ml *MarginLevels) consume(ctx context.Context, event MarginLevelsEvent) error {
	marginLevel := event.MarginLevels()
	marginLevel.Timestamp = 0
	ml.eventDeduplicator.AddEvent(ctx, event)

	return nil
}
