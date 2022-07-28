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

package services

import (
	"context"
	"sync"

	vegapb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/subscribers"
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
	defer func() { close(a.ch) }()
	for {
		select {
		case <-a.Closed():
			return
		case netParams, ok := <-a.ch:
			if !ok {
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
