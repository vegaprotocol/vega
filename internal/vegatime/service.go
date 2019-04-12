package vegatime

import (
	"time"
)

type Svc struct {
	config            *Config
	previousTimestamp time.Time
	currentTimestamp  time.Time
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
}

func (s *Svc) GetTimeNow() (time.Time, error) {
	return s.currentTimestamp, nil
}

func (s *Svc) GetTimeLastBatch() (time.Time, error) {
	return s.previousTimestamp, nil
}
