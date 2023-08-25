package sqlstore

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
)

type FundingPeriods struct {
	*ConnectionSource
}

var fundingPeriodOrdering = TableOrdering{
	ColumnOrdering{Name: "start_time", Sorting: ASC},
	ColumnOrdering{Name: "market_id", Sorting: ASC},
	ColumnOrdering{Name: "funding_period_seq", Sorting: ASC},
}

var fundingPeriodDataPointOrdering = TableOrdering{
	ColumnOrdering{Name: "timestamp", Sorting: ASC},
	ColumnOrdering{Name: "market_id", Sorting: ASC},
	ColumnOrdering{Name: "funding_period_seq", Sorting: ASC},
	ColumnOrdering{Name: "data_point_type", Sorting: ASC},
}

func NewFundingPeriods(connectionSource *ConnectionSource) *FundingPeriods {
	return &FundingPeriods{
		ConnectionSource: connectionSource,
	}
}

func (fp *FundingPeriods) AddFundingPeriod(ctx context.Context, period *entities.FundingPeriod) error {
	defer metrics.StartSQLQuery("FundingPeriods", "AddFundingPeriod")()
	_, err := fp.Connection.Exec(ctx,
		`insert into funding_period(market_id, funding_period_seq, start_time, end_time, funding_payment, funding_rate,
                           external_twap, internal_twap, vega_time, tx_hash)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) on conflict (market_id, funding_period_seq) do update
	set end_time = EXCLUDED.end_time,
	    funding_payment = EXCLUDED.funding_payment,
	    funding_rate = EXCLUDED.funding_rate,
	    external_twap = EXCLUDED.external_twap,
	    internal_twap = EXCLUDED.internal_twap,
	    vega_time = EXCLUDED.vega_time,
	    tx_hash = EXCLUDED.tx_hash`,
		period.MarketID, period.FundingPeriodSeq, period.StartTime, period.EndTime, period.FundingPayment, period.FundingRate,
		period.ExternalTwap, period.InternalTwap, period.VegaTime, period.TxHash)

	return err
}

func (fp *FundingPeriods) AddDataPoint(ctx context.Context, dataPoint *entities.FundingPeriodDataPoint) error {
	defer metrics.StartSQLQuery("FundingPeriodDataPoint", "AddDataPoint")()

	_, err := fp.Connection.Exec(ctx,
		`insert into funding_period_data_points(market_id, funding_period_seq, data_point_type, price, timestamp, twap, vega_time, tx_hash)
values ($1, $2, $3, $4, $5, $6, $7, $8) on conflict (market_id, funding_period_seq, data_point_type, vega_time) do update
	set price = EXCLUDED.price, twap = EXCLUDED.twap, timestamp = EXCLUDED.timestamp, tx_hash = EXCLUDED.tx_hash`,
		dataPoint.MarketID, dataPoint.FundingPeriodSeq, dataPoint.DataPointType, dataPoint.Price, dataPoint.Timestamp, dataPoint.Twap, dataPoint.VegaTime, dataPoint.TxHash)

	return err
}

func (fp *FundingPeriods) ListFundingPeriods(ctx context.Context, marketID entities.MarketID, dateRange entities.DateRange, pagination entities.CursorPagination) (
	[]entities.FundingPeriod, entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("FundingPeriods", "ListFundingPeriods")()
	var periods []entities.FundingPeriod
	var pageInfo entities.PageInfo
	var args []interface{}
	var err error

	query := fmt.Sprintf(`select * from funding_period where market_id = %s`, nextBindVar(&args, marketID))
	if dateRange.Start != nil {
		query = fmt.Sprintf("%s and start_time >= %s", query, nextBindVar(&args, dateRange.Start))
	}

	if dateRange.End != nil {
		query = fmt.Sprintf("%s and start_time < %s", query, nextBindVar(&args, dateRange.End))
	}

	query, args, err = PaginateQuery[entities.FundingPeriodCursor](query, args, fundingPeriodOrdering, pagination)
	if err != nil {
		return periods, pageInfo, err
	}

	err = pgxscan.Select(ctx, fp.Connection, &periods, query, args...)
	if err != nil {
		return periods, pageInfo, err
	}

	periods, pageInfo = entities.PageEntities[*v2.FundingPeriodEdge](periods, pagination)

	return periods, pageInfo, nil
}

func (fp *FundingPeriods) ListFundingPeriodDataPoints(ctx context.Context, marketID entities.MarketID, dateRange entities.DateRange,
	source *entities.FundingPeriodDataPointSource, pagination entities.CursorPagination,
) ([]entities.FundingPeriodDataPoint, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("FundingPeriodDataPoint", "ListFundingPeriodDataPoints")()
	var dataPoints []entities.FundingPeriodDataPoint
	var pageInfo entities.PageInfo
	var args []interface{}
	var err error

	query := fmt.Sprintf("select * from funding_period_data_points where market_id = %s", nextBindVar(&args, marketID))

	if dateRange.Start != nil {
		query = fmt.Sprintf("%s and timestamp >= %s", query, nextBindVar(&args, dateRange.Start))
	}
	if dateRange.End != nil {
		query = fmt.Sprintf("%s and timestamp < %s", query, nextBindVar(&args, dateRange.End))
	}

	if source != nil {
		query = fmt.Sprintf("%s and data_point_type = %s", query, nextBindVar(&args, *source))
	}

	query, args, err = PaginateQuery[entities.FundingPeriodDataPointCursor](query, args, fundingPeriodDataPointOrdering, pagination)
	if err != nil {
		return dataPoints, pageInfo, err
	}

	err = pgxscan.Select(ctx, fp.Connection, &dataPoints, query, args...)
	if err != nil {
		return dataPoints, pageInfo, err
	}

	dataPoints, pageInfo = entities.PageEntities[*v2.FundingPeriodDataPointEdge](dataPoints, pagination)

	return dataPoints, pageInfo, nil
}
