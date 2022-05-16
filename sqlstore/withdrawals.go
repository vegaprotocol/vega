package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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

func (w *Withdrawals) GetByParty(ctx context.Context, partyID string, openOnly bool, pagination entities.OffsetPagination) []entities.Withdrawal {
	var withdrawals []entities.Withdrawal
	prequery := `SELECT
		distinct on (id) id, party_id, amount, asset, status, ref, expiry, tx_hash,
		created_timestamp, withdrawn_timestamp, ext, vega_time
		FROM withdrawals
		WHERE party_id = $1
		ORDER BY id, vega_time DESC`

	var query string
	var args []interface{}

	query, args = orderAndPaginateQuery(prequery, nil, pagination, entities.NewPartyID(partyID))

	defer metrics.StartSQLQuery("Withdrawals", "GetByParty")()
	if err := pgxscan.Select(ctx, w.Connection, &withdrawals, query, args...); err != nil {
		return nil
	}

	return withdrawals
}
