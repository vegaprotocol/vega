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
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type validatorUpdateE interface {
	events.Event
	ValidatorUpdate() eventspb.ValidatorUpdate
}

type ethereumKeyRotationE interface {
	events.Event
	EthereumKeyRotation() eventspb.EthereumKeyRotation
}
type keyRotationE interface {
	events.Event
	KeyRotation() eventspb.KeyRotation
}

type Validators struct {
	*subscribers.Base
	ctx context.Context

	mu         sync.RWMutex
	validators map[string]eventspb.ValidatorUpdate
	ch         chan events.Event
}

func NewValidators(ctx context.Context) (assets *Validators) {
	defer func() { go assets.consume() }()
	return &Validators{
		Base:       subscribers.NewBase(ctx, 1000, true),
		ctx:        ctx,
		validators: map[string]eventspb.ValidatorUpdate{},
		ch:         make(chan events.Event, 100),
	}
}

func (a *Validators) consume() {
	defer func() { close(a.ch) }()
	for {
		select {
		case <-a.Closed():
			return
		case e, ok := <-a.ch:
			if !ok {
				// cleanup base
				a.Halt()
				// channel is closed
				return
			}
			a.mu.Lock()
			switch te := e.(type) {
			case keyRotationE:
				kr := te.KeyRotation()
				vu, ok := a.validators[kr.NodeId]
				if !ok {
					break
				}

				vu.VegaPubKey = kr.NewPubKey
				a.validators[kr.NodeId] = vu
			case ethereumKeyRotationE:
				kr := te.EthereumKeyRotation()
				vu, ok := a.validators[kr.NodeId]
				if !ok {
					break
				}

				vu.EthereumAddress = kr.NewAddress
				a.validators[kr.NodeId] = vu
			case validatorUpdateE:
				a.validators[te.ValidatorUpdate().NodeId] = te.ValidatorUpdate()
			}
			a.mu.Unlock()
		}
	}
}

func (a *Validators) Push(evts ...events.Event) {
	for _, e := range evts {
		switch te := e.(type) {
		case keyRotationE,
			ethereumKeyRotationE,
			validatorUpdateE:
			a.ch <- te
		}
	}
}

func (a *Validators) List() []*eventspb.ValidatorUpdate {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*eventspb.ValidatorUpdate, 0, len(a.validators))
	for _, v := range a.validators {
		v := v
		out = append(out, &v)
	}
	return out
}

func (a *Validators) Types() []events.Type {
	return []events.Type{
		events.ValidatorUpdateEvent,
		events.EthereumKeyRotationEvent,
		events.KeyRotationEvent,
	}
}
