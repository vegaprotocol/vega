// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package stubs

import (
	"context"
	"time"

	vegacontext "code.vegaprotocol.io/vega/core/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/core/libs/crypto"
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
	for _, scb := range scbs {
		t.subscribers = append(t.subscribers, scb)
	}
}

func (t *TimeStub) notify(context context.Context, newTime time.Time) {
	for _, subscriber := range t.subscribers {
		subscriber(context, newTime)
	}
}
