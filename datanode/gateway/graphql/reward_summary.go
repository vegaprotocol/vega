// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package gql

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type rewardSummaryResolver VegaResolverRoot

func (r *rewardSummaryResolver) Asset(ctx context.Context, obj *vega.RewardSummary) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.AssetId)
}

func (r *rewardSummaryResolver) RewardsConnection(ctx context.Context, summary *vega.RewardSummary, assetID *string, pagination *v2.Pagination, includeDerivedParties *bool) (*v2.RewardsConnection, error) {
	req := v2.ListRewardsRequest{
		PartyIds:              []string{summary.PartyId},
		AssetId:               assetID,
		Pagination:            pagination,
		IncludeDerivedParties: includeDerivedParties,
	}
	resp, err := r.tradingDataClientV2.ListRewards(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve rewards information: %w", err)
	}

	return resp.Rewards, nil
}
