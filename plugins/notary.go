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

type Notary struct {
	sigs map[string][]types.NodeSignature
	buf  NotaryBuf
	mu   sync.RWMutex

	ch  <-chan [][]types.NodeSignature
	ref int
}

func NewNotary(buf NotaryBuf) *Notary {
	return &Notary{
		buf:  buf,
		sigs: map[string][]types.NodeSignature{},
	}
}

// Start - start running the consume loop for the plugin
func (n *Notary) Start(ctx context.Context) {
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
func (n *Notary) Stop() {
	n.mu.Lock()
	if n.ref != 0 {
		n.buf.Unsubscribe(n.ref)
		n.ref = 0
	}
	n.mu.Unlock()
}

func (n *Notary) consume(ctx context.Context) {
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

func (n *Notary) GetByID(id string) ([]types.NodeSignature, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if v, ok := n.sigs[id]; ok {
		return v, nil
	}
	return nil, ErrNoSignaturesForID
}
