package buffer

import (
	"sync"

	types "code.vegaprotocol.io/vega/proto"
)

// Proposal - buffers all proposal changes in a map by ID (so we only preserve the last state)
type Proposal struct {
	mu    sync.Mutex
	chBuf int
	buf   map[string]types.Proposal
	chans map[int]chan []types.Proposal
	free  []int
}

// NewProposal get a new proposal buffer
func NewProposal() *Proposal {
	return &Proposal{
		chBuf: 1,
		buf:   map[string]types.Proposal{},
		chans: map[int]chan []types.Proposal{},
		free:  []int{},
	}
}

// Add a single proposal to the buffer
func (p *Proposal) Add(prop types.Proposal) {
	p.buf[prop.ID] = prop
}

// Flush the buffer, this pushes the data to subscriptions
func (p *Proposal) Flush() {
	data := make([]types.Proposal, 0, len(p.buf))
	for _, v := range p.buf {
		data = append(data, v)
	}
	p.buf = make(map[string]types.Proposal, len(p.buf))
	p.mu.Lock()
	for _, ch := range p.chans {
		ch <- data
	}
	p.mu.Unlock()
}

// Subscribe to proposal buffer
func (p *Proposal) Subscribe() (<-chan []types.Proposal, int) {
	p.mu.Lock()
	k := p.getKey()
	ch := make(chan []types.Proposal, p.chBuf)
	p.chans[k] = ch
	p.mu.Unlock()
	return ch, k
}

// Unsubscribe a given subscription
func (p *Proposal) Unsubscribe(k int) {
	p.mu.Lock()
	if ch, ok := p.chans[k]; ok {
		close(ch)
		p.free = append(p.free, k)
	}
	delete(p.chans, k)
	p.mu.Unlock()
}

func (p *Proposal) getKey() int {
	// no need to lock mutex, the caller should have the lock
	if len(p.free) != 0 {
		k := p.free[0]
		// remove first element
		p.free = p.free[1:]
		return k
	}
	// no available keys, the next available one == length of the chans map
	// add 1 to ensure we can't end up with a 0 ID
	return len(p.chans) + 1
}
