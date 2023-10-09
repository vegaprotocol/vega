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

type marketE interface {
	events.Event
	Market() vegapb.Market
}

type Markets struct {
	*subscribers.Base
	ctx context.Context

	mu      sync.RWMutex
	markets map[string]vegapb.Market
	ch      chan vegapb.Market
}

func NewMarkets(ctx context.Context) (markets *Markets) {
	defer func() { go markets.consume() }()
	return &Markets{
		Base:    subscribers.NewBase(ctx, 1000, true),
		ctx:     ctx,
		markets: map[string]vegapb.Market{},
		ch:      make(chan vegapb.Market, 100),
	}
}

func (m *Markets) consume() {
	defer func() { close(m.ch) }()
	for {
		select {
		case <-m.Closed():
			return
		case market, ok := <-m.ch:
			if !ok {
				// cleanup base
				m.Halt()
				// channel is closed
				return
			}
			m.mu.Lock()
			m.markets[market.Id] = market
			m.mu.Unlock()
		}
	}
}

func (m *Markets) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(marketE); ok {
			m.ch <- ae.Market()
		}
	}
}

func (m *Markets) List(marketID string) []*vegapb.Market {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(marketID) > 0 {
		return m.getMarket(marketID)
	}
	return m.getAllMarkets()
}

func (m *Markets) getMarket(marketID string) []*vegapb.Market {
	out := []*vegapb.Market{}
	asset, ok := m.markets[marketID]
	if ok {
		out = append(out, &asset)
	}
	return out
}

func (m *Markets) getAllMarkets() []*vegapb.Market {
	out := make([]*vegapb.Market, 0, len(m.markets))
	for _, v := range m.markets {
		v := v
		out = append(out, &v)
	}
	return out
}

func (m *Markets) Types() []events.Type {
	return []events.Type{
		events.MarketCreatedEvent,
		events.MarketUpdatedEvent,
	}
}
