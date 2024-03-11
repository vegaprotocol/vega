// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package netparams

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"golang.org/x/exp/maps"
)

var (
	ErrUnknownKey                        = errors.New("unknown key")
	ErrNetworkParameterUpdateDisabledFor = func(key string) error {
		return fmt.Errorf("network parameter update disabled for %v", key)
	}
	ErrNetworkParameterDeprecated = func(key string) error {
		return fmt.Errorf("network parameter has been deprecated: %v", key)
	}
	// a list of network parameter which cannot be updated.
	updateDisallowed = []string{
		BlockchainsPrimaryEthereumConfig,
		BlockchainsSecondaryEthereumConfig,
	}
)

// Broker - event bus.
type Broker interface {
	Send(e events.Event)
	SendBatch(evts []events.Event)
}

type Reset interface {
	Reset()
}

//nolint:interfacebloat
type value interface {
	Validate(value string) error
	Update(value string) error
	UpdateOptionalValidation(value string, validate bool) error
	String() string
	ToDecimal() (num.Decimal, error)
	ToInt() (int64, error)
	ToUint() (*num.Uint, error)
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

	checkpointOverwrites map[string]struct{}

	state *snapState

	protocolUpgradeNewParameters []string
}

func New(log *logging.Logger, cfg Config, broker Broker) *Store {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	store := defaultNetParams()
	return &Store{
		log:                  log,
		cfg:                  cfg,
		store:                store,
		broker:               broker,
		watchers:             map[string][]WatchParam{},
		paramUpdates:         map[string]struct{}{},
		checkpointOverwrites: map[string]struct{}{},
		state:                newSnapState(store),
	}
}

// UponGenesis load the initial network parameters
// from the genesis state.
func (s *Store) UponGenesis(ctx context.Context, rawState []byte) (err error) {
	s.log.Debug("Entering netparams.Store.UponGenesis")
	defer func() {
		if err != nil {
			s.log.Debug("Failure in netparams.Store.UponGenesis", logging.Error(err))
		} else {
			s.log.Debug("Leaving netparams.Store.UponGenesis without error")
		}
	}()

	state, err := LoadGenesisState(rawState)
	if err != nil {
		s.log.Error("unable to load genesis state",
			logging.Error(err))
		return err
	}

	evts := make([]events.Event, 0, len(s.store))
	keys := maps.Keys(s.store)
	sort.Strings(keys)
	// first we going to send the initial state through the broker
	for _, k := range keys {
		evts = append(evts, events.NewNetworkParameterEvent(ctx, k, s.store[k].String()))
	}
	s.broker.SendBatch(evts)

	// now iterate over all parameters and update the existing ones
	keys = maps.Keys(state)
	sort.Strings(keys)
	for _, k := range keys {
		if err := s.UpdateOptionalValidation(ctx, k, state[k], false, true); err != nil {
			return fmt.Errorf("%v: %v", k, err)
		}
	}

	// now we are going to iterate over ALL the netparams,
	// and run validation, so we will know if any was forgotten,
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
	keys = maps.Keys(s.store)
	sort.Strings(keys)
	for _, k := range keys {
		if err := s.dispatchUpdate(ctx, k); err != nil {
			return fmt.Errorf("could not propagate netparams update to listener, %v: %v", k, err)
		}
	}

	overwrites, err := LoadGenesisStateOverwrite(rawState)
	if err != nil {
		s.log.Error("unable to load genesis state overwrites",
			logging.Error(err))
		return err
	}

	for _, v := range overwrites {
		if _, ok := AllKeys[v]; !ok {
			s.log.Error("unknown network parameter", logging.String("netp", v))
		}
		s.checkpointOverwrites[v] = struct{}{}
	}

	return nil
}

// Watch a list of parameters updates.
func (s *Store) Watch(wp ...WatchParam) error {
	for _, v := range wp {
		// type check the function to dispatch updates to
		if err := s.store[v.Param].CheckDispatch(v.Watcher); err != nil {
			return fmt.Errorf("%v: %v", v.Param, err)
		}
		if watchers, ok := s.watchers[v.Param]; ok {
			s.watchers[v.Param] = append(watchers, v)
		} else {
			s.watchers[v.Param] = []WatchParam{v}
		}
	}
	return nil
}

// dispatch the update of a network parameters to all the listeners.
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

func (s *Store) AnyWatchers(p string) bool {
	return len(s.watchers[p]) > 0
}

// OnTick is trigger once per blocks
// we will send parameters update to watchers.
func (s *Store) OnTick(ctx context.Context, _ time.Time) {
	// This is useful only when a protocol upgrade
	// is running. we will dispatch all new parameter
	// on the first time update here.
	// we propagate all parameters update to paliate
	// for previous release where parameters didn't
	// get propagated.
	if len(s.protocolUpgradeNewParameters) > 0 {
		keys := maps.Keys(s.store)
		sort.Strings(keys)

		for _, k := range keys {
			s.broker.Send(events.NewNetworkParameterEvent(ctx, k, s.store[k].String()))
		}

		s.protocolUpgradeNewParameters = nil
	}

	if len(s.paramUpdates) <= 0 {
		return
	}

	// sort for deterministic order of processing.
	params := make([]string, 0, len(s.paramUpdates))
	for k := range s.paramUpdates {
		params = append(params, k)
	}
	sort.Strings(params)

	for _, k := range params {
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
// will return an error if the value do not pass validation.
func (s *Store) Update(ctx context.Context, key, value string) error {
	return s.UpdateOptionalValidation(ctx, key, value, true, true)
}

func (s *Store) UpdateOptionalValidation(ctx context.Context, key, value string, validate, failIfUnknown bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	svalue, ok := s.store[key]
	if !ok {
		if failIfUnknown {
			return ErrUnknownKey
		}

		s.log.Warn("unknown network parameter", logging.String("key", key))
		return nil
	}

	if err := svalue.UpdateOptionalValidation(value, validate); err != nil {
		return fmt.Errorf("unable to update %s: %w", key, err)
	}

	// update was successful we want to notify watchers
	s.paramUpdates[key] = struct{}{}
	// and also send it to the broker
	s.broker.Send(events.NewNetworkParameterEvent(ctx, key, value))
	s.state.update(key, value)

	return nil
}

func (s *Store) updateBatch(ctx context.Context, params map[string]string) error {
	evts := make([]events.Event, 0, len(params))
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := maps.Keys(params)
	sort.Strings(keys)
	for _, k := range keys {
		v := params[k]
		svalue, ok := s.store[k]
		if !ok {
			s.log.Warn("unknown network parameter read from checkpoint", logging.String("param", k))
			continue
		}
		if err := svalue.UpdateOptionalValidation(v, false); err != nil {
			return fmt.Errorf("unable to update %s: %w", k, err)
		}
		s.paramUpdates[k] = struct{}{}
		s.state.update(k, v)
		evts = append(evts, events.NewNetworkParameterEvent(ctx, k, v))
	}
	s.broker.SendBatch(evts)
	return nil
}

// Exists check if a value exist for the given key.
func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.store[key]
	return ok
}

// Get a value associated to the given key.
func (s *Store) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return "", ErrUnknownKey
	}
	return svalue.String(), nil
}

// GetDecimal a value associated to the given key.
func (s *Store) GetDecimal(key string) (num.Decimal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return num.DecimalZero(), ErrUnknownKey
	}
	return svalue.ToDecimal()
}

// GetInt a value associated to the given key.
func (s *Store) GetInt(key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return 0, ErrUnknownKey
	}
	return svalue.ToInt()
}

// GetUint a value associated to the given key.
func (s *Store) GetUint(key string) (*num.Uint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return num.UintZero(), ErrUnknownKey
	}
	v, err := svalue.ToUint()
	if err != nil {
		return num.UintZero(), err
	}
	return v.Clone(), nil
}

// GetBool a value associated to the given key.
func (s *Store) GetBool(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return false, ErrUnknownKey
	}
	return svalue.ToBool()
}

// GetDuration a value associated to the given key.
func (s *Store) GetDuration(key string) (time.Duration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return 0, ErrUnknownKey
	}
	return svalue.ToDuration()
}

// GetString a value associated to the given key.
func (s *Store) GetString(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svalue, ok := s.store[key]
	if !ok {
		return "", ErrUnknownKey
	}
	return svalue.ToString()
}

// GetJSONStruct a value associated to the given key.
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

func (s *Store) IsUpdateAllowed(key string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.store[key]
	if !ok {
		return ErrUnknownKey
	}

	if _, ok := Deprecated[key]; ok {
		return ErrNetworkParameterDeprecated(key)
	}

	for _, v := range updateDisallowed {
		if v == key {
			return ErrNetworkParameterUpdateDisabledFor(key)
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
