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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type MarketCreatedEvent interface {
	events.Event
	Market() vega.Market
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/markets_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers MarketsStore
type MarketsStore interface {
	Upsert(context.Context, *entities.Market) error
}

type MarketCreated struct {
	subscriber
	store MarketsStore
	log   *logging.Logger
}

func NewMarketCreated(store MarketsStore, log *logging.Logger) *MarketCreated {
	return &MarketCreated{
		store: store,
		log:   log,
	}
}

func (m *MarketCreated) Types() []events.Type {
	return []events.Type{events.MarketCreatedEvent}
}

func (m *MarketCreated) Push(ctx context.Context, evt events.Event) error {
	return m.consume(ctx, evt.(MarketCreatedEvent))
}

func (m *MarketCreated) consume(ctx context.Context, event MarketCreatedEvent) error {
	market := event.Market()
	record, err := entities.NewMarketFromProto(&market, m.vegaTime)

	if err != nil {
		return errors.Wrap(err, "converting market proto to database entity failed")
	}

	return errors.Wrap(m.store.Upsert(ctx, record), "inserting market to SQL store failed:%w")
}
