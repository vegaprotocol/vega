// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type MarginLevelsEvent interface {
	events.Event
	MarginLevels() vega.MarginLevels
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/margin_levels_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers MarginLevelsStore
type MarginLevelsStore interface {
	Add(entities.MarginLevels) error
	Flush(context.Context) error
}

type MarginLevels struct {
	subscriber
	store             MarginLevelsStore
	accountSource     AccountSource
	log               *logging.Logger
	eventDeduplicator *eventDeduplicator[int64, *vega.MarginLevels]
}

func NewMarginLevels(store MarginLevelsStore, accountSource AccountSource, log *logging.Logger) *MarginLevels {
	return &MarginLevels{
		store:         store,
		accountSource: accountSource,
		log:           log,
		eventDeduplicator: NewEventDeduplicator[int64, *vega.MarginLevels](func(ctx context.Context,
			ml *vega.MarginLevels, vegaTime time.Time) (int64, error) {
			a, err := entities.GetAccountFromMarginLevel(ctx, ml, accountSource, vegaTime)
			if err != nil {
				return 0, err
			}

			return a.ID, nil
		}),
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
		entity, err := entities.MarginLevelsFromProto(ctx, update, ml.accountSource, ml.vegaTime)
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
	ml.eventDeduplicator.AddEvent(ctx, &marginLevel, ml.vegaTime)

	return nil
}
