package stubs

import (
	"context"
	"time"
)

type TimeStub struct {
	Now    time.Time
	Notify func(context.Context, time.Time)
}

func (t *TimeStub) GetTimeNow() (time.Time, error) {
	return t.Now, nil
}

func (t *TimeStub) SetTime(newNow time.Time) {
	t.Now = newNow
	t.Notify(context.Background(), t.Now)
}

func (t *TimeStub) NotifyOnTick(f func(context.Context, time.Time)) {
	t.Notify = f
}