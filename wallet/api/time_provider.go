package api

import "time"

type TimeProvider interface {
	Now() time.Time
}

type StdTime struct{}

func (stdt *StdTime) Now() time.Time {
	return time.Now()
}

func extractTimeProvider(tp ...TimeProvider) TimeProvider {
	if len(tp) > 1 {
		panic("only one time provider allowed at most")
	}

	var t TimeProvider = &StdTime{}
	if len(tp) > 0 {
		t = tp[0]
	}

	return t
}
