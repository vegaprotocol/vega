package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
	"code.vegaprotocol.io/protos/vega"
)

type ledgerStore interface {
	Flush(ctx context.Context) ([]entities.LedgerEntry, error)
	Add(le entities.LedgerEntry) error
}

type Ledger struct {
	store             ledgerStore
	log               *logging.Logger
	transferResponses []*vega.TransferResponse
	observer          utils.Observer[*vega.TransferResponse]
}

func NewLedger(store ledgerStore, log *logging.Logger) *Ledger {
	return &Ledger{
		store:    store,
		log:      log,
		observer: utils.NewObserver[*vega.TransferResponse]("ledger", log, 0, 0),
	}
}

func (l *Ledger) Flush(ctx context.Context) error {
	_, err := l.store.Flush(ctx)
	if err != nil {
		return err
	}
	l.observer.Notify(l.transferResponses)
	l.transferResponses = []*vega.TransferResponse{}
	return nil
}

func (l *Ledger) AddLedgerEntry(le entities.LedgerEntry) error {
	return l.store.Add(le)
}

func (l *Ledger) AddTransferResponse(le *vega.TransferResponse) {
	l.transferResponses = append(l.transferResponses, le)
}

func (l *Ledger) Observe(ctx context.Context, retries int) (<-chan []*vega.TransferResponse, uint64) {
	ch, ref := l.observer.Observe(ctx,
		retries,
		func(tr *vega.TransferResponse) bool {
			return true
		})
	return ch, ref
}
