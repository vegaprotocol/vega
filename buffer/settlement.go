package buffer

import (
	"sync"

	types "code.vegaprotocol.io/vega/proto"
)

type settleConf func(s *Settlement)

type Settlement struct {
	mu    *sync.Mutex
	chBuf int
	buf   map[string]map[string]types.Position
	chans map[int]chan map[string]map[string]types.Position
	free  []int
}

// ChannelBuffer - set default channel buffers to b (default 1)
func ChannelBuffer(b int) settleConf {
	return func(s *Settlement) {
		s.chBuf = b
	}
}

// New - create new settlement buffer
func New(opts ...settleConf) *Settlement {
	s := &Settlement{
		mu:    &sync.Mutex{},
		chBuf: 1,
		buf:   map[string]map[string]types.Position{},
		chans: map[int]chan map[string]map[string]types.Position{},
		free:  []int{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Add - add position data to the buffer
func (s *Settlement) Add(p types.Position) {
	s.mu.Lock()
	if _, ok := s.buf[p.MarketID]; !ok {
		s.buf[p.MarketID] = map[string]types.Position{}
	}
	s.buf[p.MarketID][p.PartyID] = p
	s.mu.Unlock()
}

// Flush - Clear buffer, passing all channels the data
func (s *Settlement) Flush() {
	s.mu.Lock()
	buf := s.buf
	s.buf = map[string]map[string]types.Position{}
	for _, ch := range s.chans {
		ch <- buf
	}
	s.mu.Unlock()
}

// Subscribe - get a channel to get the data from this buffer on flush
func (s *Settlement) Subscirbe() (<-chan map[string]map[string]types.Position, int) {
	s.mu.Lock()
	k := s.getKey()
	ch := make(chan map[string]map[string]types.Position, s.chBuf)
	s.chans[k] = ch
	s.mu.Unlock()
	return ch, k
}

// Unsubscribe - close channel and remove from active duty
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
	if len(s.free) != 0 {
		k := s.free[0]
		// remove first element
		s.free = s.free[1:]
		return k
	}
	// no available keys, the next available one == length of the chans map
	return len(s.chans)
}
