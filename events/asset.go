package events

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type Asset struct {
	*Base
	a proto.Asset
}

func NewAssetEvent(ctx context.Context, a types.Asset) *Asset {
	return &Asset{
		Base: newBase(ctx, AssetEvent),
		a:    *a.IntoProto(),
	}
}

func (a *Asset) Asset() proto.Asset {
	return a.a
}

func (a Asset) Proto() proto.Asset {
	return a.a
}

func (a Asset) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    a.eventID(),
		Block: a.TraceID(),
		Type:  a.et.ToProto(),
		Event: &eventspb.BusEvent_Asset{
			Asset: &a.a,
		},
	}
}
