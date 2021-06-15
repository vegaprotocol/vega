package events

import (
	"context"

	"code.vegaprotocol.io/vega/types"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
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

func (a Asset) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    a.eventID(),
		Block: a.TraceID(),
		Type:  a.et.ToProto(),
		Event: &eventspb.BusEvent_Asset{
			Asset: a.a.IntoProto(),
		},
	}
}
