package gql

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
)

type rewardSummaryResolver VegaResolverRoot

func (r *rewardSummaryResolver) Asset(ctx context.Context, obj *vega.RewardSummary) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.AssetId)
}

func (r *rewardSummaryResolver) Rewards(ctx context.Context, obj *vega.RewardSummary, skip, first, last *int) ([]*vega.Reward, error) {
	return r.r.allRewards(ctx, obj.PartyId, obj.AssetId, skip, first, last)
}

func (r *rewardSummaryResolver) RewardsConnection(ctx context.Context, summary *vega.RewardSummary, asset *string, pagination *v2.Pagination) (*v2.RewardsConnection, error) {
	var assetID string
	if asset != nil {
		assetID = *asset
	}

	req := v2.GetRewardsRequest{
		PartyId:    summary.PartyId,
		AssetId:    assetID,
		Pagination: pagination,
	}
	resp, err := r.tradingDataClientV2.GetRewards(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve rewards information: %w", err)
	}

	return resp.Rewards, nil
}
