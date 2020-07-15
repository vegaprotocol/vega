package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/pkg/errors"
)

var (
	ErrNoSignaturesForID = errors.New("no signatures for id")
)

type NodeSignatureEvent interface {
	events.Event
	NodeSignature() types.NodeSignature
}

type Notary struct {
	*subscribers.Base

	sigs map[string][]types.NodeSignature
	mu   sync.RWMutex
	ch   chan types.NodeSignature
}

func NewNotary(ctx context.Context) *Notary {
	n := &Notary{
		Base: subscribers.NewBase(ctx, 10, true),
		sigs: map[string][]types.NodeSignature{},
		ch:   make(chan types.NodeSignature, 100),
	}

	go n.consume()
	return n
}

func (n *Notary) Push(evts ...events.Event) {
	for _, e := range evts {
		if nse, ok := e.(NodeSignatureEvent); ok {
			n.ch <- nse.NodeSignature()
		}
	}
}

func (n *Notary) consume() {
	defer func() { close(n.ch) }()
	for {
		select {
		case <-n.Closed():
			return
		case sig, ok := <-n.ch:
			if !ok {
				// cleanup base
				n.Halt()
				// channel is closed
				return
			}
			n.mu.Lock()
			sigs := n.sigs[sig.ID]
			n.sigs[sig.ID] = append(sigs, sig)
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

func (n *Notary) Types() []events.Type {
	return []events.Type{
		events.NodeSignatureEvent,
	}
}
