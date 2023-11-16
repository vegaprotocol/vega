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
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
)

type MarketStore interface {
	Upsert(ctx context.Context, market *entities.Market) error
	GetByID(ctx context.Context, marketID string) (entities.Market, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Market, error)
	GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination, includeSettled bool) ([]entities.Market, entities.PageInfo, error)
	ListSuccessorMarkets(ctx context.Context, marketID string, fullHistory bool, pagination entities.CursorPagination) ([]entities.SuccessorMarket, entities.PageInfo, error)
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

func (m *Markets) ListSuccessorMarkets(ctx context.Context, marketID string, childrenOnly bool, pagination entities.CursorPagination) ([]entities.SuccessorMarket, entities.PageInfo, error) {
	return m.store.ListSuccessorMarkets(ctx, marketID, childrenOnly, pagination)
}
