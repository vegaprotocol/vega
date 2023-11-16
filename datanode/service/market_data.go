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

package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
)

type MarketDataStore interface {
	Add(data *entities.MarketData) error
	Flush(ctx context.Context) ([]*entities.MarketData, error)
	GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error)
	GetMarketsData(ctx context.Context) ([]entities.MarketData, error)
	GetHistoricMarketData(ctx context.Context, marketID string, start, end *time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error)
}

type MarketData struct {
	store     MarketDataStore
	observer  utils.Observer[*entities.MarketData]
	cache     map[entities.MarketID]*entities.MarketData
	cacheLock sync.RWMutex
}

func NewMarketData(store MarketDataStore, log *logging.Logger) *MarketData {
	return &MarketData{
		store:    store,
		observer: utils.NewObserver[*entities.MarketData]("market_data", log, 0, 0),
		cache:    make(map[entities.MarketID]*entities.MarketData),
	}
}

func (m *MarketData) Add(data *entities.MarketData) error {
	if err := m.store.Add(data); err != nil {
		return err
	}
	m.cacheLock.Lock()
	m.cache[data.Market] = data
	m.cacheLock.Unlock()
	return nil
}

func (m *MarketData) Flush(ctx context.Context) error {
	flushed, err := m.store.Flush(ctx)
	if err != nil {
		return err
	}
	m.observer.Notify(flushed)
	return nil
}

func (m *MarketData) Initialise(ctx context.Context) error {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()

	all, err := m.store.GetMarketsData(ctx)
	if err != nil {
		return err
	}
	for i := 0; i < len(all); i++ {
		m.cache[all[i].Market] = &all[i]
	}
	return nil
}

func (m *MarketData) GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error) {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	data, ok := m.cache[entities.MarketID(marketID)]
	if !ok {
		return entities.MarketData{}, fmt.Errorf("no market data for market: %v", marketID)
	}
	return *data, nil
}

func (m *MarketData) GetMarketsData(ctx context.Context) ([]entities.MarketData, error) {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	data := make([]entities.MarketData, 0, len(m.cache))
	for _, v := range m.cache {
		data = append(data, *v)
	}
	return data, nil
}

func (m *MarketData) GetHistoricMarketData(ctx context.Context, marketID string, start, end *time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error) {
	return m.store.GetHistoricMarketData(ctx, marketID, start, end, pagination)
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
