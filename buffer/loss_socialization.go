package buffer

import (
	"sync"

	"code.vegaprotocol.io/vega/events"
)

type LossSocialization struct {
	mu    *sync.Mutex
	chBuf int
	buf   []events.LossSocialization
	chans map[int]chan []events.LossSocialization
	free  []int
}

// NewLossSocialization create new settlement buffer
func NewLossSocialization() *LossSocialization {
	s := &LossSocialization{
		mu:    &sync.Mutex{},
		chBuf: 1,
		buf:   []events.LossSocialization{},
		chans: map[int]chan []events.LossSocialization{},
		free:  []int{},
	}
	return s
}

// Add ...
func (s *LossSocialization) Add(buf []events.LossSocialization) {
	s.mu.Lock()
	s.buf = append(s.buf, buf...)
	s.mu.Unlock()
}

// Flush Clear buffer, passing all channels the data
func (s *LossSocialization) Flush() {
	s.mu.Lock()
	buf := s.buf
	// we've processed the buffer, clear it
	s.buf = []events.LossSocialization{}
	// no channels to push to, no need to create slice with data
	if len(s.chans) == 0 {
		s.mu.Unlock()
		return
	}
	for _, ch := range s.chans {
		ch <- buf
	}
	s.mu.Unlock()
}

// Subscribe get a channel to get the data from this buffer on flush
func (s *LossSocialization) Subscribe() (<-chan []events.LossSocialization, int) {
	s.mu.Lock()
	k := s.getKey()
	ch := make(chan []events.LossSocialization, s.chBuf)
	s.chans[k] = ch
	s.mu.Unlock()
	return ch, k
}

// Unsubscribe close channel and remove from active duty
func (s *LossSocialization) Unsubscribe(k int) {
	s.mu.Lock()
	if ch, ok := s.chans[k]; ok {
		close(ch)
		// mark this key as available
		s.free = append(s.free, k)
	}
	delete(s.chans, k)
	s.mu.Unlock()
}

func (s *LossSocialization) getKey() int {
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
