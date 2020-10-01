package netparams

import (
	"errors"
	"fmt"
	"sync"
	"time"

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

type NetParamWatcher func(string, string)

type WatchParam struct {
	param   string
	watcher NetParamWatcher
}

type Store struct {
	log    *logging.Logger
	cfg    Config
	store  map[string]Value
	mu     sync.RWMutex
	broker Broker

	watchers     map[string][]NetParamWatcher
	paramUpdates map[string]struct{}
}

func New(log *logging.Logger, cfg Config, broker Broker) *Store {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	return &Store{
		log:          log,
		cfg:          cfg,
		store:        defaultNetParams(),
		broker:       broker,
		watchers:     map[string][]NetParamWatcher{},
		paramUpdates: map[string]struct{}{},
	}
}

// UponGenesis load the initial network parameters
// from the genesis state
func (s *Store) UponGenesis(rawState []byte) error {
	s.log.Debug("loading genesis configuration")
	state, err := LoadGenesisState(rawState)
	if err != nil {
		s.log.Error("unable to load genesis state",
			logging.Error(err))
		return err
	}

	// now iterate overal parameters and update the existing ones
	for k, v := range state {
		if err := s.Update(k, v); err != nil {
			return fmt.Errorf("%v: %v", k, err)
		}
	}

	return nil
}

// Watch a list of parameters updates
func (s *Store) Watch(wp ...WatchParam) {
	for _, v := range wp {
		if watchers, ok := s.watchers[v.param]; ok {
			s.watchers[v.param] = append(watchers, v.watcher)
		} else {
			s.watchers[v.param] = []NetParamWatcher{v.watcher}
		}
	}
}

// OnChainTimeUpdate is trigger once per blocks
// we will send parameters update to watchers
func (s *Store) OnChainTimeUpdate(_ time.Time) {
	if len(s.paramUpdates) <= 0 {
		return
	}
	for k, _ := range s.paramUpdates {
		val, _ := s.Get(k)
		for _, w := range s.watchers[k] {
			w(k, val)
		}
	}
	s.paramUpdates = map[string]struct{}{}
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

	if err := svalue.Update(value); err != nil {
		return err
	}

	// update was successful we want to notify watchers
	s.paramUpdates[key] = struct{}{}

	return nil
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
