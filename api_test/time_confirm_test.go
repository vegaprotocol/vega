package api_test

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"
)

type TimeEvent interface {
	Time() time.Time
}

type TimeSub struct {
	*subscribers.Base
	mu  *sync.Mutex
	ch  chan time.Time
	rCh chan struct{}
	rcv bool
}

func NewTimeSub(ctx context.Context) *TimeSub {
	t := &TimeSub{
		Base: subscribers.NewBase(ctx, 10, true),
		mu:   &sync.Mutex{},
		ch:   make(chan time.Time, 10),
		rCh:  make(chan struct{}),
	}
	return t
}

func (t *TimeSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ch == nil {
		panic("time channel is closed")
	}
	// lock now, this could be a batch in the future
	for _, e := range evts {
		switch et := e.(type) {
		case TimeEvent:
			if !t.rcv {
				close(t.rCh)
				t.rcv = true
			}
			t.ch <- et.Time()
		}
	}
}

func (t *TimeSub) GetReveivedTimes() []time.Time {
	<-t.rCh
	t.mu.Lock()
	// sub has been halted
	if t.ch == nil {
		t.mu.Unlock()
		return nil
	}
	ch := t.getTimeCh()
	// reset reveive channel
	t.rCh = make(chan struct{})
	t.rcv = false
	t.mu.Unlock()
	// this shouldn't be possible
	if ch == nil {
		return nil
	}
	r := make([]time.Time, 0, len(ch))
	for tm := range ch {
		r = append(r, tm)
	}
	return r
}

func (t *TimeSub) getTimeCh() <-chan time.Time {
	if len(t.ch) == 0 {
		return nil
	}
	ch := t.ch
	t.ch = make(chan time.Time, 10)
	close(ch)
	return ch
}

func (t *TimeSub) Halt() {
	// we don't need to call Halt on base
	// we're cancelling the context which takes care of the cleanup already
	// t.Base.Halt()
	t.mu.Lock()
	close(t.ch)
	t.ch = nil
	if !t.rcv {
		close(t.rCh)
	}
	t.rCh = nil
	t.mu.Unlock()
}

func (*TimeSub) Types() []events.Type {
	return []events.Type{
		events.TimeUpdate,
	}
}
