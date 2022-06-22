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

package api_test

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/vega/events"
)

type EventSubscriber struct {
	*subscribers.Base
	events chan events.Event

	closed bool
	mu     sync.RWMutex
}

func NewEventSubscriber(ctx context.Context) *EventSubscriber {
	t := &EventSubscriber{
		Base:   subscribers.NewBase(ctx, 10, true),
		events: make(chan events.Event, 20),
	}

	return t
}

func (t *EventSubscriber) Push(evts ...events.Event) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return
	}

	for _, e := range evts {
		t.events <- e
	}
}

func (t *EventSubscriber) ReceivedEvent(ctx context.Context) (events.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case e := <-t.events:
		return e, nil
	}
}

func (t *EventSubscriber) Halt() {
	t.mu.Lock()
	close(t.events)
	t.closed = true
	t.mu.Unlock()
}

func (*EventSubscriber) Types() []events.Type {
	return []events.Type{}
}
