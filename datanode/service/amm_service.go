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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
)

type AMMPools struct {
	*sqlstore.AMMPools
}

func (a *AMMPools) ListByMarket(ctx context.Context, marketID string, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListByMarket(ctx, entities.MarketID(marketID), pagination)
}

func (a *AMMPools) ListByParty(ctx context.Context, partyID string, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListByParty(ctx, entities.PartyID(partyID), pagination)
}

func (a *AMMPools) ListByPool(ctx context.Context, poolID string, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListByPool(ctx, entities.AMMPoolID(poolID), pagination)
}

func (a *AMMPools) ListBySubAccount(ctx context.Context, ammPartyID string, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListBySubAccount(ctx, entities.PartyID(ammPartyID), pagination)
}
