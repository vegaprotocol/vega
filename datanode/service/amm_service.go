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
	"code.vegaprotocol.io/vega/libs/ptr"
)

type AMMPools struct {
	*sqlstore.AMMPools
}

func (a *AMMPools) ListByMarket(ctx context.Context, marketID string, liveOnly bool, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListByMarket(ctx, entities.MarketID(marketID), liveOnly, pagination)
}

func (a *AMMPools) ListByParty(ctx context.Context, partyID string, liveOnly bool, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListByParty(ctx, entities.PartyID(partyID), liveOnly, pagination)
}

func (a *AMMPools) ListByPool(ctx context.Context, poolID string, liveOnly bool, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListByPool(ctx, entities.AMMPoolID(poolID), liveOnly, pagination)
}

func (a *AMMPools) ListBySubAccount(ctx context.Context, ammPartyID string, liveOnly bool, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	return a.AMMPools.ListBySubAccount(ctx, entities.PartyID(ammPartyID), liveOnly, pagination)
}

func (a *AMMPools) ListByPartyMarketStatus(ctx context.Context, partyID, marketID *string, status *entities.AMMStatus, liveOnly bool, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	var (
		party  *entities.PartyID
		market *entities.MarketID
	)
	if partyID != nil {
		party = ptr.From(entities.PartyID(*partyID))
	}
	if marketID != nil {
		market = ptr.From(entities.MarketID(*marketID))
	}
	return a.AMMPools.ListByPartyMarketStatus(ctx, party, market, status, liveOnly, pagination)
}
