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

package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type AccountEvent interface {
	events.Event
	Account() vega.Account
}

type AccountService interface {
	Obtain(ctx context.Context, a *entities.Account) error
	AddAccountBalance(b entities.AccountBalance) error
	Flush(ctx context.Context) error
}

type Account struct {
	subscriber
	accounts AccountService
}

func NewAccount(accounts AccountService) *Account {
	return &Account{
		accounts: accounts,
	}
}

func (as *Account) Types() []events.Type {
	return []events.Type{events.AccountEvent}
}

func (as *Account) Flush(ctx context.Context) error {
	err := as.accounts.Flush(ctx)
	return errors.Wrap(err, "flushing balances")
}

func (as *Account) Push(ctx context.Context, evt events.Event) error {
	return as.consume(ctx, evt.(AccountEvent))
}

func (as *Account) consume(ctx context.Context, evt AccountEvent) error {
	protoAcc := evt.Account()
	acc, err := as.obtainAccountWithProto(ctx, &protoAcc, evt.TxHash(), as.vegaTime)
	if err != nil {
		return errors.Wrap(err, "obtaining account")
	}

	balance, err := decimal.NewFromString(protoAcc.Balance)
	if err != nil {
		return errors.Wrap(err, "parsing account balance")
	}

	ab := entities.AccountBalance{
		Balance:  balance,
		Account:  &acc,
		TxHash:   entities.TxHash(evt.TxHash()),
		VegaTime: as.vegaTime,
	}

	err = as.accounts.AddAccountBalance(ab)
	if err != nil {
		return errors.Wrap(err, "adding balance to store")
	}
	return nil
}

func (as *Account) obtainAccountWithProto(ctx context.Context, va *vega.Account, txHash string, vegaTime time.Time) (entities.Account, error) {
	a, err := entities.AccountFromProto(va, entities.TxHash(txHash))
	if err != nil {
		return entities.Account{}, errors.Wrap(err, "obtaining account for balance")
	}

	a.VegaTime = vegaTime
	err = as.accounts.Obtain(ctx, &a)
	if err != nil {
		return entities.Account{}, errors.Wrap(err, "obtaining account")
	}
	return a, nil
}
