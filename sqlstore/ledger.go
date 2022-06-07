package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type Ledger struct {
	*ConnectionSource
	batcher ListBatcher[entities.LedgerEntry]
}

func NewLedger(connectionSource *ConnectionSource) *Ledger {
	a := &Ledger{
		ConnectionSource: connectionSource,
		batcher:          NewListBatcher[entities.LedgerEntry]("ledger", entities.LedgerEntryColumns),
	}
	return a
}

func (ls *Ledger) Flush(ctx context.Context) ([]entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "Flush")()
	return ls.batcher.Flush(ctx, ls.Connection)
}

func (ls *Ledger) Add(le entities.LedgerEntry) error {
	ls.batcher.Add(le)
	return nil
}

func (ls *Ledger) GetByID(id int64) (entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "GetByID")()
	le := entities.LedgerEntry{}
	ctx := context.Background()
	err := pgxscan.Get(ctx, ls.Connection, &le,
		`SELECT id, account_from_id, account_to_id, quantity, vega_time, transfer_time, reference, type
		 FROM ledger WHERE id=$1`,
		id)
	return le, err
}

func (ls *Ledger) GetAll() ([]entities.LedgerEntry, error) {
	defer metrics.StartSQLQuery("Ledger", "GetAll")()
	ctx := context.Background()
	ledgerEntries := []entities.LedgerEntry{}
	err := pgxscan.Select(ctx, ls.Connection, &ledgerEntries, `
		SELECT id, account_from_id, account_to_id, quantity, vega_time, transfer_time, reference, type
		FROM ledger`)
	return ledgerEntries, err
}
