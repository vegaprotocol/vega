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

package liquidity

import (
	"context"
	"errors"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

var (
	ErrNoMarketOrPartyFilters = errors.New("market or party filters are required")
)

type LiquidityProvisionEvent interface {
	events.Event
	LiquidityProvision() *types.LiquidityProvision
}

type Svc struct {
	*subscribers.Base

	config Config
	log    *logging.Logger

	// marketID -> partyID -> liquidityProvision
	marketsLPs map[string]map[string]*types.LiquidityProvision
	mu         sync.RWMutex
	ch         chan *types.LiquidityProvision
}

func NewService(ctx context.Context, log *logging.Logger, config Config) *Svc {
	log = log.Named(namedLogger)
	svc := &Svc{
		Base:       subscribers.NewBase(ctx, 10, true),
		log:        log,
		config:     config,
		marketsLPs: map[string]map[string]*types.LiquidityProvision{},
		ch:         make(chan *types.LiquidityProvision, 100),
	}

	go svc.consume()
	return svc
}

// ReloadConf update the internal configuration of the order service
func (s *Svc) ReloadConf(config Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != config.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", config.Level.String()),
		)
		s.log.SetLevel(config.Level.Get())
	}

	s.config = config
}

func (s *Svc) Push(evts ...events.Event) {
	for _, e := range evts {
		select {
		case <-s.Closed():
			return
		default:
			if lpe, ok := e.(LiquidityProvisionEvent); ok {
				s.ch <- lpe.LiquidityProvision()
			}
		}
	}
}

func (s *Svc) consume() {
	defer func() { close(s.ch) }()
	for {
		select {
		case <-s.Closed():
			return
		case lp, ok := <-s.ch:
			if !ok {
				// cleanup base
				s.Halt()
				// channel is closed
				return
			}
			s.mu.Lock()
			partiesLPs, ok := s.marketsLPs[lp.MarketId]
			if !ok {
				partiesLPs = map[string]*types.LiquidityProvision{}
				s.marketsLPs[lp.MarketId] = partiesLPs
			}
			partiesLPs[lp.PartyId] = lp
			s.mu.Unlock()
		}
	}
}

func (s *Svc) Get(party, market string) ([]*types.LiquidityProvision, error) {
	if len(party) <= 0 && len(market) <= 0 {
		return nil, ErrNoMarketOrPartyFilters
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(market) > 0 {
		return s.getByMarket(market, party), nil
	}
	return s.getByParty(party), nil
}

func (s *Svc) getByMarket(market string, party string) []*types.LiquidityProvision {
	partiesLPs, ok := s.marketsLPs[market]
	if !ok {
		return nil
	}

	if len(party) > 0 {
		partyLP, ok := partiesLPs[party]
		if !ok {
			return nil
		}
		return []*types.LiquidityProvision{partyLP}
	}

	out := make([]*types.LiquidityProvision, 0, len(partiesLPs))
	for _, v := range partiesLPs {
		out = append(out, v)
	}
	return out
}

func (s *Svc) getByParty(party string) []*types.LiquidityProvision {
	out := []*types.LiquidityProvision{}
	for _, v := range s.marketsLPs {
		if plp, ok := v[party]; ok {
			out = append(out, plp)
		}
	}
	return out
}

func (s *Svc) Types() []events.Type {
	return []events.Type{
		events.LiquidityProvisionEvent,
	}
}
