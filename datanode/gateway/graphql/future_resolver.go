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

	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type myFutureResolver VegaResolverRoot

func (r *myFutureResolver) SettlementAsset(ctx context.Context, obj *protoTypes.Future) (*protoTypes.Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}

func (r *myFutureResolver) DataSourceSpecForSettlementData(_ context.Context, obj *vegapb.Future) (*DataSourceSpec, error) {
	if obj.DataSourceSpecForSettlementData == nil {
		return nil, nil
	}
	return resolveDataSourceSpec(obj.DataSourceSpecForSettlementData), nil
}

func (r *myFutureResolver) DataSourceSpecForTradingTermination(_ context.Context, obj *vegapb.Future) (*DataSourceSpec, error) {
	if obj.DataSourceSpecForTradingTermination == nil {
		return nil, nil
	}
	return resolveDataSourceSpec(obj.DataSourceSpecForTradingTermination), nil
}
