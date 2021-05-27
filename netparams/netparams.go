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
	AddRules(...interface{}) error
	GetDispatch() func(context.Context, interface{}) error
	CheckDispatch(interface{}) error
}

type WatchParam struct {
	Param string
	// this is to be cast to a function accepting the
	// inner type of the parameters
	// e.g: for a String value, the expected function
	// is to be of the type: func(string) error
	Watcher interface{}
}

type Store struct {
	log    *logging.Logger
	cfg    Config
	store  map[string]value
	mu     sync.RWMutex
	broker Broker

	watchers     map[string][]WatchParam
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
		watchers:     map[string][]WatchParam{},
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

	// now we are going to iterate over ALL the netparams,
	// and run validation, so we will now if any was forgotten,
	// and left to a default which required explicit UponGenesis
	// through the genesis block
	for k := range AllKeys {
		v, err := s.Get(k)
		if err != nil {
			return fmt.Errorf("%v: %v", k, err)
		}
		if err := s.Validate(k, v); err != nil {
			return fmt.Errorf("%v: %v", k, err)
		}
	}

	// now we can iterate again over ALL the net params,
	// and dispatch the value of them all so any watchers can get updated
	// with genesis values
	for k := range s.store {
		if err := s.dispatchUpdate(ctx, k); err != nil {
			return fmt.Errorf("could not propagate netparams update to listener, %v: %v", k, err)
		}
	}

	return nil
}

// Watch a list of parameters updates
func (s *Store) Watch(wp ...WatchParam) error {
	for _, v := range wp {
		// type check the function to dispatch updates to
		if err := s.store[v.Param].CheckDispatch(v.Watcher); err != nil {
			return err
		}
		if watchers, ok := s.watchers[v.Param]; ok {
			s.watchers[v.Param] = append(watchers, v)
		} else {
			s.watchers[v.Param] = []WatchParam{v}
		}
	}
	return nil
}

// dispatch the update of a network parameters to all the listeners
func (s *Store) dispatchUpdate(ctx context.Context, p string) error {
	val := s.store[p]
	fn := val.GetDispatch()

	var err error
	for _, v := range s.watchers[p] {
		if newerr := fn(ctx, v.Watcher); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}
	return err
}

// OnChainTimeUpdate is trigger once per blocks
// we will send parameters update to watchers
func (s *Store) OnChainTimeUpdate(ctx context.Context, _ time.Time) {
	if len(s.paramUpdates) <= 0 {
		return
	}
	for k := range s.paramUpdates {
		if err := s.dispatchUpdate(ctx, k); err != nil {
			s.log.Debug("unable to dispatch netparams update", logging.Error(err))
		}
	}
	s.paramUpdates = map[string]struct{}{}
}

func (s *Store) DispatchChanges(ctx context.Context) {
	if len(s.paramUpdates) <= 0 {
		return
	}
	for k := range s.paramUpdates {
		if err := s.dispatchUpdate(ctx, k); err != nil {
			s.log.Debug("unable to dispatch netparams update", logging.Error(err))
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
	if err := svalue.Validate(value); err != nil {
		return fmt.Errorf("unable to validate %s: %w", key, err)
	}
	return nil
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
		return fmt.Errorf("unable to update %s: %w", key, err)
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

func (s *Store) AddRules(params ...AddParamRules) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range params {
		value, ok := s.store[v.Param]
		if !ok {
			return ErrUnknownKey
		}
		if err := value.AddRules(v.Rules...); err != nil {
			return err
		}
	}
	return nil
}

type AddParamRules struct {
	Param string
	Rules []interface{}
}

func ParamStringRules(key string, rules ...StringRule) AddParamRules {
	irules := []interface{}{}
	for _, v := range rules {
		irules = append(irules, v)
	}
	return AddParamRules{
		Param: key,
		Rules: irules,
	}
}

func ParamFloatRules(key string, rules ...FloatRule) AddParamRules {
	irules := []interface{}{}
	for _, v := range rules {
		irules = append(irules, v)
	}
	return AddParamRules{
		Param: key,
		Rules: irules,
	}
}

func ParamIntRules(key string, rules ...IntRule) AddParamRules {
	irules := []interface{}{}
	for _, v := range rules {
		irules = append(irules, v)
	}
	return AddParamRules{
		Param: key,
		Rules: irules,
	}
}

func ParamDurationRules(key string, rules ...DurationRule) AddParamRules {
	irules := []interface{}{}
	for _, v := range rules {
		irules = append(irules, v)
	}
	return AddParamRules{
		Param: key,
		Rules: irules,
	}
}

func ParamJSONRules(key string, rules ...JSONRule) AddParamRules {
	irules := []interface{}{}
	for _, v := range rules {
		irules = append(irules, v)
	}
	return AddParamRules{
		Param: key,
		Rules: irules,
	}
}
