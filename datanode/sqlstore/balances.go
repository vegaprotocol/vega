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
)

type Balances struct {
	*ConnectionSource
	batcher MapBatcher[entities.AccountBalanceKey, entities.AccountBalance]
}

func NewBalances(connectionSource *ConnectionSource) *Balances {
	b := &Balances{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.AccountBalanceKey, entities.AccountBalance](
			"balances",
			entities.BalanceColumns),
	}
	return b
}

func (bs *Balances) Flush(ctx context.Context) ([]entities.AccountBalance, error) {
	defer metrics.StartSQLQuery("Balances", "Flush")()
	return bs.batcher.Flush(ctx, bs.Connection)
}

// Add inserts a row to the balance table. If there's already a balance for this
// (account, block time) update it to match with the one supplied.
func (bs *Balances) Add(b entities.AccountBalance) error {
	bs.batcher.Add(b)
	return nil
}

func (bs *Balances) Query(ctx context.Context, filter entities.AccountFilter, dateRange entities.DateRange,
	pagination entities.CursorPagination,
) (*[]entities.AggregatedBalance, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	accountsQ, args, err := filterAccountsQuery(filter, false)
	if err != nil {
		return nil, pageInfo, err
	}

	predicates := []string{}
	if dateRange.Start != nil {
		predicate := fmt.Sprintf("vega_time >= %s", nextBindVar(&args, *dateRange.Start))
		predicates = append(predicates, predicate)
	}

	if dateRange.End != nil {
		predicate := fmt.Sprintf("vega_time < %s", nextBindVar(&args, *dateRange.End))
		predicates = append(predicates, predicate)
	}

	whereClause := ""
	if len(predicates) > 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(predicates, " AND "))
	}

	query := fmt.Sprintf(`
    WITH a AS(%s)
    SELECT b.vega_time,
        a.asset_id,
        a.party_id,
        a.market_id,
        a.type,
        b.balance
    FROM balances b JOIN a ON b.account_id = a.id
	%s`, accountsQ, whereClause)

	ordering := TableOrdering{
		ColumnOrdering{Name: "vega_time", Sorting: ASC},
		ColumnOrdering{Name: "account_id", Sorting: ASC},
	}

	query, args, err = PaginateQuery[entities.AggregatedBalanceCursor](query, args, ordering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	defer metrics.StartSQLQuery("Balances", "Query")()
	rows, err := bs.Connection.Query(ctx, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying balances: %w", err)
	}
	defer rows.Close()

	groupBy := []entities.AccountField{
		entities.AccountFieldAssetID,
		entities.AccountFieldPartyID,
		entities.AccountFieldMarketID,
		entities.AccountFieldType,
	}

	balances, err := entities.AggregatedBalanceScan(groupBy, rows)
	if err != nil {
		return nil, pageInfo, err
	}

	balances, pageInfo = entities.PageEntities[*v2.AggregatedBalanceEdge](balances, pagination)

	return &balances, pageInfo, nil
}
