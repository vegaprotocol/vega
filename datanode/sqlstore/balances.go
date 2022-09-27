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

package sqlstore

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
)

var aggregateBalancesOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

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

// Query queries and sums the balances of a given subset of accounts, specified via the 'filter' argument.
// It returns a timeseries (implemented as a list of AggregateBalance structs), with a row for every time
// the summed balance of the set of specified accounts changes.
//
// Optionally you can supply a list of fields to market by, which will break down the results by those fields.
//
// # For example, if you have balances table that looks like
//
// Time  Account   Balance
// 1     a         1
// 2     b         10
// 3     c         100
//
// Querying with no filter and no grouping would give you
// Time  Balance    Party
// 1     1          nil
// 2     11         nil
// 3     111        nil
//
// Suppose accounts a and b belonged to party x, and account c belonged to party y.
// And you queried with groupBy=[AccountParty], you'd get
//
// Time  Balance    Party
// 1     1          x
// 2     11         x
// 3     100        y.
func (bs *Balances) Query(filter entities.AccountFilter, groupBy []entities.AccountField, dateRange entities.DateRange,
	pagination entities.CursorPagination,
) (*[]entities.AggregatedBalance, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	assetsQuery, args, err := filterAccountsQuery(filter)
	if err != nil {
		return nil, pageInfo, err
	}

	whereDate := ""
	if dateRange.Start != nil {
		whereDate = fmt.Sprintf("WHERE vega_time >= %s", nextBindVar(&args, *dateRange.Start))
	}

	if dateRange.End != nil {
		if whereDate != "" {
			whereDate = fmt.Sprintf("%s AND", whereDate)
		} else {
			whereDate = "WHERE "
		}
		whereDate = fmt.Sprintf("%s vega_time < %s", whereDate, nextBindVar(&args, *dateRange.End))
	}

	// This query is the one that gives us our results
	query := `
        WITH our_accounts AS (%s),
             timestamps AS (SELECT DISTINCT all_balances.vega_time
                            FROM all_balances JOIN our_accounts ON all_balances.account_id=our_accounts.id),
             keys AS (SELECT id AS account_id, timestamps.vega_time
                      FROM our_accounts CROSS JOIN timestamps),
             balances_with_nulls AS (SELECT keys.vega_time, keys.account_id, balance
                                     FROM keys LEFT JOIN all_balances
                                                      ON keys.account_id = all_balances.account_id
                                                     AND keys.vega_time=all_balances.vega_time),
             forward_filled_balances AS (SELECT vega_time, account_id, last(balance)
                                         OVER (partition by account_id order by vega_time) AS balance
                                         FROM balances_with_nulls),
        balances as (
			SELECT forward_filled_balances.vega_time %s, sum(balance) AS balance
	        FROM forward_filled_balances
    	    JOIN our_accounts ON account_id=our_accounts.id
        	WHERE balance IS NOT NULL
	        GROUP BY forward_filled_balances.vega_time %s
    	    ORDER BY forward_filled_balances.vega_time %s
		)
		%s`

	groups := ""
	for _, col := range groupBy {
		groups = fmt.Sprintf("%s, %s", groups, col.String())
	}

	// This pageQuery is the part that gives us the results for the pagination. We will only pass this part of the query
	// to the PaginateQuery function because the WHERE clause in the query above will cause an incorrect SQL statement
	// to be generated
	pageQuery := fmt.Sprintf(`SELECT vega_time %s, balance
		FROM balances 
		%s`, groups, whereDate)

	pageQuery, args, err = PaginateQuery[entities.AggregatedBalanceCursor](pageQuery, args, aggregateBalancesOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	// Here we stitch together all parts of the balances sub-query and the pagination to get the results we want.
	query = fmt.Sprintf(query, assetsQuery, groups, groups, groups, pageQuery)

	defer metrics.StartSQLQuery("Balances", "Query")()
	rows, err := bs.Connection.Query(context.Background(), query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying balances: %w", err)
	}
	defer rows.Close()

	// Scany won't let us scan strings to a pointer field, so we use an anonymous struct
	// here to work around that.
	results := []struct {
		VegaTime  time.Time
		Balance   decimal.Decimal
		AccountID entities.AccountID
		PartyID   entities.PartyID
		AssetID   entities.AssetID
		MarketID  entities.MarketID
		Type      *vega.AccountType
	}{}

	if err = pgxscan.ScanAll(&results, rows); err != nil {
		return nil, pageInfo, fmt.Errorf("scanning balances: %w", err)
	}

	balances := []entities.AggregatedBalance{}
	for _, res := range results {
		bal := entities.AggregatedBalance{
			VegaTime: res.VegaTime,
			Balance:  res.Balance,
			Type:     res.Type,
		}
		if res.AccountID != "" {
			bal.AccountID = res.AccountID
		}

		if res.PartyID != "" {
			bal.PartyID = res.PartyID
		}

		if res.AssetID != "" {
			bal.AssetID = res.AssetID
		}

		if res.MarketID != "" {
			bal.MarketID = res.MarketID
		}

		if res.AssetID != "" {
			bal.AssetID = res.AssetID
		}

		balances = append(balances, bal)
	}

	balances, pageInfo = entities.PageEntities[*v2.AggregatedBalanceEdge](balances, pagination)

	return &balances, pageInfo, nil
}
