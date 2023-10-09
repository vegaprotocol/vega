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

type assetE interface {
	events.Event
	Asset() vegapb.Asset
}

type Assets struct {
	*subscribers.Base
	ctx context.Context

	mu     sync.RWMutex
	assets map[string]vegapb.Asset
	ch     chan vegapb.Asset
}

func NewAssets(ctx context.Context) (assets *Assets) {
	defer func() { go assets.consume() }()
	return &Assets{
		Base:   subscribers.NewBase(ctx, 1000, true),
		ctx:    ctx,
		assets: map[string]vegapb.Asset{},
		ch:     make(chan vegapb.Asset, 100),
	}
}

func (a *Assets) consume() {
	defer func() { close(a.ch) }()
	for {
		select {
		case <-a.Closed():
			return
		case asset, ok := <-a.ch:
			if !ok {
				// cleanup base
				a.Halt()
				// channel is closed
				return
			}
			a.mu.Lock()
			a.assets[asset.Id] = asset
			a.mu.Unlock()
		}
	}
}

func (a *Assets) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(assetE); ok {
			a.ch <- ae.Asset()
		}
	}
}

func (a *Assets) List(assetID string) []*vegapb.Asset {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(assetID) > 0 {
		return a.getAsset(assetID)
	}
	return a.getAllAssets()
}

func (a *Assets) getAsset(assetID string) []*vegapb.Asset {
	out := []*vegapb.Asset{}
	asset, ok := a.assets[assetID]
	if ok {
		out = append(out, &asset)
	}
	return out
}

func (a *Assets) getAllAssets() []*vegapb.Asset {
	out := make([]*vegapb.Asset, 0, len(a.assets))
	for _, v := range a.assets {
		v := v
		out = append(out, &v)
	}
	return out
}

func (a *Assets) Types() []events.Type {
	return []events.Type{
		events.AssetEvent,
	}
}
