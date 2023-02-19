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
	"code.vegaprotocol.io/vega/protos/vega"
)

type AccountStore interface {
	GetByID(ctx context.Context, id entities.AccountID) (entities.Account, error)
	GetAll(ctx context.Context) ([]entities.Account, error)
	Obtain(ctx context.Context, a *entities.Account) error
	Query(ctx context.Context, filter entities.AccountFilter) ([]entities.Account, error)
	// TODO: remove.
	QueryBalancesV1(ctx context.Context, filter entities.AccountFilter, pagination entities.OffsetPagination) ([]entities.AccountBalance, error)
	QueryBalances(ctx context.Context, filter entities.AccountFilter, pagination entities.CursorPagination) ([]entities.AccountBalance, entities.PageInfo, error)
}

type BalanceStore interface {
	Flush(ctx context.Context) ([]entities.AccountBalance, error)
	Add(b entities.AccountBalance) error
	Query(ctx context.Context, filter entities.AccountFilter, dateRange entities.DateRange, pagination entities.CursorPagination) (*[]entities.AggregatedBalance, entities.PageInfo, error)
}

type Account struct {
	aStore    AccountStore
	bStore    BalanceStore
	bObserver utils.Observer[entities.AccountBalance]
}

func NewAccount(aStore AccountStore, bStore BalanceStore, log *logging.Logger) *Account {
	return &Account{
		aStore:    aStore,
		bStore:    bStore,
		bObserver: utils.NewObserver[entities.AccountBalance]("account_balance", log, 0, 0),
	}
}

func (a *Account) GetByID(ctx context.Context, id entities.AccountID) (entities.Account, error) {
	return a.aStore.GetByID(ctx, id)
}

func (a *Account) GetAll(ctx context.Context) ([]entities.Account, error) {
	return a.aStore.GetAll(ctx)
}

func (a *Account) Obtain(ctx context.Context, acc *entities.Account) error {
	return a.aStore.Obtain(ctx, acc)
}

func (a *Account) Query(ctx context.Context, filter entities.AccountFilter) ([]entities.Account, error) {
	return a.aStore.Query(ctx, filter)
}

// TODO: remove.
func (a *Account) QueryBalancesV1(ctx context.Context, filter entities.AccountFilter, pagination entities.OffsetPagination) ([]entities.AccountBalance, error) {
	return a.aStore.QueryBalancesV1(ctx, filter, pagination)
}

func (a *Account) QueryBalances(ctx context.Context, filter entities.AccountFilter, pagination entities.CursorPagination) ([]entities.AccountBalance, entities.PageInfo, error) {
	return a.aStore.QueryBalances(ctx, filter, pagination)
}

func (a *Account) AddAccountBalance(b entities.AccountBalance) error {
	return a.bStore.Add(b)
}

func (a *Account) Flush(ctx context.Context) error {
	flushed, err := a.bStore.Flush(ctx)
	if err != nil {
		return err
	}
	a.bObserver.Notify(flushed)
	return nil
}

func (a *Account) QueryAggregatedBalances(ctx context.Context, filter entities.AccountFilter, dateRange entities.DateRange, pagination entities.CursorPagination) (*[]entities.AggregatedBalance, entities.PageInfo, error) {
	return a.bStore.Query(ctx, filter, dateRange, pagination)
}

func (a *Account) ObserveAccountBalances(ctx context.Context, retries int, marketID string,
	partyID string, asset string, ty vega.AccountType,
) (accountCh <-chan []entities.AccountBalance, ref uint64) {
	ch, ref := a.bObserver.Observe(ctx,
		retries,
		func(ab entities.AccountBalance) bool {
			return (len(marketID) == 0 || marketID == ab.MarketID.String()) &&
				(len(partyID) == 0 || partyID == ab.PartyID.String()) &&
				(len(asset) == 0 || asset == ab.AssetID.String()) &&
				(ty == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED || ty == ab.Type)
		})
	return ch, ref
}
