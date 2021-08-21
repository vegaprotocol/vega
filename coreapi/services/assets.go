package services

import (
	"context"
	"fmt"
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

	mu     sync.RWMutex
	assets map[string]vegapb.Asset
}

func NewAssets(ctx context.Context) *Assets {
	return &Assets{
		Base:   subscribers.NewBase(ctx, 1000, true),
		assets: map[string]vegapb.Asset{},
	}
}

func (a *Assets) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, e := range evts {
		fmt.Printf("EVENT:%v\n", e)
		switch evt := e.(type) {
		case assetE:
			asset := evt.Asset()
			fmt.Printf("%#v", (asset).String())
			a.assets[evt.Asset().Id] = evt.Asset()
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
	for _, v := range a.assets {
		if v.Id == assetID {
			v := v
			return []*vegapb.Asset{&v}
		}
	}
	return nil
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
