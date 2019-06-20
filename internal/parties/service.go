package parties

import (
	"context"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/part_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/parties PartyStore
type PartyStore interface {
	Post(party *types.Party) error
	GetByID(id string) (*types.Party, error)
	GetAll() ([]*types.Party, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/internal/orders  Blockchain
type Blockchain interface {
	NotifyTraderAccount(ctx context.Context, notif *types.NotifyTraderAccount) (success bool, err error)
}

type Svc struct {
	Config
	log   *logging.Logger
	store PartyStore
}

// NewService creates a Parties service with the necessary dependencies
func NewService(log *logging.Logger, config Config, store PartyStore) (*Svc, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:    log,
		Config: config,
		store:  store,
	}, nil
}

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

// CreateParty stores the given party.
func (s *Svc) CreateParty(ctx context.Context, party *types.Party) error {
	return s.store.Post(party)
}

// GetByID searches for the given party by id.
func (s *Svc) GetByID(ctx context.Context, id string) (*types.Party, error) {
	return s.store.GetByID(id)
}

// GetAll returns all parties.
func (s *Svc) GetAll(ctx context.Context) ([]*types.Party, error) {
	return s.store.GetAll()
}
