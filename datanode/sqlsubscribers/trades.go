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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type TradeEvent interface {
	events.Event
	Trade() types.Trade
}

type TradesStore interface {
	Add(*entities.Trade) error
	Flush(ctx context.Context) error
}

type TradeSubscriber struct {
	subscriber
	store TradesStore
}

func NewTradesSubscriber(store TradesStore) *TradeSubscriber {
	return &TradeSubscriber{
		store: store,
	}
}

func (ts *TradeSubscriber) Types() []events.Type {
	return []events.Type{events.TradeEvent}
}

func (ts *TradeSubscriber) Flush(ctx context.Context) error {
	return ts.store.Flush(ctx)
}

func (ts *TradeSubscriber) Push(ctx context.Context, evt events.Event) error {
	return ts.consume(evt.(TradeEvent))
}

func (ts *TradeSubscriber) consume(te TradeEvent) error {
	trade := te.Trade()
	return errors.Wrap(ts.addTrade(&trade, entities.TxHash(te.TxHash()), ts.vegaTime, te.Sequence()), "failed to consume trade")
}

func (ts *TradeSubscriber) addTrade(t *types.Trade, txHash entities.TxHash, vegaTime time.Time, blockSeqNumber uint64) error {
	trade, err := entities.TradeFromProto(t, txHash, vegaTime, blockSeqNumber)
	if err != nil {
		return errors.Wrap(err, "converting event to trade")
	}

	return errors.Wrap(ts.store.Add(trade), "adding trade to store")
}

func (ts *TradeSubscriber) Name() string {
	return "TradeSubscriber"
}
