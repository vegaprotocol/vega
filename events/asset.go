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
	cpy := a.DeepClone()
	return &Asset{
		Base: newBase(ctx, AssetEvent),
		a:    *cpy,
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
		Id:    a.eventID(),
		Block: a.TraceID(),
		Type:  a.et.ToProto(),
		Event: &types.BusEvent_Asset{
			Asset: &a.a,
		},
	}
}
