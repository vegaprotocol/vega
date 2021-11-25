package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
		Version: eventspb.Version,
		Id:      a.eventID(),
		Block:   a.TraceID(),
		ChainId: a.ChainID(),
		Type:    a.et.ToProto(),
		Event: &eventspb.BusEvent_Asset{
			Asset: &a.a,
		},
	}
}

func AssetEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Asset {
	return &Asset{
		Base: newBaseFromStream(ctx, AssetEvent, be),
		a:    *be.GetAsset(),
	}
}
