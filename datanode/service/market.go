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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

var nilPagination = entities.OffsetPagination{}

type MarketStore interface {
	Upsert(ctx context.Context, market *entities.Market) error
	GetByID(ctx context.Context, marketID string) (entities.Market, error)
	GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination, includeSettled bool) ([]entities.Market, entities.PageInfo, error)
}

type Markets struct {
	store MarketStore
	log   *logging.Logger
}

func NewMarkets(store MarketStore, log *logging.Logger) *Markets {
	return &Markets{
		store: store,
		log:   log,
	}
}

func (m *Markets) Initialise(ctx context.Context) error {
	return nil
}

func (m *Markets) Upsert(ctx context.Context, market *entities.Market) error {
	return m.store.Upsert(ctx, market)
}

func (m *Markets) GetByID(ctx context.Context, marketID string) (entities.Market, error) {
	return m.store.GetByID(ctx, marketID)
}

func (m *Markets) GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination, includeSettled bool) ([]entities.Market, entities.PageInfo, error) {
	return m.store.GetAllPaged(ctx, marketID, pagination, includeSettled)
}
