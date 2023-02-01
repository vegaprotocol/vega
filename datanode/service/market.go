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
	"sync"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

var (
	nilPagination     = entities.OffsetPagination{}
	refreshCacheEvery = 3000 * time.Second
)

type MarketStore interface {
	Upsert(ctx context.Context, market *entities.Market) error
	GetByID(ctx context.Context, marketID string) (entities.Market, error)
	GetAll(ctx context.Context, pagination entities.OffsetPagination) ([]entities.Market, error)
	GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination) ([]entities.Market, entities.PageInfo, error)
}

type Markets struct {
	store     MarketStore
	log       *logging.Logger
	cache     map[entities.MarketID]*entities.Market
	cacheLock sync.RWMutex
}

func NewMarkets(store MarketStore, log *logging.Logger) *Markets {
	return &Markets{
		store: store,
		log:   log,
		cache: make(map[entities.MarketID]*entities.Market),
	}
}

func (m *Markets) Initialise(ctx context.Context) error {
	// This hacky timer is needed to ensure that the cache is re-populated when restoring
	// from network history segments. It would be better to explicitly refresh the cache when
	// loading in a segment, but this quick fix will work for now.
	go func() {
		for range time.NewTicker(refreshCacheEvery).C {
			err := m.refreshCache(ctx)
			if err != nil {
				m.log.Warn("error refreshing market cache", logging.Error(err))
			}
		}
	}()

	return m.refreshCache(ctx)
}

func (m *Markets) refreshCache(ctx context.Context) error {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()

	all, err := m.store.GetAll(ctx, entities.OffsetPagination{})
	if err != nil {
		return err
	}
	for i := 0; i < len(all); i++ {
		m.cache[all[i].ID] = &all[i]
	}
	return nil
}

func (m *Markets) Upsert(ctx context.Context, market *entities.Market) error {
	if err := m.store.Upsert(ctx, market); err != nil {
		return err
	}
	m.cacheLock.Lock()
	m.cache[market.ID] = market
	m.cacheLock.Unlock()
	return nil
}

func (m *Markets) GetByID(ctx context.Context, marketID string) (entities.Market, error) {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	data, ok := m.cache[entities.MarketID(marketID)]
	if !ok {
		return entities.Market{}, fmt.Errorf("no such market: %v", marketID)
	}
	return *data, nil
}

func (m *Markets) GetAll(ctx context.Context, pagination entities.OffsetPagination) ([]entities.Market, error) {
	if pagination != nilPagination {
		return m.store.GetAll(ctx, pagination)
	}

	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()

	data := make([]entities.Market, 0, len(m.cache))
	for _, v := range m.cache {
		data = append(data, *v)
	}
	return data, nil
}

func (m *Markets) GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination) ([]entities.Market, entities.PageInfo, error) {
	return m.store.GetAllPaged(ctx, marketID, pagination)
}
