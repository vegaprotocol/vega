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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type MarketCreatedEvent interface {
	events.Event
	Market() vega.Market
}

type MarketsStore interface {
	Upsert(context.Context, *entities.Market) error
}

type MarketCreated struct {
	subscriber
	store MarketsStore
}

func NewMarketCreated(store MarketsStore) *MarketCreated {
	return &MarketCreated{
		store: store,
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
	record, err := entities.NewMarketFromProto(&market, entities.TxHash(event.TxHash()), m.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting market proto to database entity failed")
	}

	return errors.Wrap(m.store.Upsert(ctx, record), "inserting market to SQL store failed:%w")
}

func (m *MarketCreated) Name() string {
	return "MarketCreated"
}
