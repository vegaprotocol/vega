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
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/subscribers"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	coreapipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type StakeLinkingEvent interface {
	events.Event
	StakeLinking() eventspb.StakeLinking
}

type stakingAccount struct {
	currentStakeAvailable *num.Uint
	links                 []eventspb.StakeLinking
}

type PartiesStake struct {
	*subscribers.Base

	log *logging.Logger
	ch  chan eventspb.StakeLinking

	mu sync.RWMutex
	// party id -> staking account
	stakingPerParty map[string]*stakingAccount
}

func NewPartiesStake(ctx context.Context, log *logging.Logger) (svc *PartiesStake) {
	defer func() {
		go svc.consume()
	}()

	return &PartiesStake{
		Base:            subscribers.NewBase(ctx, 10, true),
		log:             log,
		ch:              make(chan eventspb.StakeLinking, 100),
		stakingPerParty: map[string]*stakingAccount{},
	}
}

func (p *PartiesStake) List(party string) []*coreapipb.PartyStake {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(party) > 0 {
		return p.getParty(party)
	}
	return p.getAll()
}

func (p *PartiesStake) getParty(party string) []*coreapipb.PartyStake {
	partyAccount, ok := p.stakingPerParty[party]
	if !ok {
		return nil
	}

	return []*coreapipb.PartyStake{
		{
			Party:                 party,
			CurrentStakeAvailable: partyAccount.currentStakeAvailable.String(),
			StakeLinkings:         Links(partyAccount.links).IntoPointers(),
		},
	}
}

func (p *PartiesStake) getAll() []*coreapipb.PartyStake {
	out := make([]*coreapipb.PartyStake, 0, len(p.stakingPerParty))

	for k, v := range p.stakingPerParty {
		out = append(out, &coreapipb.PartyStake{
			Party:                 k,
			CurrentStakeAvailable: v.currentStakeAvailable.String(),
			StakeLinkings:         Links(v.links).IntoPointers(),
		})
	}

	return out
}

func (p *PartiesStake) Push(evts ...events.Event) {
	for _, e := range evts {
		select {
		case <-p.Closed():
			close(p.ch)
			return
		default:
			if evt, ok := e.(StakeLinkingEvent); ok {
				p.ch <- evt.StakeLinking()
			}
		}
	}
}

func (p *PartiesStake) Types() []events.Type {
	return []events.Type{
		events.StakeLinkingEvent,
	}
}

func (p *PartiesStake) consume() {
	for {
		select {
		case <-p.Closed():
			return
		case evt, ok := <-p.ch:
			if !ok {
				// cleanup base
				p.Halt()
				// channel is closed
				return
			}
			p.mu.Lock()
			partyAccount, ok := p.stakingPerParty[evt.Party]
			if !ok {
				partyAccount = &stakingAccount{
					currentStakeAvailable: num.UintZero(),
					links:                 []eventspb.StakeLinking{},
				}
				p.stakingPerParty[evt.Party] = partyAccount
			}
			p.addLink(partyAccount, evt)
			p.computeCurrentBalance(partyAccount)
			p.mu.Unlock()
		}
	}
}

func (p *PartiesStake) addLink(partyAccount *stakingAccount, evt eventspb.StakeLinking) {
	for i, v := range partyAccount.links {
		if v.Id == evt.Id {
			partyAccount.links[i] = evt
			return
		}
	}
	partyAccount.links = append(partyAccount.links, evt)
}

func (p *PartiesStake) computeCurrentBalance(pacc *stakingAccount) {
	// just sort so we are sure our stake linking are in order
	sort.SliceStable(pacc.links, func(i, j int) bool {
		return pacc.links[i].Ts < pacc.links[j].Ts
	})
	balance := num.UintZero()
	for _, link := range pacc.links {
		if link.Status == eventspb.StakeLinking_STATUS_PENDING || link.Status == eventspb.StakeLinking_STATUS_REJECTED {
			// ignore
			continue
		}
		amount, overflowed := num.UintFromString(link.Amount, 10)
		if overflowed {
			p.log.Error("received non base 10 amount to link", logging.String("amount", link.Amount))
			// not much to do, just ignore this one.
			continue
		}
		switch link.Type {
		case eventspb.StakeLinking_TYPE_LINK:
			balance.Add(balance, amount)
			continue
		case eventspb.StakeLinking_TYPE_UNLINK:
			if amount.GT(balance) {
				p.log.Error("could not apply unlink",
					logging.String("amount", link.Amount),
					logging.String("party", link.Party),
				)
				// that's an error, we are missing, events, return now.
				return
			}
			balance.Sub(balance, amount)
		}
	}
	pacc.currentStakeAvailable.Set(balance)
}

type Links []eventspb.StakeLinking

func (l Links) IntoPointers() []*eventspb.StakeLinking {
	out := make([]*eventspb.StakeLinking, 0, len(l))
	for _, v := range l {
		v := v
		out = append(out, &v)
	}
	return out
}
