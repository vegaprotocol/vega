package stubs

import (
	"context"
	"time"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
)

type TimeStub struct {
	now         time.Time
	subscribers []func(context.Context, time.Time)
}

func NewTimeStub() *TimeStub {
	startTime, _ := time.Parse("2006-01-02T15:04:05Z", "2019-11-30T00:00:00Z")
	return &TimeStub{
		now: startTime,
	}
}

func (t *TimeStub) GetTimeNow() time.Time {
	return t.now
}

func (t *TimeStub) SetTimeNow(_ context.Context, newNow time.Time) {
	t.SetTime(newNow)
}

func (t *TimeStub) SetTime(newNow time.Time) {
	t.now = newNow
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	t.notify(ctx, t.now)
}

func (t *TimeStub) NotifyOnTick(f func(context.Context, time.Time)) {
	t.subscribers = append(t.subscribers, f)
}

func (t *TimeStub) notify(context context.Context, newTime time.Time) {
	for _, subscriber := range t.subscribers {
		subscriber(context, newTime)
	}
}
