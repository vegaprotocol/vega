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
	ColumnOrdering{Name: "amm_party_id", Sorting: DESC},
	ColumnOrdering{Name: "market_id", Sorting: DESC},
	ColumnOrdering{Name: "id", Sorting: DESC},
}

func NewAMMPools(connectionSource *ConnectionSource) *AMMPools {
	return &AMMPools{
		ConnectionSource: connectionSource,
	}
}

func (p *AMMPools) Upsert(ctx context.Context, pool entities.AMMPool) error {
	defer metrics.StartSQLQuery("AMMs", "UpsertAMM")
	if _, err := p.ConnectionSource.Exec(ctx, `
insert into amms(party_id, market_id, id, amm_party_id,
commitment, status, status_reason, 	parameters_base,
parameters_lower_bound, parameters_upper_bound,
parameters_leverage_at_lower_bound, parameters_leverage_at_upper_bound,
created_at, last_updated) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
on conflict (party_id, market_id, id, amm_party_id) do update set
	commitment=excluded.commitment,
	status=excluded.status,
	status_reason=excluded.status_reason,
	parameters_base=excluded.parameters_base,
	parameters_lower_bound=excluded.parameters_lower_bound,
	parameters_upper_bound=excluded.parameters_upper_bound,
	parameters_leverage_at_lower_bound=excluded.parameters_leverage_at_lower_bound,
	parameters_leverage_at_upper_bound=excluded.parameters_leverage_at_upper_bound,
	last_updated=excluded.last_updated;`,
		pool.PartyID,
		pool.MarketID,
		pool.ID,
		pool.AmmPartyID,
		pool.Commitment,
		pool.Status,
		pool.StatusReason,
		pool.ParametersBase,
		pool.ParametersLowerBound,
		pool.ParametersUpperBound,
		pool.ParametersLeverageAtLowerBound,
		pool.ParametersLeverageAtUpperBound,
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
	query := fmt.Sprintf(`SELECT * FROM amms WHERE %s`, whereClause)
	query, args, err := PaginateQuery[entities.AMMPoolCursor](query, args, ammPoolsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, connection, &pools, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not list AMM Pools: %w", err)
	}

	pools, pageInfo = entities.PageEntities[*v2.AMMEdge](pools, pagination)
	return pools, pageInfo, nil
}

func (p *AMMPools) ListByMarket(ctx context.Context, marketID entities.MarketID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMs", "ListByMarket")
	return listBy(ctx, p.ConnectionSource, "market_id", &marketID, pagination)
}

func (p *AMMPools) ListByParty(ctx context.Context, partyID entities.PartyID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMs", "ListByParty")

	return listBy(ctx, p.ConnectionSource, "party_id", &partyID, pagination)
}

func (p *AMMPools) ListByPool(ctx context.Context, poolID entities.AMMPoolID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMs", "ListByPool")
	return listBy(ctx, p.ConnectionSource, "id", &poolID, pagination)
}

func (p *AMMPools) ListBySubAccount(ctx context.Context, ammPartyID entities.PartyID, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMs", "ListByAMMParty")
	return listBy(ctx, p.ConnectionSource, "amm_party_id", &ammPartyID, pagination)
}

func (p *AMMPools) ListByStatus(ctx context.Context, status entities.AMMStatus, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMs", "ListByStatus")
	return listBy(ctx, p.ConnectionSource, "status", &status, pagination)
}

func (p *AMMPools) ListAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMs", "ListAll")
	var (
		pools    []entities.AMMPool
		pageInfo entities.PageInfo
		args     []interface{}
	)
	query := `SELECT * FROM amms`
	query, args, err := PaginateQuery[entities.AMMPoolCursor](query, args, ammPoolsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, p.ConnectionSource, &pools, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not list AMMs: %w", err)
	}

	pools, pageInfo = entities.PageEntities[*v2.AMMEdge](pools, pagination)
	return pools, pageInfo, nil
}
