package accounts

import (
	"context"
	"sync/atomic"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrNoGeneralAccount = errors.New("general accounts not found for party")
	ErrOwnerNotInMarket = errors.New("party has no accounts for market")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/accounts AccountStore
type AccountStore interface {
	GetByParty(partyID string) ([]*types.Account, error)
	GetByPartyAndMarket(partyID string, marketID string) ([]*types.Account, error)
	GetByPartyAndType(partyID string, accType types.AccountType) ([]*types.Account, error)
	GetByPartyAndAsset(partyID string, asset string) ([]*types.Account, error)

	Subscribe(c chan []*types.Account) uint64
	Unsubscribe(id uint64) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/internal/accounts  Blockchain
type Blockchain interface {
	NotifyTraderAccount(ctx context.Context, notif *types.NotifyTraderAccount) (success bool, err error)
}

// Account service business logic.
type Svc struct {
	Config
	log           *logging.Logger
	storage       AccountStore
	chain         Blockchain
	subscriberCnt int32
}

// NewService creates an instance of the accounts service.
func NewService(log *logging.Logger, conf Config, storage AccountStore, chain Blockchain) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Svc{
		Config:  conf,
		log:     log,
		storage: storage,
		chain:   chain,
	}
}

// ReloadConf reloads configuration for this package from toml config files.
func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.Config = cfg
}

// NotifyTraderAccount performs a request to update a party account with new collateral.
// NOTE: this functionality should be removed in the future, or updated when we have test ether wallets.
func (s *Svc) NotifyTraderAccount(ctx context.Context, nta *types.NotifyTraderAccount) (bool, error) {
	return s.chain.NotifyTraderAccount(ctx, nta)
}

// GetByParty returns details of all accounts for a given party (if they have placed orders on VEGA).
func (s *Svc) GetByParty(partyID string) ([]*types.Account, error) {
	accounts, err := s.storage.GetByParty(partyID)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetByPartyAndMarket returns all accounts for a given market
// and party (if they have placed orders on that market on VEGA).
func (s *Svc) GetByPartyAndMarket(partyID string, marketID string) ([]*types.Account, error) {
	accounts, err := s.storage.GetByPartyAndMarket(partyID, marketID)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetByPartyAndType returns all accounts for a given type (on all markets)
// and party (if they have placed orders on that market on VEGA).
func (s *Svc) GetByPartyAndType(partyID string, accType types.AccountType) ([]*types.Account, error) {
	accounts, err := s.storage.GetByPartyAndType(partyID, accType)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetByPartyAndAsset returns all accounts for a given asset (on all markets)
// and party (if they have placed orders on that market on VEGA).
func (s *Svc) GetByPartyAndAsset(partyID string, asset string) ([]*types.Account, error) {
	accounts, err := s.storage.GetByPartyAndAsset(partyID, asset)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// ObserveAccounts is used by streaming subscribers to be notified when changes
// are made to accounts for:
//
//  a) All parties and markets (specify empty marketID and empty partyID)
//  b) A particular party (specify empty partyID)
//  c) A particular market (specify empty marketID)
//  d) A particular party and market (specify marketID and partyID pair)
//  e) Any of the above combinations but with an optional account type e.g. AccountType.GENERAL
//
// This function is typically used by the gRPC (or GraphQL) asynchronous streaming APIs.
func (s *Svc) ObserveAccounts(ctx context.Context, retries int, marketID string, partyID string, ty types.AccountType) (accountCh <-chan []*types.Account, ref uint64) {
	accounts := make(chan []*types.Account)
	internal := make(chan []*types.Account)
	ref = s.storage.Subscribe(internal)

	retryCount := retries
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
		ip := logging.IPAddressFromContext(ctx)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"Accounts subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				// this error only happens when the subscriber reference doesn't exist
				// so we can still safely close the channels
				if err := s.storage.Unsubscribe(ref); err != nil {
					s.log.Error(
						"Failure un-subscribing accounts subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(accounts)
				return
			case accs := <-internal:
				filtered := make([]*types.Account, 0, len(accs))
				for _, acc := range accs {
					acc := acc
					// todo: split this out into separate func?
					if (len(marketID) <= 0 || marketID == acc.MarketID) &&
						(len(partyID) <= 0 || partyID == acc.Owner) &&
						(ty == types.AccountType_NO_ACC || ty == acc.Type) {
						filtered = append(filtered, acc)
					}
				}
				select {
				case accounts <- filtered:
					retryCount = retries
					s.log.Debug(
						"Accounts for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				default:
					retryCount--
					if retryCount == 0 {
						s.log.Warn(
							"Account subscriber has hit the retry limit",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries))

						cancel()
					}
					s.log.Debug(
						"Accounts for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				}
			}
		}
	}()

	return accounts, ref

}

// todo: godoc and check where this should be used? eg. stats
func (s *Svc) GetAccountSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}