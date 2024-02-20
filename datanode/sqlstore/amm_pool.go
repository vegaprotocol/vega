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

package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type AMMPools struct {
	*ConnectionSource
}

var ammPoolsOrdering = TableOrdering{
	ColumnOrdering{Name: "created_at", Sorting: ASC},
	ColumnOrdering{Name: "party_id", Sorting: DESC},
	ColumnOrdering{Name: "sub_account", Sorting: DESC},
	ColumnOrdering{Name: "market_id", Sorting: DESC},
	ColumnOrdering{Name: "pool_id", Sorting: DESC},
}

func NewAMMPools(connectionSource *ConnectionSource) *AMMPools {
	return &AMMPools{
		ConnectionSource: connectionSource,
	}
}

func (p *AMMPools) Upsert(ctx context.Context, pool entities.AMMPool) error {
	defer metrics.StartSQLQuery("AMMPools", "UpsertAMMPool")
	if _, err := p.Connection.Exec(ctx, `
insert into amm_pool(party_id, market_id, pool_id, sub_account,
commitment, status, status_reason, 	parameters_base,
parameters_lower_bound, parameters_upper_bound,
parameters_margin_ratio_at_lower_bound, parameters_margin_ratio_at_upper_bound,
created_at, last_updated) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
on conflict (party_id, market_id, pool_id, sub_account) do update set
	commitment=excluded.commitment,
	status=excluded.status,
	status_reason=excluded.status_reason,
	parameters_base=excluded.parameters_base,
	parameters_lower_bound=excluded.parameters_lower_bound,
	parameters_upper_bound=excluded.parameters_upper_bound,
	parameters_margin_ratio_at_lower_bound=excluded.parameters_margin_ratio_at_lower_bound,
	parameters_margin_ratio_at_upper_bound=excluded.parameters_margin_ratio_at_upper_bound,
	last_updated=excluded.last_updated;`,
		pool.PartyID,
		pool.MarketID,
		pool.PoolID,
		pool.SubAccount,
		pool.Commitment,
		pool.Status,
		pool.StatusReason,
		pool.ParametersBase,
		pool.ParametersLowerBound,
		pool.ParametersUpperBound,
		pool.ParametersMarginRatioAtLowerBound,
		pool.ParametersMarginRatioAtUpperBound,
		pool.CreatedAt,
		pool.LastUpdated,
	); err != nil {
		return fmt.Errorf("could not upsert AMM Pool: %w", err)
	}

	return nil
}

func listBy[T entities.AMMPoolsFilter](ctx context.Context, connection Connection, fieldName string, filter T, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	var (
		pools       []entities.AMMPool
		pageInfo    entities.PageInfo
		args        []interface{}
		whereClause string
	)
	whereClause, args = filter.Where(&fieldName, nextBindVar, args...)
	query := fmt.Sprintf(`SELECT * FROM amm_pool WHERE %s`, whereClause)
	query, args, err := PaginateQuery[entities.AMMPoolCursor](query, args, ammPoolsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, connection, &pools, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not list AMM Pools: %w", err)
	}

	pools, pageInfo = entities.PageEntities[*v2.AMMPoolEdge](pools, pagination)
	return pools, pageInfo, nil
}

func (p *AMMPools) ListByMarket(ctx context.Context, marketID entities.MarketID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMPools", "ListByMarket")
	return listBy(ctx, p.Connection, "market_id", &marketID, pagination)
}

func (p *AMMPools) ListByParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMPools", "ListByParty")

	return listBy(ctx, p.Connection, "party_id", &partyID, pagination)
}

func (p *AMMPools) ListByPool(ctx context.Context, poolID entities.AMMPoolID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMPools", "ListByPool")
	return listBy(ctx, p.Connection, "pool_id", &poolID, pagination)
}

func (p *AMMPools) ListBySubAccount(ctx context.Context, subAccount entities.AccountID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMPools", "ListBySubAccount")
	return listBy(ctx, p.Connection, "sub_account", &subAccount, pagination)
}

func (p *AMMPools) ListByStatus(ctx context.Context, status entities.AMMPoolStatus, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMPools", "ListByStatus")
	return listBy(ctx, p.Connection, "status", &status, pagination)
}

func (p *AMMPools) ListAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMPools", "ListAll")
	var (
		pools    []entities.AMMPool
		pageInfo entities.PageInfo
		args     []interface{}
	)
	query := `SELECT * FROM amm_pool`
	query, args, err := PaginateQuery[entities.AMMPoolCursor](query, args, ammPoolsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, p.Connection, &pools, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not list AMM Pools: %w", err)
	}

	pools, pageInfo = entities.PageEntities[*v2.AMMPoolEdge](pools, pagination)
	return pools, pageInfo, nil
}
