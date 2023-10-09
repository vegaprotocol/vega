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

	"code.vegaprotocol.io/vega/protos/vega"
)

//type LiquidityProvisionUpdateResolver interface {
//	CreatedAt(ctx context.Context, obj *vega.LiquidityProvision) (string, error)
//	UpdatedAt(ctx context.Context, obj *vega.LiquidityProvision) (*string, error)
//
//	Version(ctx context.Context, obj *vega.LiquidityProvision) (string, error)
//}

type liquidityProvisionUpdateResolver VegaResolverRoot

func (r *liquidityProvisionUpdateResolver) CreatedAt(ctx context.Context, obj *vega.LiquidityProvision) (string, error) {
	return strconv.FormatInt(obj.CreatedAt, 10), nil
}

func (r *liquidityProvisionUpdateResolver) UpdatedAt(ctx context.Context, obj *vega.LiquidityProvision) (*string, error) {
	if obj.UpdatedAt == 0 {
		return nil, nil
	}

	updatedAt := strconv.FormatInt(obj.UpdatedAt, 10)

	return &updatedAt, nil
}

func (r *liquidityProvisionUpdateResolver) Version(ctx context.Context, obj *vega.LiquidityProvision) (string, error) {
	return strconv.FormatUint(obj.Version, 10), nil
}
