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
	case entities.CursorPagination:
		return w.getByPartyCursor(ctx, partyID, p, dateRange)
	default:
		panic("unsupported pagination")
	}
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
