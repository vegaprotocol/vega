package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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
	log      *logging.Logger
}

func NewAccount(
	accounts AccountService,
	log *logging.Logger,
) *Account {
	return &Account{
		accounts: accounts,
		log:      log,
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
	acc, err := as.obtainAccountWithProto(ctx, &protoAcc, as.vegaTime)
	if err != nil {
		return errors.Wrap(err, "obtaining account")
	}

	balance, err := decimal.NewFromString(protoAcc.Balance)
	if err != nil {
		return errors.Wrap(err, "parsing account balance")
	}

	ab := entities.AccountBalance{
		Balance: balance,
		Account: &acc,
	}
	ab.VegaTime = as.vegaTime

	err = as.accounts.AddAccountBalance(ab)
	if err != nil {
		return errors.Wrap(err, "adding balance to store")
	}
	return nil
}

func (as *Account) obtainAccountWithProto(ctx context.Context, va *vega.Account, vegaTime time.Time) (entities.Account, error) {
	a, err := entities.AccountFromProto(va)
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
