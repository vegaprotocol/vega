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
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "market_id", Sorting: ASC},
	ColumnOrdering{Name: "funding_period_seq", Sorting: ASC},
}

var fundingPeriodDataPointOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
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
		`insert into funding_period(market_id, funding_period_seq, start_time, end_time, funding_payment, funding_rate, vega_time, tx_hash)
values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		period.MarketID, period.FundingPeriodSeq, period.StartTime, period.EndTime, period.FundingPayment, period.FundingRate, period.VegaTime, period.TxHash)

	return err
}

func (fp *FundingPeriods) AddDataPoint(ctx context.Context, dataPoint *entities.FundingPeriodDataPoint) error {
	defer metrics.StartSQLQuery("FundingPeriodDataPoint", "AddDataPoint")()
	_, err := fp.Connection.Exec(ctx,
		`insert into funding_period_data_points(market_id, funding_period_seq, data_point_type, price, timestamp, vega_time, tx_hash)
values ($1, $2, $3, $4, $5, $6, $7) on conflict (market_id, funding_period_seq, data_point_type, vega_time) do update
	set price = $4, timestamp = $5, tx_hash = $7`,
		dataPoint.MarketID, dataPoint.FundingPeriodSeq, dataPoint.DataPointType, dataPoint.Price, dataPoint.Timestamp, dataPoint.VegaTime, dataPoint.TxHash)

	return err
}

func (fp *FundingPeriods) ListFundingPeriods(ctx context.Context, marketID entities.MarketID, seq *uint64, pagination entities.CursorPagination) (
	[]entities.FundingPeriod, entities.PageInfo, error,
) {
	defer metrics.StartSQLQuery("FundingPeriods", "ListFundingPeriods")()
	var periods []entities.FundingPeriod
	var pageInfo entities.PageInfo
	var args []interface{}
	var err error

	query := fmt.Sprintf(`select * from funding_period where market_id = %s`, nextBindVar(&args, marketID))
	if seq != nil {
		query = fmt.Sprintf("%s and funding_period_seq = %s", query, nextBindVar(&args, *seq))
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

func (fp *FundingPeriods) ListFundingPeriodDataPoints(ctx context.Context, marketID entities.MarketID, seqs []uint64,
	source *entities.FundingPeriodDataPointSource, pagination entities.CursorPagination,
) ([]entities.FundingPeriodDataPoint, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("FundingPeriodDataPoint", "ListFundingPeriodDataPoints")()
	var dataPoints []entities.FundingPeriodDataPoint
	var pageInfo entities.PageInfo
	var args []interface{}
	var err error

	query := fmt.Sprintf("select * from funding_period_data_points where market_id = %s", nextBindVar(&args, marketID))
	if len(seqs) > 0 {
		query = fmt.Sprintf("%s and funding_period_seq = ANY(%s)", query, nextBindVar(&args, seqs))
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
