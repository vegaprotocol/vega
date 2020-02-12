package buffer

import (
	"sync"

	"code.vegaprotocol.io/vega/events"
)

type settleConf func(s *Settlement)

type Settlement struct {
	mu    *sync.Mutex
	chBuf int
	buf   []events.SettlePosition
	chans map[int]chan []events.SettlePosition
	// chans map[int]chan map[string]map[string]types.Position
	free []int
}

// ChannelBuffer set default channel buffers to b (default 1)
func ChannelBuffer(b int) settleConf {
	return func(s *Settlement) {
		s.chBuf = b
	}
}

// NewSettlement create new settlement buffer
func NewSettlement(opts ...settleConf) *Settlement {
	s := &Settlement{
		mu:    &sync.Mutex{},
		chBuf: 1,
		buf:   []events.SettlePosition{},
		chans: map[int]chan []events.SettlePosition{},
		free:  []int{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Add add position data to the buffer
func (s *Settlement) Add(pos []events.SettlePosition) {
	s.mu.Lock()
	s.buf = append(s.buf, pos...)
	s.mu.Unlock()
}

// Flush Clear buffer, passing all channels the data
func (s *Settlement) Flush() {
	s.mu.Lock()
	buf := s.buf
	// we've processed the buffer, clear it
	// instanciate a new buffer roughly of the size of the previous one
	// we can expect roughtly the same amount of event...
	s.buf = make([]events.SettlePosition, 0, len(buf))
	// no channels to push to, no need to create slice with data
	if len(s.chans) == 0 {
		s.mu.Unlock()
		return
	}
	// we've got the slice, now pass it on to all "listeners"
	for _, ch := range s.chans {
		ch <- buf
	}
	s.mu.Unlock()
}

// Subscribe get a channel to get the data from this buffer on flush
func (s *Settlement) Subscribe() (<-chan []events.SettlePosition, int) {
	s.mu.Lock()
	k := s.getKey()
	ch := make(chan []events.SettlePosition, s.chBuf)
	s.chans[k] = ch
	s.mu.Unlock()
	return ch, k
}

// Unsubscribe close channel and remove from active duty
func (s *Settlement) Unsubscribe(k int) {
	s.mu.Lock()
	if ch, ok := s.chans[k]; ok {
		close(ch)
		// mark this key as available
		s.free = append(s.free, k)
	}
	delete(s.chans, k)
	s.mu.Unlock()
}

func (s *Settlement) getKey() int {
	// no need to lock mutex, the caller should have the lock
	if len(s.free) != 0 {
		k := s.free[0]
		// remove first element
		s.free = s.free[1:]
		return k
	}
	// no available keys, the next available one == length of the chans map
	return len(s.chans)
}
