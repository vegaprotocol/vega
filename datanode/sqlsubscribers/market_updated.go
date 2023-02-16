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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type MarketUpdatedEvent interface {
	events.Event
	Market() vega.Market
}

type MarketUpdated struct {
	subscriber
	store MarketsStore
}

func NewMarketUpdated(store MarketsStore) *MarketUpdated {
	return &MarketUpdated{
		store: store,
	}
}

func (m *MarketUpdated) Types() []events.Type {
	return []events.Type{events.MarketUpdatedEvent}
}

func (m *MarketUpdated) Push(ctx context.Context, evt events.Event) error {
	return m.consume(ctx, evt.(MarketUpdatedEvent))
}

func (m *MarketUpdated) consume(ctx context.Context, event MarketUpdatedEvent) error {
	market := event.Market()
	record, err := entities.NewMarketFromProto(&market, entities.TxHash(event.TxHash()), m.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting market proto to database entity failed")
	}

	return errors.Wrap(m.store.Upsert(ctx, record), "updating market to SQL store failed")
}
