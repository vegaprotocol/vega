// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type Withdrawals struct {
	*ConnectionSource
}

func NewWithdrawals(connectionSource *ConnectionSource) *Withdrawals {
	return &Withdrawals{
		ConnectionSource: connectionSource,
	}
}

func (w *Withdrawals) Upsert(ctx context.Context, withdrawal *entities.Withdrawal) error {
	defer metrics.StartSQLQuery("Withdrawals", "Upsert")()
	query := `insert into withdrawals(
		id, party_id, amount, asset, status, ref, expiry, tx_hash,
		created_timestamp, withdrawn_timestamp, ext, vega_time
	)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		on conflict (id, vega_time) do update
		set
			party_id=EXCLUDED.party_id,
			amount=EXCLUDED.amount,
			asset=EXCLUDED.asset,
			status=EXCLUDED.status,
			ref=EXCLUDED.ref,
			expiry=EXCLUDED.expiry,
			tx_hash=EXCLUDED.tx_hash,
			created_timestamp=EXCLUDED.created_timestamp,
			withdrawn_timestamp=EXCLUDED.withdrawn_timestamp,
			ext=EXCLUDED.ext`

	if _, err := w.Connection.Exec(ctx, query,
		withdrawal.ID,
		withdrawal.PartyID,
		withdrawal.Amount,
		withdrawal.Asset,
		withdrawal.Status,
		withdrawal.Ref,
		withdrawal.Expiry,
		withdrawal.TxHash,
		withdrawal.CreatedTimestamp,
		withdrawal.WithdrawnTimestamp,
		withdrawal.Ext,
		withdrawal.VegaTime); err != nil {
		err = fmt.Errorf("could not insert deposit into database: %w", err)
		return err
	}

	return nil
}

func (w *Withdrawals) GetByID(ctx context.Context, withdrawalID string) (entities.Withdrawal, error) {
	defer metrics.StartSQLQuery("Withdrawals", "GetByID")()
	var withdrawal entities.Withdrawal

	query := `select distinct on (id) id, party_id, amount, asset, status, ref, expiry, tx_hash, created_timestamp, withdrawn_timestamp, ext, vega_time
		from withdrawals
		where id = $1
		order by id, vega_time desc`

	err := pgxscan.Get(ctx, w.Connection, &withdrawal, query, entities.NewWithdrawalID(withdrawalID))
	return withdrawal, err
}

func (w *Withdrawals) GetByParty(ctx context.Context, partyID string, openOnly bool, pagination entities.Pagination) (
	[]entities.Withdrawal, entities.PageInfo, error) {
	switch p := pagination.(type) {
	case entities.OffsetPagination:
		return w.getByPartyOffset(ctx, partyID, openOnly, p)
	case entities.CursorPagination:
		return w.getByPartyCursor(ctx, partyID, openOnly, p)
	default:
		return w.getByPartyOffset(ctx, partyID, openOnly, entities.OffsetPagination{})
	}
}

func (w *Withdrawals) getByPartyOffset(ctx context.Context, partyID string, openOnly bool,
	pagination entities.OffsetPagination) ([]entities.Withdrawal, entities.PageInfo, error) {
	var withdrawals []entities.Withdrawal
	var pageInfo entities.PageInfo
	var args []interface{}

	query := fmt.Sprintf("%s WHERE party_id = %s ORDER BY id, vega_time DESC",
		getWithdrawalsByPartyQuery(), nextBindVar(&args, entities.NewPartyID(partyID)))
	query, args = orderAndPaginateQuery(query, nil, pagination, args...)

	defer metrics.StartSQLQuery("Withdrawals", "GetByParty")()
	if err := pgxscan.Select(ctx, w.Connection, &withdrawals, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get withdrawals by party: %w", err)
	}

	return withdrawals, pageInfo, nil

}

func (w *Withdrawals) getByPartyCursor(ctx context.Context, partyID string, openOnly bool,
	pagination entities.CursorPagination) ([]entities.Withdrawal, entities.PageInfo, error) {
	var withdrawals []entities.Withdrawal
	var pageInfo entities.PageInfo

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	wc := &entities.WithdrawalCursor{}
	if err := wc.Parse(cursor); err != nil {
		return nil, pageInfo, fmt.Errorf("could not parse cursor information: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("party_id", sorting, "=", entities.NewPartyID(partyID)),
		NewCursorQueryParameter("vega_time", sorting, cmp, wc.VegaTime),
		NewCursorQueryParameter("id", sorting, cmp, entities.NewWithdrawalID(wc.ID)),
	}

	var args []interface{}
	query := getWithdrawalsByPartyQuery()
	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	defer metrics.StartSQLQuery("Withdrawals", "GetByParty")()
	if err := pgxscan.Select(ctx, w.Connection, &withdrawals, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get withdrawals by party: %w", err)
	}

	withdrawals, pageInfo = entities.PageEntities[*v2.WithdrawalEdge](withdrawals, pagination)

	return withdrawals, pageInfo, nil
}

func getWithdrawalsByPartyQuery() string {
	return `SELECT
		id, party_id, amount, asset, status, ref, expiry, tx_hash,
		created_timestamp, withdrawn_timestamp, ext, vega_time
		FROM withdrawals_current`
}
