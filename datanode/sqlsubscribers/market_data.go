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

type MarketDataEvent interface {
	events.Event
	MarketData() types.MarketData
}

type MarketDataStore interface {
	Add(*entities.MarketData) error
	Flush(context.Context) error
}

type MarketData struct {
	subscriber
	store MarketDataStore
}

func (md *MarketData) Flush(ctx context.Context) error {
	return md.store.Flush(ctx)
}

func (md *MarketData) Push(ctx context.Context, evt events.Event) error {
	return md.consume(evt.(MarketDataEvent))
}

func (md *MarketData) Types() []events.Type {
	return []events.Type{events.MarketDataEvent}
}

func NewMarketData(store MarketDataStore) *MarketData {
	return &MarketData{
		store: store,
	}
}

func (md *MarketData) consume(event MarketDataEvent) error {
	var record *entities.MarketData
	var err error
	mdProto := event.MarketData()

	if record, err = md.convertMarketDataProto(&mdProto, event.Sequence(), entities.TxHash(event.TxHash())); err != nil {
		errors.Wrap(err, "converting market data proto for persistence failed")
	}

	return errors.Wrap(md.store.Add(record), "inserting market data to SQL store failed")
}

func (md *MarketData) convertMarketDataProto(data *types.MarketData, seqNum uint64, txHash entities.TxHash) (*entities.MarketData, error) {
	record, err := entities.MarketDataFromProto(data, txHash)
	if err != nil {
		return nil, err
	}

	record.VegaTime = md.vegaTime
	record.SeqNum = seqNum
	record.SyntheticTime = md.vegaTime.Add(time.Duration(seqNum) * time.Microsecond)

	return record, nil
}
