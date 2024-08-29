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
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type AMMPools struct {
	*ConnectionSource
}

var (
	ammPoolsOrdering = TableOrdering{
		ColumnOrdering{Name: "created_at", Sorting: ASC},
		ColumnOrdering{Name: "party_id", Sorting: DESC},
		ColumnOrdering{Name: "amm_party_id", Sorting: DESC},
		ColumnOrdering{Name: "market_id", Sorting: DESC},
		ColumnOrdering{Name: "id", Sorting: DESC},
	}

	activeStates = []entities.AMMStatus{entities.AMMStatusActive, entities.AMMStatusReduceOnly}
)

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
created_at, last_updated, proposed_fee,
lower_virtual_liquidity, lower_theoretical_position,
upper_virtual_liquidity, upper_theoretical_position) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
on conflict (party_id, market_id, id, amm_party_id) do update set
	commitment=excluded.commitment,
	status=excluded.status,
	status_reason=excluded.status_reason,
	parameters_base=excluded.parameters_base,
	parameters_lower_bound=excluded.parameters_lower_bound,
	parameters_upper_bound=excluded.parameters_upper_bound,
	parameters_leverage_at_lower_bound=excluded.parameters_leverage_at_lower_bound,
	parameters_leverage_at_upper_bound=excluded.parameters_leverage_at_upper_bound,
	last_updated=excluded.last_updated,
	proposed_fee=excluded.proposed_fee,
	lower_virtual_liquidity=excluded.lower_virtual_liquidity,
	lower_theoretical_position=excluded.lower_theoretical_position,
	upper_virtual_liquidity=excluded.upper_virtual_liquidity,
	upper_theoretical_position=excluded.upper_theoretical_position;`,
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
		pool.ProposedFee,
		pool.LowerVirtualLiquidity,
		pool.LowerTheoreticalPosition,
		pool.UpperVirtualLiquidity,
		pool.UpperTheoreticalPosition,
	); err != nil {
		return fmt.Errorf("could not upsert AMM Pool: %w", err)
	}

	return nil
}

func listByFields(ctx context.Context, connection Connection, fields map[string]entities.AMMFilterType, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	var (
		pools       []entities.AMMPool
		pageInfo    entities.PageInfo
		whereClause string
	)
	where := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields))
	for field, val := range fields {
		var clause string
		clause, args = val.Where(&field, nextBindVar, args...)
		where = append(where, clause)
	}
	whereClause = strings.Join(where, " AND ")
	query := fmt.Sprintf(`SELECT * FROM amms WHERE %s`, whereClause)
	query, args, err := PaginateQuery[entities.AMMPoolCursor](query, args, ammPoolsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}
	if err := pgxscan.Select(ctx, connection, &pools, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not list AMM Pools: %w", err)
	}

	pools, pageInfo = entities.PageEntities(pools, pagination)
	return pools, pageInfo, nil
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

func (p *AMMPools) GetSubKeysForParties(ctx context.Context, partyIDs []string, marketIDs []string) ([]string, error) {
	if len(partyIDs) == 0 {
		return nil, nil
	}
	parties := strings.Builder{}
	args := make([]any, 0, len(partyIDs)+len(marketIDs))
	query := `SELECT amm_party_id FROM amms WHERE "`
	for i, party := range partyIDs {
		if i > 0 {
			parties.WriteString(",")
		}
		parties.WriteString(nextBindVar(&args, party))
	}
	query = fmt.Sprintf(`%s party_id IN (%s)`, query, parties.String())
	if len(marketIDs) > 0 {
		markets := strings.Builder{}
		for i, mID := range marketIDs {
			if i > 0 {
				markets.WriteString(",")
			}
			markets.WriteString(nextBindVar(&args, mID))
		}
		query = fmt.Sprintf("%s AND market_id IN(%s)", query, markets.String())
	}

	subKeys := []string{}
	if err := pgxscan.Select(ctx, p.ConnectionSource, &subKeys, query, args...); err != nil {
		return nil, err
	}
	return subKeys, nil
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

func (p *AMMPools) ListByPartyMarketStatus(ctx context.Context, party *entities.PartyID, market *entities.MarketID, status *entities.AMMStatus, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("AMMs", "ListByPartyMarketStatus")
	fields := make(map[string]entities.AMMFilterType, 3)
	if party != nil {
		fields["party_id"] = party
	}
	if market != nil {
		fields["market_id"] = market
	}
	if status != nil {
		fields["status"] = status
	}
	return listByFields(ctx, p.ConnectionSource, fields, pagination)
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

func (p *AMMPools) ListActive(ctx context.Context) ([]entities.AMMPool, error) {
	defer metrics.StartSQLQuery("AMMs", "ListAll")
	var (
		pools       []entities.AMMPool
		args        []interface{}
		whereClause string
	)

	states := strings.Builder{}
	for i, status := range activeStates {
		if i > 0 {
			states.WriteString(",")
		}
		states.WriteString(nextBindVar(&args, status))
	}
	whereClause += fmt.Sprintf("status IN (%s)", states.String())
	query := fmt.Sprintf(`SELECT * from amms WHERE %s`, whereClause)

	if err := pgxscan.Select(ctx, p.ConnectionSource, &pools, query, args...); err != nil {
		return nil, fmt.Errorf("could not list active AMMs: %w", err)
	}

	return pools, nil
}
