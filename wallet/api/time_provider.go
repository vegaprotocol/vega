package api

import "time"

type TimeProvider interface {
	Now() time.Time
}

type StdTime struct{}

func (stdt *StdTime) Now() time.Time {
	return time.Now()
}
