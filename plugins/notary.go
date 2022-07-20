// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package plugins

import (
	"bytes"
	"context"
	"sync"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/pkg/errors"
)

var ErrNoSignaturesForID = errors.New("no signatures for id")

type NodeSignatureEvent interface {
	events.Event
	NodeSignature() commandspb.NodeSignature
}

type Notary struct {
	*subscribers.Base

	sigs map[string][]commandspb.NodeSignature
	mu   sync.RWMutex
	ch   chan commandspb.NodeSignature
}

func NewNotary(ctx context.Context) *Notary {
	n := &Notary{
		Base: subscribers.NewBase(ctx, 10, true),
		sigs: map[string][]commandspb.NodeSignature{},
		ch:   make(chan commandspb.NodeSignature, 100),
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
			n.appendSig(sig)
			n.mu.Unlock()
		}
	}
}

func (n *Notary) appendSig(sig commandspb.NodeSignature) {
	sigs := n.sigs[sig.Id]
	for _, s := range sigs {
		if bytes.Equal(s.Sig, sig.Sig) {
			// we already have this sig
			return
		}
	}
	// let's add the sig to the list
	n.sigs[sig.Id] = append(sigs, sig)
}

func (n *Notary) GetByID(id string) ([]commandspb.NodeSignature, error) {
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
