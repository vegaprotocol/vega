package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type EventFilter func(events.Event) bool

type StreamEvent interface {
	events.Event
	StreamMessage() *types.BusEvent
}

type MarketStreamEvent interface {
	StreamEvent
	StreamMarketMessage() *types.BusEvent
}

type StreamSub struct {
	*Base
	mu             *sync.Mutex // pointer because types is a value receiver, linter complains
	types          []events.Type
	data           []StreamEvent
	filters        []EventFilter
	bufSize        int
	changeCount    int
	updated        chan struct{}
	marketEvtsOnly bool
}

func NewStreamSub(ctx context.Context, types []events.Type, batchSize int, filters ...EventFilter) *StreamSub {
	trades, meo := false, (len(types) == 1 && types[0] == events.MarketEvent)
	expandedTypes := make([]events.Type, 0, len(types))
	for _, t := range types {
		if t == events.All {
			expandedTypes = nil
			break
		}
		if t == events.MarketEvent {
			expandedTypes = append(expandedTypes, events.MarketEvents()...)
		} else {
			if t == events.TradeEvent {
				trades = true
			}
			expandedTypes = append(expandedTypes, t)
		}
	}
	// size of a given batch and then some (in case batch size is increased
	bufLen := batchSize * 5
	if bufLen == 0 {
		bufLen := len(expandedTypes) * 10 // each type adds a buffer of 10
		if bufLen == 0 {
			// we're subscribing to all events, buffer should be way more than 0, obviously
			// there's roughly 20 event types, each need a buffer of at least 10
			// trades are a special case: there's potentially hundreds of trade events per block
			// wo we want to ensure our buffers are large enough for a normal trade volume
			trades = true
			bufLen = 200 // 20 event types, buffer of 10 each
		}
		if trades {
			bufLen += 100 // add buffer for 100 events
		}
	}
	s := &StreamSub{
		Base:           NewBase(ctx, bufLen, false),
		mu:             &sync.Mutex{},
		types:          expandedTypes,
		data:           make([]StreamEvent, 0, batchSize), // cap to batch size
		filters:        filters,
		bufSize:        batchSize,
		updated:        make(chan struct{}), // create a blocking channel for these
		marketEvtsOnly: meo,
	}
	// running or not, we're using the channel
	go s.loop(s.ctx)
	return s
}

func (s *StreamSub) Halt() {
	s.mu.Lock()
	if s.changeCount == 0 {
		close(s.updated)
	}
	s.mu.Unlock()
	s.Base.Halt() // close channel outside of the lock. to avoid race
}

func (s *StreamSub) loop(ctx context.Context) {
	s.running = true // allow for Pause to work (ensures the pause channel can, and will be closed)
	for {
		select {
		case <-ctx.Done():
			s.Halt()
			return
		case e, ok := <-s.ch:
			// just return if closed, don't call Halt, because that would try to close s.ch a second time
			if !ok {
				return
			}
			s.Push(e)
		}
	}
}

func (s *StreamSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	s.mu.Lock()
	// update channel is eligible for closing if no events are in buffer, or the nr of changes are less than the buffer size
	closeUpdate := (s.changeCount == 0 || s.changeCount < s.bufSize)
	save := make([]StreamEvent, 0, len(evts))
	for _, e := range evts {
		var se StreamEvent
		if s.marketEvtsOnly {
			// ensure we can get a market stream event from this
			me, ok := e.(MarketStreamEvent)
			if !ok {
				continue
			}
			se = me
		} else if ste, ok := e.(StreamEvent); ok {
			se = ste
		} else {
			continue
		}
		keep := true
		for _, f := range s.filters {
			if !f(e) {
				keep = false
				break
			}
		}
		if keep {
			save = append(save, se)
		}
	}
	s.changeCount += len(save)
	s.data = append(s.data, save...)
	if closeUpdate && ((s.bufSize > 0 && s.changeCount >= s.bufSize) || (s.bufSize == 0 && s.changeCount > 0)) {
		close(s.updated)
	}
	s.mu.Unlock()
}

// UpdateBatchSize changes the batch size, and returns whatever the current buffer contains
// it's effectively a poll of current events ignoring requested batch size
func (s *StreamSub) UpdateBatchSize(size int) []*types.BusEvent {
	s.mu.Lock()
	s.changeCount = 0
	data := s.data
	dc := size
	if dc == 0 {
		dc = cap(s.data)
	}
	s.data = make([]StreamEvent, 0, dc)
	s.bufSize = size
	s.mu.Unlock()
	messages := make([]*types.BusEvent, 0, len(data))
	for _, d := range data {
		if s.marketEvtsOnly {
			e, ok := d.(MarketStreamEvent)
			if ok {
				messages = append(messages, e.StreamMarketMessage())
			}
		} else {
			messages = append(messages, d.StreamMessage())
		}
	}
	return messages
}

// GetData returns events from buffer, all if bufSize == 0, or max buffer size (rest are kept in data slice)
func (s *StreamSub) GetData() []*types.BusEvent {
	<-s.updated
	s.mu.Lock()
	// create a new update channel + reset update counter
	s.updated = make(chan struct{})
	// this seems to happen with a buffer of 1 sometimes
	// or could be an issue if s.updated was closed, but the UpdateBatchSize call acquired a lock first
	if len(s.data) < s.bufSize {
		// data was drained (possibly UpdateBatchSize), so create new updated channel and carry on as if nothing happened
		s.mu.Unlock()
		return nil
	}
	dc := s.bufSize
	s.changeCount = 0
	// copy the data for return, clear the internal slice
	data := s.data
	if s.bufSize == 0 {
		dc = cap(s.data)
		s.data = make([]StreamEvent, 0, dc)
	} else if len(s.data) == s.bufSize {
		// data var is exactly the right size, data has to be emptied, allocate new slice
		s.data = make([]StreamEvent, 0, dc)
	} else {
		data = data[:s.bufSize]     // only get the batch requested
		s.data = s.data[s.bufSize:] // leave rest in the buffer
		s.changeCount = len(s.data) // keep change count in sync with data slice
	}
	s.mu.Unlock()
	messages := make([]*types.BusEvent, 0, len(data))
	for _, d := range data {
		if s.marketEvtsOnly {
			e := d.(MarketStreamEvent) // we know this works already
			messages = append(messages, e.StreamMessage())
		} else {
			messages = append(messages, d.StreamMessage())
		}
	}
	return messages
}

func (s StreamSub) Types() []events.Type {
	return s.types
}
