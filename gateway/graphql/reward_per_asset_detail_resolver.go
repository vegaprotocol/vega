package gql

// TODO: This resolver is depricated in favour of RewardSummary; delete once front end has switched over
import (
	"context"
	"math"

	"code.vegaprotocol.io/protos/vega"
)

type rewardPerAssetDetailResolver VegaResolverRoot

func (r *rewardPerAssetDetailResolver) Asset(ctx context.Context, obj *vega.RewardSummary) (*vega.Asset, error) {
	asset, err := r.r.getAssetByID(ctx, obj.AssetId)
	if err != nil {
		return nil, err
	}

	return asset, nil
}

func (r *rewardPerAssetDetailResolver) Rewards(ctx context.Context, obj *vega.RewardSummary) ([]*vega.Reward, error) {
	maxInt := math.MaxInt
	return r.r.allRewards(ctx, obj.PartyId, obj.AssetId, nil, &maxInt, nil)
}

func (r *rewardPerAssetDetailResolver) TotalAmount(ctx context.Context, obj *vega.RewardSummary) (string, error) {
	return obj.Amount, nil
}
