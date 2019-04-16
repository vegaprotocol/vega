package vegatime

import (
	"sync"
	"time"
)

type Svc struct {
	config *Config

	previousTimestamp time.Time
	currentTimestamp  time.Time

	listeners []func(time.Time)
	mu        sync.Mutex
}

func NewService(conf *Config) *Svc {
	return &Svc{config: conf}
}

func (s *Svc) SetTimeNow(t time.Time) {
	// ensure the t is using UTC
	t = t.UTC()

	// We need to cache the last timestamp so we can distribute trades
	// in a block transaction evenly between last timestamp and current timestamp
	if s.currentTimestamp.Unix() > 0 {
		s.previousTimestamp = s.currentTimestamp
	}

	// Convert unix epoch+nanoseconds into the current UTC date and time
	// we could pass this in as a var but doing the conversion here isolates
	// it to this method
	// s.currentDatetime = epochTimeNano.Datetime().UTC()

	s.currentTimestamp = t

	// Ensure we always set previousTimestamp it'll be 0 on the first block transaction
	if s.previousTimestamp.Unix() < 1 {
		s.previousTimestamp = s.currentTimestamp
	}

	s.notify(t)
}

func (s *Svc) GetTimeNow() (time.Time, error) {
	return s.currentTimestamp, nil
}

func (s *Svc) notify(t time.Time) {
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, f := range s.listeners {
			go f(t)
		}
	}()
}

func (s *Svc) NotifyOnTick(f func(time.Time)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, f)
}

func (s *Svc) GetTimeLastBatch() (time.Time, error) {
	return s.previousTimestamp, nil
}
