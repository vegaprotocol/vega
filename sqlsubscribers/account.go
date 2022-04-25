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

type AccountStore interface {
	Obtain(ctx context.Context, a *entities.Account) error
}

type BalanceStore interface {
	Add(b entities.Balance) error
	Flush(ctx context.Context) error
}

type Account struct {
	accounts AccountStore
	vegaTime time.Time
	balances BalanceStore
	log      *logging.Logger
}

func NewAccount(
	accounts AccountStore,
	balances BalanceStore,
	log *logging.Logger,
) *Account {
	return &Account{
		accounts: accounts,
		balances: balances,
		log:      log,
	}
}

func (as *Account) Types() []events.Type {
	return []events.Type{events.AccountEvent}
}

func (as *Account) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		as.vegaTime = e.Time()
		err := as.balances.Flush(ctx)
		return errors.Wrap(err, "flushing balances")
	case AccountEvent:
		return as.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
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

	b := entities.Balance{
		AccountID: acc.ID,
		Balance:   balance,
		VegaTime:  as.vegaTime,
	}

	err = as.balances.Add(b)
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
