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

	types "code.vegaprotocol.io/protos/vega"
)

type newAssetResolver VegaResolverRoot

func (r *newAssetResolver) Source(ctx context.Context, obj *types.NewAsset) (AssetSource, error) {
	return AssetSourceFromProto(obj.Changes)
}

func (r newAssetResolver) Name(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.Name, nil
}

func (r newAssetResolver) Symbol(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.Symbol, nil
}

func (r newAssetResolver) TotalSupply(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.TotalSupply, nil
}

func (r *newAssetResolver) Decimals(ctx context.Context, obj *types.NewAsset) (int, error) {
	return int(obj.Changes.Decimals), nil
}

func (r *newAssetResolver) MinLpStake(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.Quantum, nil
}
