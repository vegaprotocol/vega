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
	// we can ignore this value throughout the call-chain, but internally we have to account for it
	// this is equivalent to 0, but used for GQL mapping
	if batchSize == -1 {
		batchSize = 0
	}
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
	base := batchSize
	factor := len(expandedTypes)
	if expandedTypes == nil {
		// ~20 events
		trades = true
		factor = 20
	}
	// we've got filtering going on? increase internal buffer
	if len(filters) > 0 {
		factor *= 10
	}
	if base == 0 {
		base = 100
		if trades {
			base *= 2
		}
	}
	// @TODO find a more intellegent way than this messy thing. We don't want to allocate 2k events per stream unless we have to
	// but we don't want to run into issues because the event stream is using too small of a local buffer
	// size of a given batch and then some (in case batch size is increased
	bufLen := base * factor
	// not sure if we can even have a 0 bufLen here
	if bufLen == 0 {
		bufLen = 1000 * factor
		if trades {
			bufLen = 2000 * factor
		}
	}
	s := &StreamSub{
		Base:           NewBase(ctx, bufLen, false),
		mu:             &sync.Mutex{},
		types:          expandedTypes,
		data:           make([]StreamEvent, 0, bufLen), // cap to batch size
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
	if s.changeCount == 0 || s.changeCount < s.bufSize {
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
			s.Push(e...)
		}
	}
}

func (s *StreamSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	s.mu.Lock()
	// update channel is eligible for closing if no events are in buffer, or the nr of changes are less than the buffer size
	// closeUpdate := (s.changeCount == 0 || s.changeCount >= s.bufSize)
	closeUpdate := true
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
		select {
		case <-s.updated:
		default:
			close(s.updated)
		}
		//s.updated = make(chan struct{})
	}
	s.mu.Unlock()
}

// UpdateBatchSize changes the batch size, and returns whatever the current buffer contains
// it's effectively a poll of current events ignoring requested batch size
func (s *StreamSub) UpdateBatchSize(ctx context.Context, size int) []*types.BusEvent {
	s.mu.Lock()
	if size == s.bufSize {
		s.mu.Unlock()
		// this is equivalent to polling for data again, wait for the buffer to be full and return
		return s.GetData(ctx)
	}
	if len(s.data) == 0 {
		s.changeCount = 0
		if size != 0 {
			s.bufSize = size
		}
		s.mu.Unlock()
		return nil
	}
	s.changeCount = 0
	data := make([]StreamEvent, len(s.data))
	copy(data, s.data)
	dc := size
	if dc == 0 { // size == 0
		dc = cap(s.data)
	} else if size != s.bufSize { // size was not 0, reassign bufSize
		// buffer size changes
		s.bufSize = size
	}
	s.data = make([]StreamEvent, 0, dc)
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
func (s *StreamSub) GetData(ctx context.Context) []*types.BusEvent {
	select {
	case <-ctx.Done():
		// stream was closed
		return nil
	case <-s.updated:
		s.mu.Lock()
		// create new channel
		s.updated = make(chan struct{})
	}
	dl := len(s.data)
	// this seems to happen with a buffer of 1 sometimes
	// or could be an issue if s.updated was closed, but the UpdateBatchSize call acquired a lock first
	if dl < s.bufSize || dl == 0 {
		// data was drained (possibly UpdateBatchSize), so create new updated channel and carry on as if nothing happened
		s.mu.Unlock()
		return nil
	}
	s.changeCount = 0
	c := s.bufSize
	if c == 0 {
		c = dl
	}
	// copy the data for return, clear the internal slice
	data := make([]StreamEvent, c)
	copy(data, s.data)
	if s.bufSize == 0 {
		// if we use s.data = s.data[:0] here, we get a data race somehow
		s.data = s.data[:0]
	} else if len(s.data) == s.bufSize {
		s.data = s.data[:0]
	} else {
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
