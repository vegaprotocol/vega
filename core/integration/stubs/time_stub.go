// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

func (t *TimeStub) NotifyOnTick(scbs ...func(context.Context, time.Time)) {
	t.subscribers = append(t.subscribers, scbs...)
}

func (t *TimeStub) notify(context context.Context, newTime time.Time) {
	for _, subscriber := range t.subscribers {
		subscriber(context, newTime)
	}
}
