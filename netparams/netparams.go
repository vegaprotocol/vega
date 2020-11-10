package netparams

import (
	"context"
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

type Reset interface {
	Reset()
}

type value interface {
	Validate(value string) error
	Update(value string) error
	String() string
	ToFloat() (float64, error)
	ToInt() (int64, error)
	ToUint() (uint64, error)
	ToBool() (bool, error)
	ToString() (string, error)
	ToDuration() (time.Duration, error)
	ToJSONStruct(Reset) error
}

type NetParamWatcher func(string, string)

type WatchParam struct {
	param   string
	watcher NetParamWatcher
}

type Store struct {
	log    *logging.Logger
	cfg    Config
	store  map[string]value
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
func (s *Store) UponGenesis(ctx context.Context, rawState []byte) error {
	s.log.Debug("loading genesis configuration")
	state, err := LoadGenesisState(rawState)
	if err != nil {
		s.log.Error("unable to load genesis state",
			logging.Error(err))
		return err
	}

	// first we going to send the initial state through the broker
	for k, v := range s.store {
		s.broker.Send(events.NewNetworkParameterEvent(ctx, k, v.String()))
	}

	// now iterate overal parameters and update the existing ones
	for k, v := range state {
		if err := s.Update(ctx, k, v); err != nil {
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
	for k := range s.paramUpdates {
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
func (s *Store) Update(ctx context.Context, key, value string) error {
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
	// and also send it to the broker
	s.broker.Send(events.NewNetworkParameterEvent(ctx, key, value))

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

// GetFloat a value associated to the given key
func (s *Store) GetFloat(key string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return 0, ErrUnknownKey
	}
	return svalue.ToFloat()
}

// GetInt a value associated to the given key
func (s *Store) GetInt(key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return 0, ErrUnknownKey
	}
	return svalue.ToInt()
}

// GetUint a value associated to the given key
func (s *Store) GetUint(key string) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return 0, ErrUnknownKey
	}
	return svalue.ToUint()
}

// GetBool a value associated to the given key
func (s *Store) GetBool(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return false, ErrUnknownKey
	}
	return svalue.ToBool()
}

// GetDuration a value associated to the given key
func (s *Store) GetDuration(key string) (time.Duration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return 0, ErrUnknownKey
	}
	return svalue.ToDuration()
}

// GetString a value associated to the given key
func (s *Store) GetString(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return "", ErrUnknownKey
	}
	return svalue.ToString()
}

// GetJSONStruct a value associated to the given key
func (s *Store) GetJSONStruct(key string, v Reset) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return ErrUnknownKey
	}
	return svalue.ToJSONStruct(v)
}
