package vegatime

import (
	"context"
	"sync"
	"time"
)

// Svc represents the Service managing time inside Vega.
// this is basically based on the time of the chain in use.
type Svc struct {
	config Config

	previousTimestamp time.Time
	currentTimestamp  time.Time

	listeners []func(context.Context, time.Time)
	mu        sync.Mutex
}

// New instantiates a new vegatime service
func New(conf Config) *Svc {
	return &Svc{config: conf}
}

// ReloadConf reload the configuration for the vegatime service
func (s *Svc) ReloadConf(conf Config) {
	// do nothing here, conf is not used for now
}

// SetTimeNow update the current time
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

	s.notify(ctx, t)
}

// GetTimeNow returns the current time in vega
func (s *Svc) GetTimeNow() (time.Time, error) {
	return s.currentTimestamp, nil
}

// NotifyOnTick allows other services to register a callback function
// which will be called once the vega time is updated (SetTimeNow is called)
func (s *Svc) NotifyOnTick(f func(context.Context, time.Time)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, f)
}

// GetTimeLastBatch returns the previous vega time
func (s *Svc) GetTimeLastBatch() (time.Time, error) {
	return s.previousTimestamp, nil
}

func (s *Svc) notify(ctx context.Context, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.listeners {
		f(ctx, t)
	}
}
