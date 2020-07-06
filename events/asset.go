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
