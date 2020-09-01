package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
)

type EventFilter func(events.Event) bool

type StreamSub struct {
	*Base
	mu          *sync.Mutex // pointer because types is a value receiver, linter complains
	types       []events.Type
	data        []events.Event
	filters     []EventFilter
	changeCount int
	updated     chan struct{}
}

func NewStreamSub(ctx context.Context, types []events.Type, filters ...EventFilter) *StreamSub {
	s := &StreamSub{
		Base:    NewBase(ctx, len(types), false),
		mu:      &sync.Mutex{},
		types:   types,
		data:    []events.Event{},
		filters: filters,
		updated: make(chan struct{}), // create a blocking channel for these
	}
	if s.isRunning() {
		go s.loop(s.ctx)
	}
	return s
}

func (s *StreamSub) Halt() {
	s.mu.Lock()
	if s.changeCount == 0 {
		close(s.updated)
	}
	s.mu.Unlock()
	s.Base.Halt()
}

func (s *StreamSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.Halt()
			return
		case e := <-s.ch:
			if s.isRunning() {
				s.Push(e)
			}
		}
	}
}

func (s *StreamSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	s.mu.Lock()
	closeUpdate := (s.changeCount == 0)
	save := make([]events.Event, 0, len(evts))
	for _, e := range evts {
		keep := true
		for _, f := range s.filters {
			if !f(e) {
				keep = false
				break
			}
		}
		if keep {
			save = append(save, e)
		}
	}
	s.changeCount += len(save)
	if closeUpdate && s.changeCount > 0 {
		close(s.updated)
	}
	s.data = append(s.data, save...)
	s.mu.Unlock()
}

func (s *StreamSub) GetData() []events.Event {
	<-s.updated
	s.mu.Lock()
	// create a new update channel + reset update counter
	s.updated = make(chan struct{})
	s.changeCount = 0
	// copy the data for return, clear the internal slice
	data := s.data
	s.data = make([]events.Event, 0, cap(data))
	s.mu.Unlock()
	return data
}

func (s StreamSub) Types() []events.Type {
	return s.types
}
