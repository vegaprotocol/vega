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

package services

import (
	"context"
	"sync"

	vegapb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/subscribers"
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
