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

package vegatime

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/vegatime TimeService
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

	listeners []func(context.Context, time.Time)
	mu        sync.RWMutex

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
