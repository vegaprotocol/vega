package gql

import (
	"context"

	"code.vegaprotocol.io/vega/proto/oracles/v1"
)

type propertyKeyResolver VegaResolverRoot

func (p propertyKeyResolver) Type(ctx context.Context, obj *v1.PropertyKey) (PropertyKeyType, error) {
	return convertPropertyKeyTypeFromProto(obj.Type)
}

type conditionResolver VegaResolverRoot

func (c conditionResolver) Operator(ctx context.Context, obj *v1.Condition) (ConditionOperator, error) {
	return convertConditionOperatorFromProto(obj.Operator)
}
