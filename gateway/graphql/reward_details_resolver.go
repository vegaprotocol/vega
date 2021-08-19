package gql

import (
	"context"

	protoapi "code.vegaprotocol.io/protos/vega"
)

type rewardDetailsResolver VegaResolverRoot

func (r *rewardDetailsResolver) Details(ctx context.Context, obj *protoapi.RewardDetails) ([]*RewardPerAssetDetails, error) {
	return nil, nil
}
