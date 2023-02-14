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
	"time"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	types "code.vegaprotocol.io/vega/protos/vega"
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
