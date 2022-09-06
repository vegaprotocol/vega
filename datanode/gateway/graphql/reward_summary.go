// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type rewardSummaryResolver VegaResolverRoot

func (r *rewardSummaryResolver) Asset(ctx context.Context, obj *vega.RewardSummary) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.AssetId)
}

func (r *rewardSummaryResolver) Rewards(ctx context.Context, obj *vega.RewardSummary, skip, first, last *int) ([]*vega.Reward, error) {
	return r.r.allRewards(ctx, obj.PartyId, obj.AssetId, skip, first, last)
}

func (r *rewardSummaryResolver) RewardsConnection(ctx context.Context, summary *vega.RewardSummary, assetID *string, pagination *v2.Pagination) (*v2.RewardsConnection, error) {
	req := v2.ListRewardsRequest{
		PartyId:    summary.PartyId,
		AssetId:    assetID,
		Pagination: pagination,
	}

	header := metadata.MD{}

	resp, err := r.tradingDataClientV2.ListRewards(ctx, &req, grpc.Header(&header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve rewards information: %w", err)
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Rewards, nil
}
