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

package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/subscribers"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"

	"github.com/pkg/errors"
)

var (
	ErrNoAssetForID = errors.New("no asset for ID")
)

type AssetEvent interface {
	events.Event
	Asset() types.Asset
}

type Asset struct {
	*subscribers.Base
	assets map[string]types.Asset
	mu     sync.RWMutex
	ch     chan types.Asset
}

func NewAsset(ctx context.Context) (a *Asset) {
	defer func() { go a.consume() }()
	return &Asset{
		Base:   subscribers.NewBase(ctx, 10, true),
		assets: map[string]types.Asset{},
		ch:     make(chan types.Asset, 100),
	}
}

func (a *Asset) Push(evts ...events.Event) {
	for _, e := range evts {
		if ae, ok := e.(AssetEvent); ok {
			a.ch <- ae.Asset()
		}
	}
}

func (a *Asset) consume() {
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

func (a *Asset) GetByID(id string) (*types.Asset, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a, ok := a.assets[id]; ok {
		return &a, nil
	}
	return nil, ErrNoAssetForID
}

func (a *Asset) GetAll() []types.Asset {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]types.Asset, 0, len(a.assets))
	for _, a := range a.assets {
		out = append(out, a)
	}
	return out
}

func (a *Asset) Types() []events.Type {
	return []events.Type{
		events.AssetEvent,
	}
}
