package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type myNewAssetResolver VegaResolverRoot

func (r *myNewAssetResolver) Source(ctx context.Context, obj *types.NewAsset) (AssetSource, error) {
	return AssetSourceFromProto(obj.Changes)
}
