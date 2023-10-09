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
	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type partyE interface {
	events.Event
	Party() vegapb.Party
}

type Parties struct {
	*subscribers.Base
	ctx context.Context

	mu      sync.RWMutex
	parties map[string]vegapb.Party
	ch      chan vegapb.Party
}

func NewParties(ctx context.Context) (parties *Parties) {
	defer func() {
		parties.parties[types.NetworkParty] = vegapb.Party{Id: types.NetworkParty}
		go parties.consume()
	}()
	return &Parties{
		Base:    subscribers.NewBase(ctx, 1000, true),
		ctx:     ctx,
		parties: map[string]vegapb.Party{},
		ch:      make(chan vegapb.Party, 100),
	}
}

func (a *Parties) consume() {
	defer func() { close(a.ch) }()
	for {
		select {
		case <-a.Closed():
			return
		case party, ok := <-a.ch:
			if !ok {
				// cleanup base
				a.Halt()
				// channel is closed
				return
			}
			a.mu.Lock()
			a.parties[party.Id] = party
			a.mu.Unlock()
		}
	}
}

func (a *Parties) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(partyE); ok {
			a.ch <- ae.Party()
		}
	}
}

func (a *Parties) List() []*vegapb.Party {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*vegapb.Party, 0, len(a.parties))
	for _, v := range a.parties {
		v := v
		out = append(out, &v)
	}
	return out
}

func (a *Parties) Types() []events.Type {
	return []events.Type{
		events.PartyEvent,
	}
}
