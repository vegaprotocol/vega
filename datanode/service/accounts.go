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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

type AccountStore interface {
	// Use get by raw ID to avoid using the AccountID type because mockgen does not support it
	GetByRawID(ctx context.Context, id string) (entities.Account, error)
	GetAll(ctx context.Context) ([]entities.Account, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Account, error)
	Obtain(ctx context.Context, a *entities.Account) error
	Query(ctx context.Context, filter entities.AccountFilter) ([]entities.Account, error)
	QueryBalances(ctx context.Context, filter entities.AccountFilter, pagination entities.CursorPagination) ([]entities.AccountBalance, entities.PageInfo, error)
	GetBalancesByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.AccountBalance, error)
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
	return a.aStore.GetByRawID(ctx, id.String())
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

func (a *Account) QueryBalances(ctx context.Context, filter entities.AccountFilter, pagination entities.CursorPagination) ([]entities.AccountBalance, entities.PageInfo, error) {
	return a.aStore.QueryBalances(ctx, filter, pagination)
}

func (a *Account) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Account, error) {
	return a.aStore.GetByTxHash(ctx, txHash)
}

func (a *Account) GetBalancesByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.AccountBalance, error) {
	return a.aStore.GetBalancesByTxHash(ctx, txHash)
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

func (a *Account) Unsubscribe(ctx context.Context, ref uint64) error {
	return a.bObserver.Unsubscribe(ctx, ref)
}

func (a *Account) QueryAggregatedBalances(ctx context.Context, filter entities.AccountFilter, dateRange entities.DateRange, pagination entities.CursorPagination) (*[]entities.AggregatedBalance, entities.PageInfo, error) {
	return a.bStore.Query(ctx, filter, dateRange, pagination)
}

func (a *Account) ObserveAccountBalances(ctx context.Context, retries int, marketID string,
	asset string, ty vega.AccountType, partyIDs map[string]string,
) (accountCh <-chan []entities.AccountBalance, ref uint64) {
	ch, ref := a.bObserver.Observe(ctx,
		retries,
		func(ab entities.AccountBalance) bool {
			_, partyOK := partyIDs[ab.PartyID.String()]

			return (len(marketID) == 0 || marketID == ab.MarketID.String()) &&
				(partyOK) &&
				(len(asset) == 0 || asset == ab.AssetID.String()) &&
				(ty == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED || ty == ab.Type)
		})
	return ch, ref
}
