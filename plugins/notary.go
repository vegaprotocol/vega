package plugins

import (
	"context"
	"sync"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrNoSignaturesForID = errors.New("no signatures for id")
)

// NodeSigsBuf...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_sigs_buf_mock.go -package mocks code.vegaprotocol.io/vega/plugins PropBuffer
type NodeSigsBuf interface {
	Subscribe() (<-chan [][]types.NodeSignature, int)
	Unsubscribe(int)
}

type NodeSigs struct {
	sigs map[string][]types.NodeSignature
	buf  NodeSigsBuf
	mu   sync.RWMutex

	ch  <-chan [][]types.NodeSignature
	ref int
}

func NewNodeSigs(buf NodeSigsBuf) *NodeSigs {
	return &NodeSigs{
		buf:  buf,
		sigs: map[string][]types.NodeSignature{},
	}
}

// Start - start running the consume loop for the plugin
func (n *NodeSigs) Start(ctx context.Context) {
	n.mu.Lock()
	running := true
	if n.ch == nil {
		n.ch, n.ref = n.buf.Subscribe()
		running = false
	}
	if !running {
		go n.consume(ctx)
	}
	n.mu.Unlock()
}

// Stop - stop running the plugin. Does not set channels to nil to avoid data-race in consume loop
func (n *NodeSigs) Stop() {
	n.mu.Lock()
	if n.ref != 0 {
		n.buf.Unsubscribe(n.ref)
		n.ref = 0
	}
	n.mu.Unlock()
}

func (n *NodeSigs) consume(ctx context.Context) {
	defer func() {
		n.Stop()
		n.ch = nil
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case sigs, ok := <-n.ch:
			if !ok {
				// channel is closed
				return
			}
			// support empty slices for testing
			if len(sigs) == 0 {
				continue
			}
			n.mu.Lock()
			for _, v := range sigs {
				if len(v) > 0 {
					n.sigs[v[0].ID] = v
				}
			}
			n.mu.Unlock()
		}
	}
}

func (n *NodeSigs) GetByID(id string) ([]types.NodeSignature, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if v, ok := n.sigs[id]; ok {
		return v, nil
	}
	return nil, ErrNoSignaturesForID
}
