package netparams

import (
	"errors"
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrUnknownKey = errors.New("unknown key")
)

// Broker - event bus
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/netparams Broker
type Broker interface {
	Send(e events.Event)
}

type Value interface {
	Validate(value string) error
	Update(value string) error
	String() string
}

type Store struct {
	log    *logging.Logger
	cfg    Config
	store  map[string]Value
	mu     sync.RWMutex
	broker Broker
}

func New(log *logging.Logger, cfg Config, broker Broker) *Store {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	return &Store{
		log:    log,
		cfg:    cfg,
		store:  defaultNetParams(),
		broker: broker,
	}
}

// Validate will call validation on the Value stored
// for the given key.
func (s *Store) Validate(key, value string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return ErrUnknownKey
	}
	return svalue.Validate(value)
}

// Update will update the stored value for a given key
// will return an error if the value do not pass validation
func (s *Store) Update(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	svalue, ok := s.store[key]
	if !ok {
		return ErrUnknownKey
	}
	return svalue.Update(value)
}

// Exists check if a value exist for the given key
func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.store[key]
	return ok
}

// Get a value associated to the given key
func (s *Store) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return "", ErrUnknownKey
	}
	return svalue.String(), nil
}
