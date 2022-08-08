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

	"code.vegaprotocol.io/vega/datanode/vegatime"
	protoapi "code.vegaprotocol.io/vega/protos/data-node/api/v1"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	v1 "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
)

type oracleSpecResolver VegaResolverRoot

func (o *oracleSpecResolver) CreatedAt(_ context.Context, obj *v1.OracleSpec) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}

func (o *oracleSpecResolver) UpdatedAt(_ context.Context, obj *v1.OracleSpec) (*string, error) {
	if obj.UpdatedAt <= 0 {
		return nil, nil
	}
	formattedTime := vegatime.Format(vegatime.UnixNano(obj.UpdatedAt))
	return &formattedTime, nil
}

func (o oracleSpecResolver) Data(ctx context.Context, obj *v1.OracleSpec) ([]*v1.OracleData, error) {
	resp, err := o.tradingDataClient.OracleDataBySpec(ctx, &protoapi.OracleDataBySpecRequest{Id: obj.Id})
	if err != nil {
		return nil, err
	}
	return resp.OracleData, nil
}

func (o oracleSpecResolver) DataConnection(ctx context.Context, spec *v1.OracleSpec, pagination *v2.Pagination) (*v2.OracleDataConnection, error) {
	var specID *string
	if spec != nil && spec.Id != "" {
		specID = &spec.Id
	}
	req := v2.ListOracleDataRequest{
		OracleSpecId: specID,
		Pagination:   pagination,
	}

	resp, err := o.tradingDataClientV2.ListOracleData(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.OracleData, nil
}

type propertyKeyResolver VegaResolverRoot

type conditionResolver VegaResolverRoot

type oracleDataResolver VegaResolverRoot

func (o *oracleDataResolver) BroadcastAt(_ context.Context, obj *v1.OracleData) (string, error) {
	if obj.BroadcastAt == 0 {
		return "", nil
	}
	return vegatime.Format(vegatime.UnixNano(obj.BroadcastAt)), nil
}
