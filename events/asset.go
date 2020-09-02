package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Asset struct {
	*Base
	a types.Asset
}

func NewAssetEvent(ctx context.Context, a types.Asset) *Asset {
	return &Asset{
		Base: newBase(ctx, AssetEvent),
		a:    a,
	}
}

func (a *Asset) Asset() types.Asset {
	return a.a
}

func (a Asset) Proto() types.Asset {
	return a.a
}

func (a Asset) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:   a.traceID,
		Type: a.et.ToProto(),
		Event: &types.BusEvent_Asset{
			Asset: &a.a,
		},
	}
}
