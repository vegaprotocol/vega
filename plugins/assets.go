package plugins

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

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

func (a *Asset) Push(e events.Event) {
	ae, ok := e.(AssetEvent)
	if !ok {
		return
	}
	a.ch <- ae.Asset()
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
			a.assets[asset.ID] = asset
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
