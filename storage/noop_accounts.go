package storage

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type NoopAccount struct {
	Config

	log          *logging.Logger
	subscribers  map[uint64]chan []*types.Account
	subscriberID uint64
	mu           sync.Mutex
}

func NewNoopAccounts(log *logging.Logger, c Config) *NoopAccount {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())
	return &NoopAccount{
		log:         log,
		Config:      c,
		subscribers: map[uint64]chan []*types.Account{},
	}
}

func (a *NoopAccount) ReloadConf(cfg Config) {
	a.log.Info("reloading configuration")
	if a.log.GetLevel() != cfg.Level.Get() {
		a.log.Info("updating log level",
			logging.String("old", a.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		a.log.SetLevel(cfg.Level.Get())
	}

	a.Config = cfg
}

func (a *NoopAccount) Close() error {
	return nil
}

func (a *NoopAccount) GetPartyAccounts(partyID, marketID, asset string, ty types.AccountType) ([]*types.Account, error) {
	return []*types.Account{}, nil
}

func (a *NoopAccount) GetMarketAccounts(marketID, asset string) ([]*types.Account, error) {
	return []*types.Account{}, nil
}

func (a *NoopAccount) GetFeeInfrastructureAccounts(asset string) ([]*types.Account, error) {
	return []*types.Account{}, nil
}

func (a *NoopAccount) SaveBatch(accs []*types.Account) error {
	return nil
}

// Subscribe to account store updates, any changes will be pushed out on this channel.
func (a *NoopAccount) Subscribe(c chan []*types.Account) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.subscriberID++
	a.subscribers[a.subscriberID] = c

	a.log.Debug("NoopAccount subscriber added in account store",
		logging.Uint64("subscriber-id", a.subscriberID))

	return a.subscriberID
}

// Unsubscribe from account store updates.
func (a *NoopAccount) Unsubscribe(id uint64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.subscribers) == 0 {
		a.log.Debug("Un-subscribe called in account store, no subscribers connected",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	if _, exists := a.subscribers[id]; exists {
		delete(a.subscribers, id)

		a.log.Debug("Un-subscribe called in account store, subscriber removed",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	a.log.Warn("Un-subscribe called in account store, subscriber does not exist",
		logging.Uint64("subscriber-id", id))

	return fmt.Errorf("subscriber to NoopAccount does not exist with id: %d", id)
}
