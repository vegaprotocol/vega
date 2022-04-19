package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Ledger struct {
	*SQLStore
	batcher ListBatcher
}

func NewLedger(sqlStore *SQLStore) *Ledger {
	a := &Ledger{
		SQLStore: sqlStore,
		batcher:  NewListBatcher("ledger", entities.LedgerEntryColumns),
	}
	return a
}

func (ls *Ledger) Flush(ctx context.Context) error {
	return ls.batcher.Flush(ctx, ls.pool)
}

func (ls *Ledger) Add(le *entities.LedgerEntry) error {
	ls.batcher.Add(le)
	return nil
}

func (ls *Ledger) GetByID(id int64) (entities.LedgerEntry, error) {
	le := entities.LedgerEntry{}
	ctx := context.Background()
	err := pgxscan.Get(ctx, ls.pool, &le,
		`SELECT id, account_from_id, account_to_id, quantity, vega_time, transfer_time, reference, type
		 FROM ledger WHERE id=$1`,
		id)
	return le, err
}

func (ls *Ledger) GetAll() ([]entities.LedgerEntry, error) {
	ctx := context.Background()
	ledgerEntries := []entities.LedgerEntry{}
	err := pgxscan.Select(ctx, ls.pool, &ledgerEntries, `
		SELECT id, account_from_id, account_to_id, quantity, vega_time, transfer_time, reference, type
		FROM ledger`)
	return ledgerEntries, err
}
