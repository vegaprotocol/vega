package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Balances struct {
	*SQLStore
	batcher MapBatcher[entities.BalanceKey, entities.Balance]
}

func NewBalances(sqlStore *SQLStore) *Balances {
	b := &Balances{
		SQLStore: sqlStore,
		batcher: NewMapBatcher[entities.BalanceKey, entities.Balance](
			"balances",
			entities.BalanceColumns),
	}
	return b
}

func (bs *Balances) Flush(ctx context.Context) error {
	return bs.batcher.Flush(ctx, bs.pool)
}

// Add inserts a row to the balance table. If there's already a balance for this
// (account, block time) update it to match with the one supplied.
func (bs *Balances) Add(b entities.Balance) error {
	bs.batcher.Add(b)
	return nil
}

// Query queries and sums the balances of a given subset of accounts, specified via the 'filter' argument.
// It returns a timeseries (implemented as a list of AggregateBalance structs), with a row for every time
// the summed balance of the set of specified accounts changes.
//
// Optionally you can supply a list of fields to market by, which will break down the results by those fields.
//
// For example, if you have balances table that looks like
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
// 3     100        y
//
func (bs *Balances) Query(filter entities.AccountFilter, groupBy []entities.AccountField) (*[]entities.AggregatedBalance, error) {
	assetsQuery, args, err := filterAccountsQuery(filter)
	if err != nil {
		return nil, err
	}

	query := `
        WITH our_accounts AS (%s),
             timestamps AS (SELECT DISTINCT balances.vega_time
                            FROM balances JOIN our_accounts ON balances.account_id=our_accounts.id),
             keys AS (SELECT id AS account_id, timestamps.vega_time
                      FROM our_accounts CROSS JOIN timestamps),
             balances_with_nulls AS (SELECT keys.vega_time, keys.account_id, balance
                                     FROM keys LEFT JOIN balances
                                                      ON keys.account_id = balances.account_id
                                                     AND keys.vega_time=balances.vega_time),
             forward_filled_balances AS (SELECT vega_time, account_id, last(balance)
                                         OVER (partition by account_id order by vega_time) AS balance
                                         FROM balances_with_nulls)
        SELECT forward_filled_balances.vega_time %s, sum(balance) AS balance
        FROM forward_filled_balances
        JOIN our_accounts ON account_id=our_accounts.id
        WHERE balance IS NOT NULL
        GROUP BY forward_filled_balances.vega_time %s
        ORDER BY forward_filled_balances.vega_time %s;`

	groups := ""
	for _, col := range groupBy {
		groups = fmt.Sprintf("%s, %s", groups, col.String())
	}

	query = fmt.Sprintf(query, assetsQuery, groups, groups, groups)
	rows, err := bs.pool.Query(context.Background(), query, args...)
	defer rows.Close()
	if err != nil {
		return nil, fmt.Errorf("querying balances: %w", err)
	}

	res := []entities.AggregatedBalance{}

	if err = pgxscan.ScanAll(&res, rows); err != nil {
		return nil, fmt.Errorf("scanning balances: %w", err)
	}

	return &res, nil
}
