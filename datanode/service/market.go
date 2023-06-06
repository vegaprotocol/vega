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
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
)

type MarketStore interface {
	Upsert(ctx context.Context, market *entities.Market) error
	GetByID(ctx context.Context, marketID string) (entities.Market, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Market, error)
	GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination, includeSettled bool) ([]entities.Market, entities.PageInfo, error)
	ListSuccessorMarkets(ctx context.Context, marketID string, fullHistory bool) ([]entities.Market, error)
}

type Markets struct {
	store     MarketStore
	cacheLock sync.RWMutex
	sf        map[entities.MarketID]num.Decimal
}

func NewMarkets(store MarketStore) *Markets {
	return &Markets{
		store: store,
		sf:    map[entities.MarketID]num.Decimal{},
	}
}

func (m *Markets) Initialise(ctx context.Context) error {
	return nil
}

func (m *Markets) Upsert(ctx context.Context, market *entities.Market) error {
	if err := m.store.Upsert(ctx, market); err != nil {
		return err
	}
	m.cacheLock.Lock()
	if market.State == entities.MarketStateSettled || market.State == entities.MarketStateRejected {
		// a settled or rejected market can be safely removed from this map.
		delete(m.sf, market.ID)
	} else {
		// just in case this gets updated, or the market is new.
		m.sf[market.ID] = num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(int64(market.PositionDecimalPlaces)))
	}
	m.cacheLock.Unlock()
	return nil
}

func (m *Markets) GetByID(ctx context.Context, marketID string) (entities.Market, error) {
	return m.store.GetByID(ctx, marketID)
}

func (m *Markets) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Market, error) {
	return m.store.GetByTxHash(ctx, txHash)
}

func (m *Markets) GetMarketScalingFactor(ctx context.Context, marketID string) (num.Decimal, bool) {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()
	if pf, ok := m.sf[entities.MarketID(marketID)]; ok {
		return pf, true
	}

	market, err := m.store.GetByID(ctx, marketID)
	if err != nil {
		return num.Decimal{}, false
	}

	pf := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(int64(market.PositionDecimalPlaces)))
	return pf, true
}

func (m *Markets) GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination, includeSettled bool) ([]entities.Market, entities.PageInfo, error) {
	return m.store.GetAllPaged(ctx, marketID, pagination, includeSettled)
}

func (m *Markets) ListSuccessorMarkets(ctx context.Context, marketID string, childrenOnly bool) ([]entities.Market, error) {
	return m.store.ListSuccessorMarkets(ctx, marketID, childrenOnly)
}
