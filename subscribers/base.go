package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/events"
)

type Base struct {
	ctx     context.Context
	cfunc   context.CancelFunc
	sCh     chan struct{}
	ch      chan events.Event
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
		ch:      make(chan events.Event, buf),
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

// Ack returns whether or not this is a synchronous/async subscriber
func (b *Base) Ack() bool {
	return b.ack
}

// Pause the current subscriber will not receive events from the channel
func (b *Base) Pause() {
	if b.running {
		b.running = false
		close(b.sCh)
	}
}

// Resume unpauzes the subscriber
func (b *Base) Resume() {
	if !b.running {
		b.sCh = make(chan struct{})
		b.running = true
	}
}

func (b Base) isRunning() bool {
	return b.running
}

// C returns the event channel for optional subscribers
func (b *Base) C() chan<- events.Event {
	return b.ch
}

// Closed indicates to the broker that the subscriber is closed for business
func (b *Base) Closed() <-chan struct{} {
	return b.ctx.Done()
}

// Skip lets the broker know that the subscriber is not receiving events
func (b *Base) Skip() <-chan struct{} {
	return b.sCh
}

// Halt is called by the broker on shutdown, this closes the open channels
func (b *Base) Halt() {
	b.cfunc()   // cancels the subscriber context, which breaks the loop
	b.Pause()   // close the skip channel
	close(b.ch) // close the event channel
}

// SetID set the ID (exposed only to broker)
func (b *Base) SetID(id int) {
	b.id = id
}

// ID returns the subscriber ID
func (b *Base) ID() int {
	return b.id
}
