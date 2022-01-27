package stubs

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/oracles"
)

type TimeStub struct {
	now                       time.Time
	subscribers               []func(context.Context, time.Time)
	internalOracleSubscribers []func(context.Context, oracles.OracleData)
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

func (t *TimeStub) SetTime(newNow time.Time) {
	t.now = newNow
	t.notify(context.Background(), t.now)
	t.publishOracleData(context.Background(), t.now)
}

func (t *TimeStub) NotifyOnTick(f func(context.Context, time.Time)) {
	t.subscribers = append(t.subscribers, f)
}

func (t *TimeStub) notify(context context.Context, newTime time.Time) {
	for _, subscriber := range t.subscribers {
		subscriber(context, newTime)
	}
}

func (t *TimeStub) publishOracleData(ctx context.Context, ts time.Time) {
	for _, s := range t.internalOracleSubscribers {
		data := oracles.OracleData{
			Data: map[string]string{
				oracles.BuiltinOracleTimestamp: fmt.Sprintf("%d", ts.UnixNano()),
			},
		}
		s(ctx, data)
	}
}
