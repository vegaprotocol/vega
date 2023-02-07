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

package service

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
	lru "github.com/hashicorp/golang-lru/v2"
)

type MarketDataStore interface {
	Add(data *entities.MarketData) error
	Flush(ctx context.Context) ([]*entities.MarketData, error)
	GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error)
	GetMarketsData(ctx context.Context) ([]entities.MarketData, error)
	GetBetweenDatesByID(ctx context.Context, marketID string, start, end time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error)
	GetFromDateByID(ctx context.Context, marketID string, start time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error)
	GetToDateByID(ctx context.Context, marketID string, end time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error)
}

type MarketData struct {
	store    MarketDataStore
	log      *logging.Logger
	observer utils.Observer[*entities.MarketData]
	cache    *lru.Cache[entities.MarketID, *entities.MarketData]
}

func NewMarketData(store MarketDataStore, log *logging.Logger) *MarketData {
	cache, err := lru.New[entities.MarketID, *entities.MarketData](10000)
	if err != nil {
		panic(err)
	}

	return &MarketData{
		log:      log,
		store:    store,
		observer: utils.NewObserver[*entities.MarketData]("market_data", log, 0, 0),
		cache:    cache,
	}
}

func (m *MarketData) Add(data *entities.MarketData) error {
	if err := m.store.Add(data); err != nil {
		return err
	}
	return nil
}

func (m *MarketData) Flush(ctx context.Context) error {
	flushed, err := m.store.Flush(ctx)
	if err != nil {
		return err
	}
	m.observer.Notify(flushed)

	for _, data := range flushed {
		m.cache.Add(data.Market, data)
	}

	return nil
}

func (m *MarketData) Initialise(ctx context.Context) error {
	return nil
}

func (m *MarketData) GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error) {
	data, ok := m.cache.Get(entities.MarketID(marketID))
	if !ok {
		data, err := m.store.GetMarketDataByID(ctx, marketID)
		if err != nil {
			return entities.MarketData{}, fmt.Errorf("no market data for market: %v", marketID)
		}
		m.cache.Add(entities.MarketID(data.Market), &data)
	}
	return *data, nil
}

func (m *MarketData) GetMarketsData(ctx context.Context) ([]entities.MarketData, error) {
	all, err := m.store.GetMarketsData(ctx)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(all); i++ {
		m.cache.Add(all[i].Market, &all[i])
	}
	return all, nil
}

func (m *MarketData) GetBetweenDatesByID(ctx context.Context, marketID string, start, end time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error) {
	return m.store.GetBetweenDatesByID(ctx, marketID, start, end, pagination)
}

func (m *MarketData) GetFromDateByID(ctx context.Context, marketID string, start time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error) {
	return m.store.GetFromDateByID(ctx, marketID, start, pagination)
}

func (m *MarketData) GetToDateByID(ctx context.Context, marketID string, end time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error) {
	return m.store.GetToDateByID(ctx, marketID, end, pagination)
}

func (m *MarketData) ObserveMarketData(
	ctx context.Context, retries int, marketID []string,
) (<-chan []*entities.MarketData, uint64) {
	markets := map[string]bool{}
	for _, id := range marketID {
		markets[id] = true
	}

	ch, ref := m.observer.Observe(ctx,
		retries,
		func(md *entities.MarketData) bool { return markets[md.Market.String()] })
	return ch, ref
}
