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

package service

import (
	"context"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/utils"
	"code.vegaprotocol.io/data-node/logging"
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

func (r *Ledger) GetSubscribersCount() int32 {
	return r.observer.GetSubscribersCount()
}
