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

type marketDataE interface {
	events.Event
	MarketData() vegapb.MarketData
}

type MarketsData struct {
	*subscribers.Base
	ctx context.Context

	mu          sync.RWMutex
	marketsData map[string]vegapb.MarketData
	ch          chan vegapb.MarketData
}

func NewMarketsData(ctx context.Context) (marketsData *MarketsData) {
	defer func() { go marketsData.consume() }()
	return &MarketsData{
		Base:        subscribers.NewBase(ctx, 1000, true),
		ctx:         ctx,
		marketsData: map[string]vegapb.MarketData{},
		ch:          make(chan vegapb.MarketData, 100),
	}
}

func (m *MarketsData) consume() {
	defer func() { close(m.ch) }()
	for {
		select {
		case <-m.Closed():
			return
		case marketData, ok := <-m.ch:
			if !ok {
				// cleanup base
				m.Halt()
				// channel is closed
				return
			}
			m.mu.Lock()
			m.marketsData[marketData.Market] = marketData
			m.mu.Unlock()
		}
	}
}

func (m *MarketsData) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(marketDataE); ok {
			m.ch <- ae.MarketData()
		}
	}
}

func (m *MarketsData) List(marketID string) []*vegapb.MarketData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(marketID) > 0 {
		return m.getMarketData(marketID)
	}
	return m.getAllMarketsData()
}

func (m *MarketsData) getMarketData(marketID string) []*vegapb.MarketData {
	out := []*vegapb.MarketData{}
	asset, ok := m.marketsData[marketID]
	if ok {
		out = append(out, &asset)
	}
	return out
}

func (m *MarketsData) getAllMarketsData() []*vegapb.MarketData {
	out := make([]*vegapb.MarketData, 0, len(m.marketsData))
	for _, v := range m.marketsData {
		v := v
		out = append(out, &v)
	}
	return out
}

func (m *MarketsData) Types() []events.Type {
	return []events.Type{
		events.MarketDataEvent,
	}
}
