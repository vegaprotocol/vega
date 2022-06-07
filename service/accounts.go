package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
	"code.vegaprotocol.io/protos/vega"
)

type AccountStore interface {
	GetByID(id int64) (entities.Account, error)
	GetAll() ([]entities.Account, error)
	Obtain(ctx context.Context, a *entities.Account) error
	Query(filter entities.AccountFilter) ([]entities.Account, error)
	QueryBalances(ctx context.Context, filter entities.AccountFilter, pagination entities.OffsetPagination) ([]entities.AccountBalance, error)
}

type BalanceStore interface {
	Flush(ctx context.Context) ([]entities.AccountBalance, error)
	Add(b entities.AccountBalance) error
	Query(filter entities.AccountFilter, groupBy []entities.AccountField) (*[]entities.AggregatedBalance, error)
}

type Account struct {
	aStore    AccountStore
	bStore    BalanceStore
	bObserver utils.Observer[entities.AccountBalance]
	log       *logging.Logger
}

func NewAccount(aStore AccountStore, bStore BalanceStore, log *logging.Logger) *Account {
	return &Account{
		aStore:    aStore,
		bStore:    bStore,
		bObserver: utils.NewObserver[entities.AccountBalance]("account_balance", log, 0, 0),
		log:       log,
	}
}

func (a *Account) GetByID(id int64) (entities.Account, error) {
	return a.aStore.GetByID(id)
}

func (a *Account) GetAll() ([]entities.Account, error) {
	return a.aStore.GetAll()
}

func (a *Account) Obtain(ctx context.Context, acc *entities.Account) error {
	return a.aStore.Obtain(ctx, acc)
}

func (a *Account) Query(filter entities.AccountFilter) ([]entities.Account, error) {
	return a.aStore.Query(filter)
}

func (a *Account) QueryBalances(ctx context.Context, filter entities.AccountFilter, pagination entities.OffsetPagination) ([]entities.AccountBalance, error) {
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

func (a *Account) QueryAggregatedBalances(filter entities.AccountFilter, groupBy []entities.AccountField) (*[]entities.AggregatedBalance, error) {
	return a.bStore.Query(filter, groupBy)
}

func (a *Account) ObserveAccountBalances(ctx context.Context, retries int, marketID string,
	partyID string, asset string, ty vega.AccountType) (accountCh <-chan []entities.AccountBalance, ref uint64) {
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
