package v2

import "time"

type StdTime struct{}

func (t *StdTime) Now() time.Time {
	return time.Now()
}

func NewStdTime() *StdTime {
	return &StdTime{}
}
