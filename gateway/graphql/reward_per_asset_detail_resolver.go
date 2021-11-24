package gql

import (
	"context"

	"code.vegaprotocol.io/protos/vega"
)

type rewardPerAssetDetailResolver VegaResolverRoot

func (r *rewardPerAssetDetailResolver) Asset(ctx context.Context, obj *vega.RewardPerAssetDetail) (*vega.Asset, error) {
	asset, err := r.r.getAssetByID(ctx, obj.Asset)
	if err != nil {
		return nil, err
	}

	return asset, nil
}

func (r *rewardPerAssetDetailResolver) Rewards(ctx context.Context, obj *vega.RewardPerAssetDetail) ([]*vega.RewardDetails, error) {
	return obj.Details, nil
}

func (r *rewardPerAssetDetailResolver) TotalAmount(ctx context.Context, obj *vega.RewardPerAssetDetail) (string, error) {
	return obj.TotalForAsset, nil
}
