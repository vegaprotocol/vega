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
