package accounts

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

var (
	// ErrMissingPartyID signals that the payload is expected to contain a party id
	ErrMissingPartyID = errors.New("missing party id")
	// usually the party specified an amount of 0
	ErrInvalidWithdrawAmount = errors.New("invalid withdraw amount (must be > 0)")
	// ErrMissingAsset signals that an asset was required but not specified
	ErrMissingAsset = errors.New("missing asset")
)

// AccountStore represents a store for the accounts
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/accounts AccountStore
type AccountStore interface {
	GetPartyAccounts(string, string, string, types.AccountType) ([]*types.Account, error)
	GetMarketAccounts(string, string) ([]*types.Account, error)
	GetFeeInfrastructureAccounts(string) ([]*types.Account, error)
	Subscribe(c chan []*types.Account) uint64
	Unsubscribe(id uint64) error
}

// Svc implements the Account service business logic.
type Svc struct {
	Config
	log           *logging.Logger
	storage       AccountStore
	subscriberCnt int32
}

// NewService creates an instance of the accounts service.
func NewService(log *logging.Logger, conf Config, storage AccountStore) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Svc{
		Config:  conf,
		log:     log,
		storage: storage,
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

func (s *Svc) PrepareWithdraw(ctx context.Context, w *commandspb.WithdrawSubmission) error {
	if len(w.PartyId) <= 0 {
		return ErrMissingPartyID
	}
	if len(w.Asset) <= 0 {
		return ErrMissingAsset
	}
	if w.Amount == 0 {
		return ErrInvalidWithdrawAmount
	}
	return nil
}

func (s *Svc) GetPartyAccounts(partyID, marketID, asset string, ty types.AccountType) ([]*types.Account, error) {
	if ty == types.AccountType_ACCOUNT_TYPE_GENERAL {
		// General accounts for party are not specific to one market, therefore marketID should not be set
		marketID = ""
	}
	accounts, err := s.storage.GetPartyAccounts(partyID, marketID, asset, ty)
	// Prevent internal "!" special character from leaking out via marketID
	// There is a ticket to improve and clean this up in the collateral-engine:
	// https://github.com/vegaprotocol/vega/issues/416
	for _, acc := range accounts {
		if acc.GetType() == types.AccountType_ACCOUNT_TYPE_GENERAL || acc.GetType() == types.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW {
			if acc.GetMarketId() == "!" {
				acc.MarketId = ""
			}
		}
	}
	return accounts, err
}

func (s *Svc) GetMarketAccounts(marketID, asset string) ([]*types.Account, error) {
	accounts, err := s.storage.GetMarketAccounts(marketID, asset)
	// Prevent internal "*" special character from leaking out via owner (similar to above).
	// There is a ticket to improve and clean this up in the collateral-engine:
	// https://github.com/vegaprotocol/vega/issues/416
	for _, acc := range accounts {
		if acc.Owner == "*" {
			acc.Owner = ""
		}
	}
	return accounts, err
}

func (s *Svc) GetFeeInfrastructureAccounts(asset string) ([]*types.Account, error) {
	accounts, err := s.storage.GetFeeInfrastructureAccounts(asset)
	// Prevent internal "*" special character from leaking out via owner (similar to above).
	// There is a ticket to improve and clean this up in the collateral-engine:
	// https://github.com/vegaprotocol/vega/issues/416
	for _, acc := range accounts {
		if acc.Owner == "*" {
			acc.Owner = ""
		}
		if acc.MarketId == "!" {
			acc.MarketId = ""
		}
	}
	return accounts, err
}

// ObserveAccounts is used by streaming subscribers to be notified when changes
// are made to accounts for:
//
//  a) All parties and markets (specify empty marketID and empty partyID)
//  b) A particular party (specify empty partyID)
//  c) A particular market (specify empty marketID)
//  d) A particular party and market (specify marketID and partyID pair)
//  e) Any of the above combinations but with an optional account type e.g. AccountType.GENERAL
//  f) Optionally filter results by asset code e.g. USD
//
// This function is typically used by the gRPC (or GraphQL) asynchronous streaming APIs.
func (s *Svc) ObserveAccounts(ctx context.Context, retries int, marketID string,
	partyID string, asset string, ty types.AccountType) (accountCh <-chan []*types.Account, ref uint64) {

	accounts := make(chan []*types.Account)
	internal := make(chan []*types.Account)
	ref = s.storage.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
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
					if (len(marketID) <= 0 || marketID == acc.MarketId) &&
						(len(partyID) <= 0 || partyID == acc.Owner) &&
						(len(asset) <= 0 || asset == acc.Asset) &&
						(ty == types.AccountType_ACCOUNT_TYPE_UNSPECIFIED || ty == acc.Type) {
						filtered = append(filtered, acc)
					}
				}
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case accounts <- filtered:
						retryCount = retries
						s.log.Debug(
							"Accounts for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount > 0 {
							s.log.Debug(
								"Accounts for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip))
						}
						time.Sleep(time.Duration(10) * time.Millisecond)
					}
				}
				if !success && retryCount <= 0 {
					s.log.Warn(
						"Account subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries))
					cancel()
				}

			}
		}
	}()

	return accounts, ref

}

// GetAccountSubscribersCount returns the total number of active subscribers for ObserveAccounts.
func (s *Svc) GetAccountSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}
