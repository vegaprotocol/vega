package sqlstore

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Deposits struct {
	*SQLStore
}

const (
	sqlDepositsColumns = `id, status, party_id, asset, amount, tx_hash,
		credited_timestamp, created_timestamp, vega_time`
)

func NewDeposits(sqlStore *SQLStore) *Deposits {
	return &Deposits{
		SQLStore: sqlStore,
	}
}

func (d *Deposits) Upsert(deposit *entities.Deposit) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.conf.Timeout.Duration)
	defer cancel()

	query := fmt.Sprintf(`insert into deposits(%s)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
on conflict (id, party_id, vega_time) do update
set
	status=EXCLUDED.status,
	asset=EXCLUDED.asset,
	amount=EXCLUDED.amount,
	tx_hash=EXCLUDED.tx_hash,
	credited_timestamp=EXCLUDED.credited_timestamp,
	created_timestamp=EXCLUDED.created_timestamp`, sqlDepositsColumns)

	if _, err := d.pool.Exec(ctx, query, deposit.ID, deposit.Status, deposit.PartyID, deposit.Asset, deposit.Amount,
		deposit.TxHash, deposit.CreditedTimestamp, deposit.CreatedTimestamp, deposit.VegaTime); err != nil {
		err = fmt.Errorf("could not insert deposit into database: %w", err)
		return err
	}

	return nil
}

func (d *Deposits) GetByID(ctx context.Context, depositID string) (entities.Deposit, error) {
	id, err := hex.DecodeString(depositID)
	if err != nil {
		return entities.Deposit{}, err
	}

	var deposit entities.Deposit

	query := `select distinct on (id, party_id) id, status, party_id, asset, amount, tx_hash, credited_timestamp, created_timestamp, vega_time
		from deposits
		where id = $1
		order by id, party_id, vega_time desc`

	err = pgxscan.Get(ctx, d.pool, &deposit, query, id)
	return deposit, err
}

func (d *Deposits) GetByParty(ctx context.Context, party string, openOnly bool, pagination entities.Pagination) []entities.Deposit {
	id, err := hex.DecodeString(party)
	if err != nil {
		return nil
	}

	var deposits []entities.Deposit
	var args []interface{}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(fmt.Sprintf(`select distinct on (id, party_id) id, status, party_id, asset, amount, tx_hash, credited_timestamp, created_timestamp, vega_time
		from deposits
		where party_id = %s `, nextBindVar(&args, id)))

	if openOnly {
		queryBuilder.WriteString(fmt.Sprintf(`and status = %s`, nextBindVar(&args, entities.DepositStatusOpen)))
	}

	queryBuilder.WriteString(" order by id, party_id, vega_time desc")

	var query string
	query, args = orderAndPaginateQuery(queryBuilder.String(), nil, pagination, args...)

	if err = pgxscan.Select(ctx, d.pool, &deposits, query, args...); err != nil {
		return nil
	}

	return deposits
}
