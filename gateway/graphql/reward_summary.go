// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
