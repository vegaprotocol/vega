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

package staking

import (
	"context"
	"sort"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types/num"
)

type StakeLinkingEvent interface {
	events.Event
	StakeLinking() eventspb.StakeLinking
}

type stakingAccount struct {
	currentStakeAvailable *num.Uint
	links                 []eventspb.StakeLinking
}

type Service struct {
	*subscribers.Base

	log *logging.Logger
	ch  chan eventspb.StakeLinking

	mu sync.RWMutex
	// party id -> staking account
	stakingPerParty map[string]*stakingAccount
}

func NewService(ctx context.Context, log *logging.Logger) (svc *Service) {
	defer func() {
		go svc.consume()
	}()

	return &Service{
		Base:            subscribers.NewBase(ctx, 10, true),
		log:             log,
		ch:              make(chan eventspb.StakeLinking, 100),
		stakingPerParty: map[string]*stakingAccount{},
	}
}

func (s *Service) GetStake(party string) (*num.Uint, []eventspb.StakeLinking) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	partyAccount, ok := s.stakingPerParty[party]
	if !ok {
		return num.Zero(), nil
	}

	return partyAccount.currentStakeAvailable.Clone(), partyAccount.links
}

func (s *Service) Push(evts ...events.Event) {
	for _, e := range evts {
		select {
		case <-s.Closed():
			close(s.ch)
			return
		default:
			if evt, ok := e.(StakeLinkingEvent); ok {
				s.ch <- evt.StakeLinking()
			}
		}
	}
}

func (s *Service) Types() []events.Type {
	return []events.Type{
		events.StakeLinkingEvent,
	}
}

func (s *Service) consume() {
	for {
		select {
		case <-s.Closed():
			return
		case evt, ok := <-s.ch:
			if !ok {
				// cleanup base
				s.Halt()
				// channel is closed
				return
			}
			s.mu.Lock()
			partyAccount, ok := s.stakingPerParty[evt.Party]
			if !ok {
				partyAccount = &stakingAccount{
					currentStakeAvailable: num.Zero(),
					links:                 []eventspb.StakeLinking{},
				}
				s.stakingPerParty[evt.Party] = partyAccount
			}
			s.addLink(partyAccount, evt)
			s.computeCurrentBalance(partyAccount)
			s.mu.Unlock()
		}
	}
}

func (s *Service) addLink(partyAccount *stakingAccount, evt eventspb.StakeLinking) {
	for i, v := range partyAccount.links {
		if v.Id == evt.Id {
			partyAccount.links[i] = evt
			return
		}
	}
	partyAccount.links = append(partyAccount.links, evt)
}

func (s *Service) computeCurrentBalance(pacc *stakingAccount) {
	// just sort so we are sure our stake linking are in order
	sort.SliceStable(pacc.links, func(i, j int) bool {
		return pacc.links[i].Ts < pacc.links[j].Ts
	})
	balance := num.Zero()
	for _, link := range pacc.links {
		if link.Status == eventspb.StakeLinking_STATUS_PENDING || link.Status == eventspb.StakeLinking_STATUS_REJECTED {
			// ignore
			continue
		}
		amount, overflowed := num.UintFromString(link.Amount, 10)
		if overflowed {
			s.log.Error("received non base 10 amount to link", logging.String("amount", link.Amount))
			// not much to do, just ignore this one.
			continue
		}
		switch link.Type {
		case eventspb.StakeLinking_TYPE_LINK:
			balance.Add(balance, amount)
			continue
		case eventspb.StakeLinking_TYPE_UNLINK:
			if amount.GT(balance) {
				// that's an error, we are missing, events, return now.
				return
			}
			balance.Sub(balance, amount)
		}
	}
	pacc.currentStakeAvailable.Set(balance)
}
