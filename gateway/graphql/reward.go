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

	"code.vegaprotocol.io/data-node/vegatime"
	"code.vegaprotocol.io/protos/vega"
)

type rewardResolver VegaResolverRoot

func (r *rewardResolver) Asset(ctx context.Context, obj *vega.Reward) (*vega.Asset, error) {
	asset, err := r.r.getAssetByID(ctx, obj.AssetId)
	if err != nil {
		return nil, err
	}

	return asset, nil
}

func (r *rewardResolver) Party(ctx context.Context, obj *vega.Reward) (*vega.Party, error) {
	return &vega.Party{Id: obj.PartyId}, nil
}

func (r *rewardResolver) ReceivedAt(ctx context.Context, obj *vega.Reward) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.ReceivedAt)), nil
}

func (r *rewardResolver) Epoch(ctx context.Context, obj *vega.Reward) (*vega.Epoch, error) {
	epoch, err := r.r.getEpochByID(ctx, obj.Epoch)
	if err != nil {
		return nil, err
	}

	return epoch, nil
}

func (r *rewardResolver) RewardType(ctx context.Context, obj *vega.Reward) (vega.AccountType, error) {
	accountType, ok := vega.AccountType_value[obj.RewardType]
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("Unknown account type %v", obj.RewardType)
	}

	return vega.AccountType(accountType), nil
}
