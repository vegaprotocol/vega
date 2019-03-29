package vegatime

import (
	"time"
)

type Svc struct {
	config            *Config
	previousTimestamp Stamp
	currentTimestamp  Stamp
	previousDatetime  time.Time
	currentDatetime   time.Time
}

func NewService(conf *Config) *Svc {
	return &Svc{config: conf}
}

func (s *Svc) SetTimeNow(epochTimeNano Stamp) {

	// We need to cache the last timestamp so we can distribute trades
	// in a block transaction evenly between last timestamp and current timestamp
	if s.currentTimestamp > 0 {
		s.previousTimestamp = s.currentTimestamp
	}

	// Convert unix epoch+nanoseconds into the current UTC date and time
	// we could pass this in as a var but doing the conversion here isolates
	// it to this method
	s.currentDatetime = epochTimeNano.Datetime().UTC()
	s.currentTimestamp = epochTimeNano

	// Ensure we always set previousTimestamp it'll be 0 on the first block transaction
	if s.previousTimestamp < 1 {
		s.previousTimestamp = s.currentTimestamp
	}
}

func (s *Svc) GetTimeNow() (epochTimeNano Stamp, datetime time.Time, err error) {
	return s.currentTimestamp, s.currentDatetime, nil
}

func (s *Svc) GetTimeLastBatch() (epochTimeNano Stamp, datetime time.Time, err error) {
	return s.previousTimestamp, s.previousDatetime, nil
}
