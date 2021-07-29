package gql

import (
	"context"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/data-node/vegatime"
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

type propertyKeyResolver VegaResolverRoot

func (p propertyKeyResolver) Type(_ context.Context, obj *v1.PropertyKey) (PropertyKeyType, error) {
	return convertPropertyKeyTypeFromProto(obj.Type)
}

type conditionResolver VegaResolverRoot

func (c conditionResolver) Operator(_ context.Context, obj *v1.Condition) (ConditionOperator, error) {
	return convertConditionOperatorFromProto(obj.Operator)
}
