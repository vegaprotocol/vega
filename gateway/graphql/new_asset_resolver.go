package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type newAssetResolver VegaResolverRoot

func (r *newAssetResolver) Source(ctx context.Context, obj *types.NewAsset) (AssetSource, error) {
	return AssetSourceFromProto(obj.Changes)
}

func (r newAssetResolver) Name(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.Name, nil
}

func (r newAssetResolver) Symbol(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.Symbol, nil
}

func (r newAssetResolver) TotalSupply(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.TotalSupply, nil
}

func (r *newAssetResolver) Decimals(ctx context.Context, obj *types.NewAsset) (int, error) {
	return int(obj.Changes.Decimals), nil
}

func (r *newAssetResolver) MinLpStake(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.MinLpStake, nil
}
