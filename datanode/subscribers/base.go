// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/core/events"
)

type Base struct {
	ctx     context.Context
	cfunc   context.CancelFunc
	sCh     chan struct{}
	ch      chan []events.Event
	ack     bool
	running bool
	id      int
}

func NewBase(ctx context.Context, buf int, ack bool) *Base {
	ctx, cfunc := context.WithCancel(ctx)
	b := &Base{
		ctx:     ctx,
		cfunc:   cfunc,
		sCh:     make(chan struct{}),
		ch:      make(chan []events.Event, buf),
		ack:     ack,
		running: !ack, // assume the implementation will start a routine asap
	}
	if b.ack {
		go b.cleanup()
	}
	return b
}

func (b *Base) cleanup() {
	<-b.ctx.Done()
	b.Halt()
}

// Ack returns whether or not this is a synchronous/async subscriber.
func (b *Base) Ack() bool {
	return b.ack
}

// Pause the current subscriber will not receive events from the channel.
func (b *Base) Pause() {
	if b.running {
		b.running = false
		close(b.sCh)
	}
}

// Resume unpauzes the subscriber.
func (b *Base) Resume() {
	if !b.running {
		b.sCh = make(chan struct{})
		b.running = true
	}
}

func (b Base) isRunning() bool {
	return b.running
}

// C returns the event channel for optional subscribers.
func (b *Base) C() chan<- []events.Event {
	return b.ch
}

// Closed indicates to the broker that the subscriber is closed for business.
func (b *Base) Closed() <-chan struct{} {
	return b.ctx.Done()
}

// Skip lets the broker know that the subscriber is not receiving events.
func (b *Base) Skip() <-chan struct{} {
	return b.sCh
}

// Halt is called by the broker on shutdown, this closes the open channels.
func (b *Base) Halt() {
	// This is a hacky fix, but the fact remains: closing this channel occasionally causes a data race
	// we're using select, hoist the channel assignment, but the fact is: select is not atomic
	// allow attempted writes during shutdown, unless this is an acking sub, with a potential blocking channel
	defer func() {
		if !b.ack {
			time.Sleep(20 * time.Millisecond) // add sleep to avoid race (send on closed channel), 20ms should be plenty
		}
		close(b.ch) // close the event channel after pause (skip) and cfunc (closed) are toggled
	}()
	b.cfunc() // cancels the subscriber context, which breaks the loop
	b.Pause() // close the skip channel
}

// SetID set the ID (exposed only to broker).
func (b *Base) SetID(id int) {
	b.id = id
}

// ID returns the subscriber ID.
func (b *Base) ID() int {
	return b.id
}
