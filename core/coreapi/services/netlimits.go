// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package services

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/subscribers"
	"code.vegaprotocol.io/vega/protos/vega"
	"google.golang.org/protobuf/proto"
)

type netLimitsEvent interface {
	events.Event
	NetworkLimits() *vega.NetworkLimits
}

type NetLimits struct {
	*subscribers.Base
	ctx    context.Context
	limits vega.NetworkLimits
	ch     chan vega.NetworkLimits
	mu     sync.RWMutex
}

func NewNetLimits(ctx context.Context) (netLimits *NetLimits) {
	defer func() { go netLimits.consume() }()
	return &NetLimits{
		Base: subscribers.NewBase(ctx, 1000, true),
		ctx:  ctx,
		ch:   make(chan vega.NetworkLimits, 100),
	}
}

func (n *NetLimits) consume() {
	defer func() { close(n.ch) }()
	for {
		select {
		case <-n.Closed():
			return
		case limits, ok := <-n.ch:
			if !ok {
				n.Halt()
				return
			}
			n.mu.Lock()
			n.limits = limits
			n.mu.Unlock()
		}
	}
}

func (n *NetLimits) Get() *vega.NetworkLimits {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return proto.Clone(&n.limits).(*vega.NetworkLimits)
}

func (n *NetLimits) Push(evts ...events.Event) {
	for _, e := range evts {
		if ne, ok := e.(netLimitsEvent); ok {
			n.ch <- *ne.NetworkLimits()
		}
	}
}

func (n *NetLimits) Types() []events.Type {
	return []events.Type{events.NetworkLimitsEvent}
}
