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

	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type myFutureResolver VegaResolverRoot

func (r *myFutureResolver) SettlementAsset(ctx context.Context, obj *protoTypes.Future) (*protoTypes.Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}

func (r *myFutureResolver) DataSourceSpecForSettlementData(ctx context.Context, obj *vegapb.Future) (*DataSourceSpec, error) {
	ds := &DataSourceSpec{
		Data: &DataSourceDefinition{},
	}

	if obj.DataSourceSpecForSettlementData != nil {
		ds = resolveDataSourceSpec(obj.DataSourceSpecForSettlementData)
	}

	return ds, nil
}

func (r *myFutureResolver) DataSourceSpecForTradingTermination(ctx context.Context, obj *vegapb.Future) (*DataSourceSpec, error) {
	ds := &DataSourceSpec{
		Data: &DataSourceDefinition{},
	}

	if obj.DataSourceSpecForTradingTermination != nil {
		ds = resolveDataSourceSpec(obj.DataSourceSpecForTradingTermination)
	}

	return ds, nil
}
