package buffer

import (
	"sync"

	"code.vegaprotocol.io/vega/events"
)

type settleConf func(s *Settlement)

type Settlement struct {
	mu    *sync.Mutex
	chBuf int
	buf   map[string]map[string]events.SettlePosition
	chans map[int]chan []events.SettlePosition
	// chans map[int]chan map[string]map[string]types.Position
	free []int
}

// ChannelBuffer - set default channel buffers to b (default 1)
func ChannelBuffer(b int) settleConf {
	return func(s *Settlement) {
		s.chBuf = b
	}
}

// New - create new settlement buffer
func NewSettlement(opts ...settleConf) *Settlement {
	s := &Settlement{
		mu:    &sync.Mutex{},
		chBuf: 1,
		buf:   map[string]map[string]events.SettlePosition{},
		chans: map[int]chan []events.SettlePosition{},
		free:  []int{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Add - add position data to the buffer
func (s *Settlement) Add(pos []events.SettlePosition) {
	s.mu.Lock()
	for _, p := range pos {
		mID := p.MarketID()
		if _, ok := s.buf[mID]; !ok {
			s.buf[mID] = map[string]events.SettlePosition{}
		}
		s.buf[mID][p.Party()] = p
	}
	s.mu.Unlock()
}

// Flush - Clear buffer, passing all channels the data
func (s *Settlement) Flush() {
	s.mu.Lock()
	buf := s.buf
	// we've processed the buffer, clear it
	s.buf = map[string]map[string]events.SettlePosition{}
	// no channels to push to, no need to create slice with data
	if len(s.chans) == 0 {
		s.mu.Unlock()
		return
	}
	// rough cap estimate: all markets * size of the first one
	size := len(buf)
	for _, m := range buf {
		size *= len(m)
		break
	}
	positions := make([]events.SettlePosition, 0, size)
	for _, traders := range buf {
		for _, t := range traders {
			positions = append(positions, t)
		}
	}
	// we've got the slice, now pass it on to all "listeners"
	for _, ch := range s.chans {
		ch <- positions
	}
	s.mu.Unlock()
}

// Subscribe - get a channel to get the data from this buffer on flush
func (s *Settlement) Subscribe() (<-chan []events.SettlePosition, int) {
	s.mu.Lock()
	k := s.getKey()
	ch := make(chan []events.SettlePosition, s.chBuf)
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
