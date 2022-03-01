package gql

import (
	"context"

	"code.vegaprotocol.io/protos/vega"
)

type rewardSummaryResolver VegaResolverRoot

func (r *rewardSummaryResolver) Asset(ctx context.Context, obj *vega.RewardSummary) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.AssetId)
}

func (r *rewardSummaryResolver) Rewards(ctx context.Context, obj *vega.RewardSummary, skip, first, last *int) ([]*vega.Reward, error) {
	return r.r.allRewards(ctx, obj.PartyId, obj.AssetId, skip, first, last)
}
