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

func (r *newAssetResolver) Decimals(ctx context.Context, obj *types.NewAsset) (int, error) {
	return int(obj.Changes.Decimals), nil
}

func (r *newAssetResolver) Quantum(ctx context.Context, obj *types.NewAsset) (string, error) {
	return obj.Changes.Quantum, nil
}
