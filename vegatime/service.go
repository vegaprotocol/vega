// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package vegatime

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/vegatime TimeService
type TimeService interface {
	GetTimeNow() time.Time
	NotifyOnTick(...func(context.Context, time.Time))
}

type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// Svc represents the Service managing time inside Vega.
// this is basically based on the time of the chain in use.
type Svc struct {
	config Config

	previousTimestamp time.Time
	currentTimestamp  time.Time

	listeners      []func(context.Context, time.Time)
	stateListeners []func(context.Context, time.Time)
	mu             sync.RWMutex

	broker Broker
}

// New instantiates a new vegatime service.
func New(conf Config, broker Broker) *Svc {
	return &Svc{config: conf, broker: broker}
}

// ReloadConf reload the configuration for the vegatime service.
func (s *Svc) ReloadConf(conf Config) {
	// do nothing here, conf is not used for now
}

// SetTimeNow update the current time.
func (s *Svc) SetTimeNow(ctx context.Context, t time.Time) {
	// ensure the t is using UTC
	t = t.UTC()

	// We need to cache the last timestamp so we can distribute trades
	// in a block transaction evenly between last timestamp and current timestamp
	if s.currentTimestamp.Unix() > 0 {
		s.previousTimestamp = s.currentTimestamp
	}
	s.currentTimestamp = t

	// Ensure we always set previousTimestamp it'll be 0 on the first block transaction
	if s.previousTimestamp.Unix() < 1 {
		s.previousTimestamp = s.currentTimestamp
	}

	evt := events.NewTime(ctx, t)
	s.broker.Send(evt)
	s.notify(ctx, t)
}

// GetTimeNow returns the current time in vega.
func (s *Svc) GetTimeNow() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentTimestamp
}

// NotifyOnTick allows other services to register a callback function
// which will be called once the vega time is updated (SetTimeNow is called).
func (s *Svc) NotifyOnTick(callbacks ...func(context.Context, time.Time)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, callbacks...)
}

// GetTimeLastBatch returns the previous vega time.
func (s *Svc) GetTimeLastBatch() time.Time {
	return s.previousTimestamp
}

func (s *Svc) notify(ctx context.Context, t time.Time) {
	// Call listeners for triggering actions.
	for _, f := range s.listeners {
		f(ctx, t)
	}
}
