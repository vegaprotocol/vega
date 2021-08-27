package staking

import (
	"context"
	"sort"
	"sync"

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

	ch chan eventspb.StakeLinking

	mu sync.RWMutex
	// party id -> staking account
	stakingPerParty map[string]*stakingAccount
}

func NewService(ctx context.Context) (svc *Service) {
	defer func() {
		go svc.consume()
	}()

	return &Service{
		Base:            subscribers.NewBase(ctx, 10, true),
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
	defer func() { close(s.ch) }()
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
			partyAccount.links = append(partyAccount.links, evt)
			s.computeCurrentBalance(partyAccount)
			s.mu.Unlock()
		}
	}
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
			// not much to do, just ignore this one.
			continue
		}
		switch link.Type {
		case eventspb.StakeLinking_TYPE_LINK:
			balance = balance.Add(balance, amount)
			continue
		case eventspb.StakeLinking_TYPE_UNLINK:
			if amount.GT(balance) {
				// that's an error, we are missing, events, return now.
				return
			}
			balance = balance.Sub(balance, amount)
		}
	}
	pacc.currentStakeAvailable.Set(balance)
}
