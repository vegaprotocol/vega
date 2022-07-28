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
