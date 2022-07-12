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

	"code.vegaprotocol.io/data-node/vegatime"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
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

func (o oracleSpecResolver) Status(_ context.Context, obj *v1.OracleSpec) (OracleSpecStatus, error) {
	return convertOracleSpecStatusFromProto(obj.Status)
}

func (o oracleSpecResolver) Data(ctx context.Context, obj *v1.OracleSpec) ([]*v1.OracleData, error) {
	resp, err := o.tradingDataClient.OracleDataBySpec(ctx, &protoapi.OracleDataBySpecRequest{Id: obj.Id})
	if err != nil {
		return nil, err
	}
	return resp.OracleData, nil
}

func (o oracleSpecResolver) DataConnection(ctx context.Context, spec *v1.OracleSpec, pagination *v2.Pagination) (*v2.OracleDataConnection, error) {
	req := v2.GetOracleDataConnectionRequest{
		SpecId:     spec.Id,
		Pagination: pagination,
	}

	resp, err := o.tradingDataClientV2.GetOracleDataConnection(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.OracleData, nil
}

type propertyKeyResolver VegaResolverRoot

func (p propertyKeyResolver) Type(_ context.Context, obj *v1.PropertyKey) (PropertyKeyType, error) {
	return convertPropertyKeyTypeFromProto(obj.Type)
}

type conditionResolver VegaResolverRoot

func (c conditionResolver) Operator(_ context.Context, obj *v1.Condition) (ConditionOperator, error) {
	return convertConditionOperatorFromProto(obj.Operator)
}

type oracleDataResolver VegaResolverRoot

func (o *oracleDataResolver) BroadcastAt(_ context.Context, obj *v1.OracleData) (string, error) {
	if obj.BroadcastAt == 0 {
		return "", nil
	}
	return vegatime.Format(vegatime.UnixNano(obj.BroadcastAt)), nil
}
