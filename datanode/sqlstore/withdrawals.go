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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type Withdrawals struct {
	*ConnectionSource
}

const withdrawalsFilterDateColumn = "vega_time"

func NewWithdrawals(connectionSource *ConnectionSource) *Withdrawals {
	return &Withdrawals{
		ConnectionSource: connectionSource,
	}
}

var withdrawalsOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "id", Sorting: ASC},
}

func (w *Withdrawals) Upsert(ctx context.Context, withdrawal *entities.Withdrawal) error {
	defer metrics.StartSQLQuery("Withdrawals", "Upsert")()
	query := `insert into withdrawals(
		id, party_id, amount, asset, status, ref, foreign_tx_hash,
		created_timestamp, withdrawn_timestamp, ext, tx_hash, vega_time
	)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		on conflict (id, party_id, vega_time) do update
		set
			amount=EXCLUDED.amount,
			asset=EXCLUDED.asset,
			status=EXCLUDED.status,
			ref=EXCLUDED.ref,
			foreign_tx_hash=EXCLUDED.foreign_tx_hash,
			created_timestamp=EXCLUDED.created_timestamp,
			withdrawn_timestamp=EXCLUDED.withdrawn_timestamp,
			ext=EXCLUDED.ext,
			tx_hash=EXCLUDED.tx_hash`

	if _, err := w.Connection.Exec(ctx, query,
		withdrawal.ID,
		withdrawal.PartyID,
		withdrawal.Amount,
		withdrawal.Asset,
		withdrawal.Status,
		withdrawal.Ref,
		withdrawal.ForeignTxHash,
		withdrawal.CreatedTimestamp,
		withdrawal.WithdrawnTimestamp,
		withdrawal.Ext,
		withdrawal.TxHash,
		withdrawal.VegaTime); err != nil {
		err = fmt.Errorf("could not insert withdrawal into database: %w", err)
		return err
	}

	return nil
}

func (w *Withdrawals) GetByID(ctx context.Context, withdrawalID string) (entities.Withdrawal, error) {
	defer metrics.StartSQLQuery("Withdrawals", "GetByID")()
	var withdrawal entities.Withdrawal
	query := `select distinct on (id) id, party_id, amount, asset, status, ref,
									  foreign_tx_hash, created_timestamp, withdrawn_timestamp,
									  ext, tx_hash, vega_time
		from withdrawals
		where id = $1
		order by id, vega_time desc`

	return withdrawal, w.wrapE(pgxscan.Get(ctx, w.Connection, &withdrawal, query, entities.WithdrawalID(withdrawalID)))
}

func (w *Withdrawals) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Withdrawal, error) {
	defer metrics.StartSQLQuery("Withdrawals", "GetByTxHash")()

	var withdrawals []entities.Withdrawal
	query := `SELECT id, party_id, amount, asset, status, ref,
				foreign_tx_hash, created_timestamp, withdrawn_timestamp,
				ext, tx_hash, vega_time
		FROM withdrawals WHERE tx_hash = $1`

	err := pgxscan.Select(ctx, w.Connection, &withdrawals, query, txHash)
	if err != nil {
		return nil, w.wrapE(err)
	}

	return withdrawals, nil
}

func (w *Withdrawals) GetByParty(ctx context.Context, partyID string, openOnly bool, pagination entities.Pagination, dateRange entities.DateRange) (
	[]entities.Withdrawal, entities.PageInfo, error,
) {
	switch p := pagination.(type) {
	case entities.OffsetPagination:
		return w.getByPartyOffset(ctx, partyID, p)
	case entities.CursorPagination:
		return w.getByPartyCursor(ctx, partyID, p, dateRange)
	default:
		return w.getByPartyOffset(ctx, partyID, entities.OffsetPagination{})
	}
}

func (w *Withdrawals) getByPartyOffset(ctx context.Context, partyID string, pagination entities.OffsetPagination) ([]entities.Withdrawal, entities.PageInfo, error) {
	var withdrawals []entities.Withdrawal
	var pageInfo entities.PageInfo

	query, args := getWithdrawalsByPartyQuery(partyID, entities.DateRange{})
	query = fmt.Sprintf("%s ORDER BY id, vega_time DESC", query)
	query, args = orderAndPaginateQuery(query, nil, pagination, args...)

	defer metrics.StartSQLQuery("Withdrawals", "GetByParty")()
	if err := pgxscan.Select(ctx, w.Connection, &withdrawals, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get withdrawals by party: %w", err)
	}

	return withdrawals, pageInfo, nil
}

func (w *Withdrawals) getByPartyCursor(ctx context.Context, partyID string, pagination entities.CursorPagination, dateRange entities.DateRange) ([]entities.Withdrawal, entities.PageInfo, error) {
	var (
		withdrawals []entities.Withdrawal
		pageInfo    entities.PageInfo
		err         error
	)

	query, args := getWithdrawalsByPartyQuery(partyID, dateRange)
	query, args, err = PaginateQuery[entities.WithdrawalCursor](query, args, withdrawalsOrdering, pagination)
	if err != nil {
		return withdrawals, pageInfo, err
	}

	defer metrics.StartSQLQuery("Withdrawals", "GetByParty")()
	if err = pgxscan.Select(ctx, w.Connection, &withdrawals, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get withdrawals by party: %w", err)
	}

	withdrawals, pageInfo = entities.PageEntities[*v2.WithdrawalEdge](withdrawals, pagination)

	return withdrawals, pageInfo, nil
}

func getWithdrawalsByPartyQuery(partyID string, dateRange entities.DateRange) (string, []interface{}) {
	var args []interface{}

	query := `SELECT
		id, party_id, amount, asset, status, ref, foreign_tx_hash,
		created_timestamp, withdrawn_timestamp, ext, tx_hash, vega_time
		FROM withdrawals_current`

	first := true
	if partyID != "" {
		query = fmt.Sprintf("%s WHERE party_id = %s", query, nextBindVar(&args, entities.PartyID(partyID)))
		first = false
	}

	return filterDateRange(query, withdrawalsFilterDateColumn, dateRange, first, args...)
}
