// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
