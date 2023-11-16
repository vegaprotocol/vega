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

package service

import (
	"context"
	"io"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

type ledgerStore interface {
	Flush(ctx context.Context) ([]entities.LedgerEntry, error)
	Add(le entities.LedgerEntry) error
	Query(ctx context.Context, filter *entities.LedgerEntryFilter, dateRange entities.DateRange, pagination entities.CursorPagination) (*[]entities.AggregatedLedgerEntry, entities.PageInfo, error)
	Export(ctx context.Context, partyID string, assetID *string, dateRange entities.DateRange, writer io.Writer) error
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.LedgerEntry, error)
}

type LedgerEntriesStore interface {
	Query(filter *entities.LedgerEntryFilter, dateRange entities.DateRange, pagination entities.CursorPagination) (*[]entities.AggregatedLedgerEntry, entities.PageInfo, error)
	Export(ctx context.Context, partyID, assetID string, dateRange entities.DateRange, pagination entities.CursorPagination) ([]byte, entities.PageInfo, error)
}

type Ledger struct {
	store             ledgerStore
	transferResponses []*vega.LedgerMovement
	observer          utils.Observer[*vega.LedgerMovement]
}

func NewLedger(store ledgerStore, log *logging.Logger) *Ledger {
	return &Ledger{
		store:    store,
		observer: utils.NewObserver[*vega.LedgerMovement]("ledger", log, 0, 0),
	}
}

func (l *Ledger) Flush(ctx context.Context) error {
	_, err := l.store.Flush(ctx)
	if err != nil {
		return err
	}
	l.observer.Notify(l.transferResponses)
	l.transferResponses = []*vega.LedgerMovement{}
	return nil
}

func (l *Ledger) AddLedgerEntry(le entities.LedgerEntry) error {
	return l.store.Add(le)
}

func (l *Ledger) AddTransferResponse(le *vega.LedgerMovement) {
	l.transferResponses = append(l.transferResponses, le)
}

func (l *Ledger) Observe(ctx context.Context, retries int) (<-chan []*vega.LedgerMovement, uint64) {
	ch, ref := l.observer.Observe(ctx,
		retries,
		func(tr *vega.LedgerMovement) bool {
			return true
		})
	return ch, ref
}

func (l *Ledger) GetSubscribersCount() int32 {
	return l.observer.GetSubscribersCount()
}

func (l *Ledger) Query(
	ctx context.Context,
	filter *entities.LedgerEntryFilter,
	dateRange entities.DateRange,
	pagination entities.CursorPagination,
) (*[]entities.AggregatedLedgerEntry, entities.PageInfo, error) {
	return l.store.Query(
		ctx,
		filter,
		dateRange,
		pagination)
}

func (l *Ledger) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.LedgerEntry, error) {
	return l.store.GetByTxHash(ctx, txHash)
}

func (l *Ledger) Export(
	ctx context.Context,
	partyID string,
	assetID *string,
	dateRange entities.DateRange,
	writer io.Writer,
) error {
	return l.store.Export(
		ctx,
		partyID,
		assetID,
		dateRange,
		writer)
}
