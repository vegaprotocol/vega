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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type netParamsE interface {
	events.Event
	NetworkParameter() vegapb.NetworkParameter
}

type NetParams struct {
	*subscribers.Base
	ctx context.Context

	mu        sync.RWMutex
	netParams map[string]vegapb.NetworkParameter
	isClosed  bool
	ch        chan vegapb.NetworkParameter
}

func NewNetParams(ctx context.Context) (netParams *NetParams) {
	defer func() { go netParams.consume() }()
	return &NetParams{
		Base:      subscribers.NewBase(ctx, 1000, true),
		ctx:       ctx,
		netParams: map[string]vegapb.NetworkParameter{},
		ch:        make(chan vegapb.NetworkParameter, 100),
	}
}

func (a *NetParams) consume() {
	defer func() {
		// clear the channel before we close it
		// can't use a WaitGroup as suggested because if we get to this point
		// if the Push is still in progress and the queue is full, we'll
		// end up in a situation where we're waiting for the queue to empty, but
		// no consumer.
		for {
			select {
			case _, ok := <-a.ch:
				if !ok {
					close(a.ch)
					return
				}
			}
		}
	}()

	for {
		select {
		case <-a.Closed():
			a.mu.Lock()
			a.isClosed = true
			a.mu.Unlock()
			return
		case netParams, ok := <-a.ch:
			if !ok || a.isClosed {
				// cleanup base
				a.Halt()
				// channel is closed
				return
			}
			a.mu.Lock()
			a.netParams[netParams.Key] = netParams
			a.mu.Unlock()
		}
	}
}

func (a *NetParams) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(netParamsE); ok {
			if a.isClosed {
				return
			}
			a.ch <- ae.NetworkParameter()
		}
	}
}

func (a *NetParams) List(netParamsID string) []*vegapb.NetworkParameter {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(netParamsID) > 0 {
		return a.getNetParam(netParamsID)
	}
	return a.getAllNetParams()
}

func (a *NetParams) getNetParam(param string) []*vegapb.NetworkParameter {
	out := []*vegapb.NetworkParameter{}
	netParam, ok := a.netParams[param]
	if ok {
		out = append(out, &netParam)
	}
	return out
}

func (a *NetParams) getAllNetParams() []*vegapb.NetworkParameter {
	out := make([]*vegapb.NetworkParameter, 0, len(a.netParams))
	for _, v := range a.netParams {
		v := v
		out = append(out, &v)
	}
	return out
}

func (a *NetParams) Types() []events.Type {
	return []events.Type{
		events.NetworkParameterEvent,
	}
}
