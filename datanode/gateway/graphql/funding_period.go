package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
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

func (r *fundingPeriodResolver) DataPoints(ctx context.Context, obj *v1.FundingPeriod, dateRange *v2.DateRange, source *v1.FundingPeriodDataPoint_Source,
	pagination *v2.Pagination,
) (*v2.FundingPeriodDataPointConnection, error) {
	req := &v2.ListFundingPeriodDataPointsRequest{
		MarketId:   obj.MarketId,
		DateRange:  dateRange,
		Source:     source,
		Pagination: pagination,
	}

	res, err := r.tradingDataClientV2.ListFundingPeriodDataPoints(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.FundingPeriodDataPoints, nil
}

type fundingPeriodDataPointResolver VegaResolverRoot

func (r *fundingPeriodDataPointResolver) Seq(ctx context.Context, obj *v1.FundingPeriodDataPoint) (int, error) {
	return int(obj.Seq), nil
}

func (r *fundingPeriodDataPointResolver) DataPointSource(ctx context.Context, obj *v1.FundingPeriodDataPoint) (*v1.FundingPeriodDataPoint_Source, error) {
	return &obj.DataPointType, nil
}
