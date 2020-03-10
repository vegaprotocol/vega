package buffer

import (
	"sync"

	"github.com/tendermint/tendermint/types"
)

// Vote buffer
type Vote struct {
	mu    sync.Mutex
	chBuf int
	buf   []types.Vote
	chans map[int]chan []types.Vote
	free  []int
}

// NewVotes creates a new vote buffer
func NewVotes() *Vote {
	return &Vote{
		chBuf: 1,
		buf:   []types.Vote{},
		chans: map[int]chan []types.Vote{},
		free:  []int{},
	}
}

// Add a vote to the buffer
func (v *Vote) Add(vote types.Vote) {
	v.buf = append(v.buf, vote)
}

// Flush the buffer
func (v *Vote) Flush() {
	// create new slice, use cap of previous -> avoid allocations
	cpy := v.buf
	v.buf = make([]types.Vote, 0, cap(cpy))
	v.mu.Lock()
	for _, ch := range v.chans {
		ch <- cpy
	}
	c.mu.Unlock()
}

// Subscribe to the buffer, on flush, subscriptions will receive the data
func (v *Vote) Subscribe() (<-chan []types.Vote, int) {
	v.mu.Lock()
	ch := make(chan []types.Vote, v.chBuf)
	k := v.getKey()
	v.chans[k] = ch
	v.mu.Unlock()
	return ch, k
}

func (v *Vote) Unsubscribe(k int) {
	v.mu.Lock()
	if ch, ok := v.chans[k]; ok {
		close(ch)
		delete(v.chans, k)
		v.free = append(v.free, k)
	}
	v.mu.Unlock()
}

func (v *Vote) getKey() int {
	// no need to lock mutex, the caller should have the lock
	if len(v.free) != 0 {
		k := v.free[0]
		// remove first element
		v.free = v.free[1:]
		return k
	}
	// no available keys, the next available one == length of the chans map
	// add 1 to ensure we can't end up with a 0 ID
	return len(v.chans) + 1
}
