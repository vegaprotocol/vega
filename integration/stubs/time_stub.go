package stubs

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"

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
	ctx := vegacontext.WithTraceID(context.Background(), randomSha256Hash())
	t.notify(ctx, t.now)
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
	for _, subscriber := range t.internalOracleSubscribers {
		data := oracles.OracleData{
			Data: map[string]string{
				oracles.BuiltinOracleTimestamp: fmt.Sprintf("%d", ts.UnixNano()),
			},
		}
		subscriber(ctx, data)
	}
}

func randomSha256Hash() string {
	data := make([]byte, 10)
	rand.Read(data)
	return hex.EncodeToString(crypto.Hash(data))
}
