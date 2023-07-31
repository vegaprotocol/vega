package gql

import (
	"context"

	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type fundingPeriodResolver VegaResolverRoot

func (r *fundingPeriodResolver) Seq(ctx context.Context, obj *v1.FundingPeriod) (int, error) {
	return int(obj.Seq), nil
}

func (r *fundingPeriodResolver) StartTime(ctx context.Context, obj *v1.FundingPeriod) (int64, error) {
	return obj.Start, nil
}

func (r *fundingPeriodResolver) EndTime(ctx context.Context, obj *v1.FundingPeriod) (*int64, error) {
	return obj.End, nil
}

type fundingPeriodDataPointResolver VegaResolverRoot

func (r *fundingPeriodDataPointResolver) Seq(ctx context.Context, obj *v1.FundingPeriodDataPoint) (int, error) {
	return int(obj.Seq), nil
}

func (r *fundingPeriodDataPointResolver) DataPointSource(ctx context.Context, obj *v1.FundingPeriodDataPoint) (*v1.FundingPeriodDataPoint_Source, error) {
	return &obj.DataPointType, nil
}
