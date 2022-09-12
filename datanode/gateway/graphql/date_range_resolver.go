package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type dateRangeResolver VegaResolverRoot

func (r *dateRangeResolver) Start(ctx context.Context, obj *v2.DateRange, data *int64) error {
	obj.StartTimestamp = data
	return nil
}

func (r *dateRangeResolver) End(ctx context.Context, obj *v2.DateRange, data *int64) error {
	obj.EndTimestamp = data
	return nil
}
