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

	types "code.vegaprotocol.io/vega/protos/vega"
)

type updateAssetResolver VegaResolverRoot

func (r *updateAssetResolver) Source(ctx context.Context, obj *types.UpdateAsset) (UpdateAssetSource, error) {
	return UpdateAssetSourceFromProto(obj.Changes)
}

func (r *updateAssetResolver) Quantum(ctx context.Context, obj *types.UpdateAsset) (string, error) {
	return obj.Changes.Quantum, nil
}

func UpdateAssetSourceFromProto(pdetails *types.AssetDetailsUpdate) (UpdateAssetSource, error) {
	if pdetails == nil {
		return nil, ErrNilAssetSource
	}

	switch asimpl := pdetails.Source.(type) {
	case *types.AssetDetailsUpdate_Erc20:
		return UpdateERC20FromProto(asimpl.Erc20), nil
	default:
		return nil, ErrUnimplementedAssetSource
	}
}

func UpdateERC20FromProto(ea *types.ERC20Update) *UpdateErc20 {
	return &UpdateErc20{
		LifetimeLimit:     ea.LifetimeLimit,
		WithdrawThreshold: ea.WithdrawThreshold,
	}
}
