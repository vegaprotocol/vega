package gql

import (
	"context"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
)

type rewardDetailsResolver VegaResolverRoot

func (r *rewardDetailsResolver) Asset(ctx context.Context, obj *protoapi.GetRewardDetailsResponse) (string, error) {
	return obj.AssetId, nil
}

func (r *rewardDetailsResolver) LastReward(ctx context.Context, obj *protoapi.GetRewardDetailsResponse) (string, error) {
	return obj.LastReward, nil
}

func (r *rewardDetailsResolver) LastRewardPercentage(ctx context.Context, obj *protoapi.GetRewardDetailsResponse) (string, error) {
	return obj.LastRewardPercentage, nil
}

func (r *rewardDetailsResolver) TotalReward(ctx context.Context, obj *protoapi.GetRewardDetailsResponse) (string, error) {
	return obj.TotalReward, nil
}
