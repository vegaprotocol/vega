package buffer

import (
	"sync"

	types "code.vegaprotocol.io/vega/proto"
)

// NodeSig - buffers all proposal changes in a map by ID (so we only preserve the last state)
type NodeSig struct {
	mu    sync.Mutex
	chBuf int
	buf   [][]types.NodeSignature
	chans map[int]chan [][]types.NodeSignature
	free  []int
}

// NewNodeSig get a new proposal buffer
func NewNodeSig() *NodeSig {
	return &NodeSig{
		chBuf: 1,
		buf:   [][]types.NodeSignature{},
		chans: map[int]chan [][]types.NodeSignature{},
		free:  []int{},
	}
}

// Add a single proposal to the buffer
func (n *NodeSig) Add(sigs []types.NodeSignature) {
	n.buf = append(n.buf, sigs)
}

// Flush the buffer, this pushes the data to subscriptions
func (p *NodeSig) Flush() {
	data := make([][]types.NodeSignature, 0, len(p.buf))
	for _, v := range p.buf {
		data = append(data, v)
	}
	p.buf = [][]types.NodeSignature{}
	p.mu.Lock()
	for _, ch := range p.chans {
		ch <- data
	}
	p.mu.Unlock()
}

// Subscribe to proposal buffer
func (n *NodeSig) Subscribe() (<-chan [][]types.NodeSignature, int) {
	n.mu.Lock()
	k := n.getKey()
	ch := make(chan [][]types.NodeSignature, n.chBuf)
	n.chans[k] = ch
	n.mu.Unlock()
	return ch, k
}

// Unsubscribe a given subscription
func (n *NodeSig) Unsubscribe(k int) {
	n.mu.Lock()
	if ch, ok := n.chans[k]; ok {
		close(ch)
		n.free = append(n.free, k)
	}
	delete(n.chans, k)
	n.mu.Unlock()
}

func (n *NodeSig) getKey() int {
	// no need to lock mutex, the caller should have the lock
	if len(n.free) != 0 {
		k := n.free[0]
		// remove first element
		n.free = n.free[1:]
		return k
	}
	// no available keys, the next available one == length of the chans map
	// add 1 to ensure we can't end up with a 0 ID
	return len(n.chans) + 1
}
