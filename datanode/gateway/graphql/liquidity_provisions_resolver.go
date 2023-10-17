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
	"strconv"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	types "code.vegaprotocol.io/vega/protos/vega"
)

// LiquidityProvision resolver

type myLiquidityProvisionResolver VegaResolverRoot

func (r *myLiquidityProvisionResolver) Version(_ context.Context, obj *v2.LiquidityProvision) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}

func (r *myLiquidityProvisionResolver) Party(_ context.Context, obj *v2.LiquidityProvision) (*types.Party, error) {
	return &types.Party{Id: obj.PartyId}, nil
}

func (r *myLiquidityProvisionResolver) CreatedAt(ctx context.Context, obj *types.LiquidityProvision) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}

func (r *myLiquidityProvisionResolver) UpdatedAt(ctx context.Context, obj *types.LiquidityProvision) (*string, error) {
	var updatedAt *string
	if obj.UpdatedAt > 0 {
		t := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
		updatedAt = &t
	}
	return updatedAt, nil
}

func (r *myLiquidityProvisionResolver) Market(ctx context.Context, obj *v2.LiquidityProvision) (*types.Market, error) {
	return r.r.getMarketByID(ctx, obj.MarketId)
}

func (r *myLiquidityProvisionResolver) CommitmentAmount(ctx context.Context, obj *types.LiquidityProvision) (string, error) {
	return obj.CommitmentAmount, nil
}
