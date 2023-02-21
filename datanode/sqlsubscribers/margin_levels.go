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
	"code.vegaprotocol.io/vega/protos/vega"
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
	store         MarginLevelsStore
	accountSource AccountSource
}

func NewMarginLevels(store MarginLevelsStore, accountSource AccountSource) *MarginLevels {
	return &MarginLevels{
		store:         store,
		accountSource: accountSource,
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
	err := ml.store.Flush(ctx)
	return errors.Wrap(err, "flushing margin levels")
}

func (ml *MarginLevels) consume(ctx context.Context, event MarginLevelsEvent) error {
	marginLevel := event.MarginLevels()
	marginLevel.Timestamp = 0

	proto := event.MarginLevels()
	entity, err := entities.MarginLevelsFromProto(ctx, &proto, ml.accountSource, entities.TxHash(event.TxHash()), ml.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting margin level to database entity failed")
	}

	err = ml.store.Add(entity)
	return errors.Wrap(err, "add margin level to store")
}
