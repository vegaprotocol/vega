package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type orderFilterResolver VegaResolverRoot

func (o orderFilterResolver) Status(ctx context.Context, obj *v2.OrderFilter, data []vega.Order_Status) error {
	obj.Statuses = data
	return nil
}

func (o orderFilterResolver) TimeInForce(ctx context.Context, obj *v2.OrderFilter, data []vega.Order_TimeInForce) error {
	obj.TimeInForces = data
	return nil
}
