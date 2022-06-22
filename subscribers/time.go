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

package subscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
)

// TimeE - TimeEvent
type TimeE interface {
	Time() time.Time
}

type TimeService interface {
	SetTimeNow(context.Context, time.Time)
}

type Time struct {
	*Base
	tsvc TimeService
	log  *logging.Logger
}

func NewTimeSub(
	ctx context.Context,
	tsvc TimeService,
	log *logging.Logger,
	ack bool,
) *Time {
	t := &Time{
		Base: NewBase(ctx, 1, ack),
		tsvc: tsvc,
		log:  log,
	}
	if t.isRunning() {
		go t.loop(t.ctx)
	}
	return t
}

func (t *Time) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			t.Halt()
			return
		case e := <-t.ch:
			if t.isRunning() {
				t.Push(e...)
			}
		}
	}
}

func (t *Time) Push(evts ...events.Event) {
	for _, e := range evts {
		switch et := e.(type) {
		case TimeE:
			t.tsvc.SetTimeNow(context.Background(), et.Time())
		default:
			t.log.Panic("Unknown event type in time subscriber",
				logging.String("Type", et.Type().String()))
		}
	}
}

func (t *Time) Types() []events.Type {
	return []events.Type{
		events.TimeUpdate,
	}
}
