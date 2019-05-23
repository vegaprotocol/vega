package accounts

import (
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrNoGeneralAccount = errors.New("no general accounts for trader")
	ErrOwnerNotInMarket = errors.New("trader has no accounts for given market")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/accounts AccountStore
type AccountStore interface {
	GetMarketAccountsForOwner(id, market string) ([]*types.Account, error)
	GetAccountsForOwner(owner string) ([]*types.Account, error)
	GetAccountsForOwnerByType(owner string, accType types.AccountType) ([]*types.Account, error)
}

// Service - the accounts service itself
type Service struct {
	Config
	log     *logging.Logger
	storage AccountStore
}

// New - create new accounts service
func New(log *logging.Logger, conf Config, storage AccountStore) *Service {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Service{
		Config:  conf,
		log:     log,
		storage: storage,
	}
}

func (s *Service) GetTraderAccounts(id string) ([]*types.Account, error) {
	// we can just return this outright, but we might want to use
	accs, err := s.storage.GetAccountsForOwner(id)
	if err != nil {
		return nil, err
	}
	return accs, nil
}

func (s *Service) GetTraderAccountsForMarket(trader, market string) ([]*types.Account, error) {
	accs, err := s.storage.GetMarketAccountsForOwner(trader, market)
	if err != nil {
		if err == storage.ErrOwnerNotFound {
			err = ErrOwnerNotInMarket
		}
		return nil, err
	}
	return accs, nil
}

// Get all accounts relevant for a trader on a market, so we can get the total balance available breakdown
func (s *Service) GetTraderMarketBalance(trader, market string) ([]*types.Account, error) {
	accs, err := s.GetTraderAccountsForMarket(trader, market)
	if err != nil {
		return nil, err
	}
	// get general account, too - need this balance for total funds available
	gen, err := s.storage.GetAccountsForOwnerByType(trader, types.AccountType_GENERAL)
	if err != nil {
		if err == storage.ErrAccountNotFound {
			err = ErrNoGeneralAccount
		}
		return nil, err
	}
	genMap := map[string]*types.Account{}
	for _, g := range gen {
		// this is tricky with bad test data, but tests should account for real life scenario's
		// we shouldn't write sub-optimal prod code to accomodate bad tests
		genMap[g.Asset] = g
	}
	// check base assets for accounts in market, only use general accounts with relevant asset
	relevant := make([]*types.Account, 0, len(gen))
	for _, a := range accs {
		if g, ok := genMap[a.Asset]; ok {
			relevant = append(relevant, g)
			// add general account once, remove from map after we're done here
			delete(genMap, a.Asset)
		}
	}
	accs = append(accs, relevant...)
	return accs, nil
}
