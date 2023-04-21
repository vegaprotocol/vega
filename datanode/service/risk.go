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

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
)

type MarginLevelsStore interface {
	Add(marginLevel entities.MarginLevels) error
	Flush(ctx context.Context) ([]entities.MarginLevels, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.MarginLevels, error)
	GetMarginLevelsByIDWithCursorPagination(ctx context.Context, partyID, marketID string, pagination entities.CursorPagination) ([]entities.MarginLevels, entities.PageInfo, error)
}

type AccountSource interface {
	GetByID(ctx context.Context, id entities.AccountID) (entities.Account, error)
}

type Risk struct {
	mlStore       MarginLevelsStore
	accountSource AccountSource
	observer      utils.Observer[entities.MarginLevels]
}

func NewRisk(mlStore MarginLevelsStore, accountSource AccountSource, log *logging.Logger) *Risk {
	return &Risk{
		mlStore:       mlStore,
		accountSource: accountSource,
		observer:      utils.NewObserver[entities.MarginLevels]("margin_levels", log, 0, 0),
	}
}

func (r *Risk) Add(marginLevel entities.MarginLevels) error {
	return r.mlStore.Add(marginLevel)
}

func (r *Risk) Flush(ctx context.Context) error {
	flushed, err := r.mlStore.Flush(ctx)
	if err != nil {
		return err
	}
	r.observer.Notify(flushed)
	return nil
}

func (r *Risk) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.MarginLevels, error) {
	return r.mlStore.GetByTxHash(ctx, txHash)
}

func (r *Risk) GetMarginLevelsByIDWithCursorPagination(ctx context.Context, partyID, marketID string, pagination entities.CursorPagination) ([]entities.MarginLevels, entities.PageInfo, error) {
	return r.mlStore.GetMarginLevelsByIDWithCursorPagination(ctx, partyID, marketID, pagination)
}

func (r *Risk) ObserveMarginLevels(
	ctx context.Context, retries int, partyID, marketID string,
) (accountCh <-chan []entities.MarginLevels, ref uint64) {
	ch, ref := r.observer.Observe(ctx, retries,
		func(ml entities.MarginLevels) bool {
			acc, err := r.accountSource.GetByID(ctx, ml.AccountID)
			if err != nil {
				return false
			}
			return (len(marketID) == 0 || marketID == acc.MarketID.String()) &&
				len(partyID) == 0 || partyID == acc.PartyID.String()
		})
	return ch, ref
}
