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
