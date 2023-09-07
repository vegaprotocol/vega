package sqlstore_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/require"
)

type fundingPeriodTestStores struct {
	bs *sqlstore.Blocks
	ms *sqlstore.Markets
	fp *sqlstore.FundingPeriods

	blocks     []entities.Block
	markets    []entities.Market
	periods    []entities.FundingPeriod
	dataPoints []entities.FundingPeriodDataPoint
}

func setupFundingPeriodTests(ctx context.Context, t *testing.T) *fundingPeriodTestStores {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ms := sqlstore.NewMarkets(connectionSource)
	fp := sqlstore.NewFundingPeriods(connectionSource)

	return newFundingPeriodTestStores(bs, ms, fp).Initialize(ctx, t)
}

func newFundingPeriodTestStores(bs *sqlstore.Blocks, ms *sqlstore.Markets, fp *sqlstore.FundingPeriods) *fundingPeriodTestStores {
	return &fundingPeriodTestStores{
		bs: bs,
		ms: ms,
		fp: fp,
	}
}

func (s *fundingPeriodTestStores) Initialize(ctx context.Context, t *testing.T) *fundingPeriodTestStores {
	t.Helper()
	s.blocks = make([]entities.Block, 0, 10)
	s.markets = make([]entities.Market, 0, 3)

	for i := 0; i < 10; i++ {
		block := addTestBlock(t, ctx, s.bs)
		s.blocks = append(s.blocks, block)
		if i < 3 {
			s.markets = append(s.markets, helpers.AddTestMarket(t, ctx, s.ms, block))
		}
	}

	return s
}

func TestFundingPeriod_AddFundingPeriod(t *testing.T) {
	t.Run("should add funding period if the market exists and the sequence number does not exist", testAddFundingPeriodShouldSucceedIfMarketExistsAndSequenceDoesNotExist)
	t.Run("should update funding period if the market exists and the sequence number already exists", testAddFundingPeriodShouldUpdateIfMarketExistsAndSequenceExists)
}

func testAddFundingPeriodShouldSucceedIfMarketExistsAndSequenceDoesNotExist(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupFundingPeriodTests(ctx, t)

	period := entities.FundingPeriod{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 1,
		StartTime:        stores.blocks[3].VegaTime,
		EndTime:          nil,
		FundingPayment:   nil,
		FundingRate:      nil,
		VegaTime:         stores.blocks[3].VegaTime,
		TxHash:           generateTxHash(),
	}

	err := stores.fp.AddFundingPeriod(ctx, &period)
	require.NoError(t, err)
}

func testAddFundingPeriodShouldUpdateIfMarketExistsAndSequenceExists(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupFundingPeriodTests(ctx, t)

	period := entities.FundingPeriod{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 1,
		StartTime:        stores.blocks[3].VegaTime,
		EndTime:          nil,
		FundingPayment:   nil,
		FundingRate:      nil,
		ExternalTwap:     nil,
		InternalTwap:     nil,
		VegaTime:         stores.blocks[3].VegaTime,
		TxHash:           generateTxHash(),
	}

	err := stores.fp.AddFundingPeriod(ctx, &period)
	require.NoError(t, err)

	var dbResult entities.FundingPeriod
	err = pgxscan.Get(ctx, stores.fp.Connection, &dbResult, `select * from funding_period where market_id = $1 and funding_period_seq = $2`, stores.markets[0].ID, 1)
	require.NoError(t, err)
	assert.Equal(t, period, dbResult)

	period.EndTime = &stores.blocks[9].VegaTime
	period.FundingPayment = ptr.From(num.DecimalFromFloat(1.0))
	period.FundingRate = ptr.From(num.DecimalFromFloat(1.0))
	period.ExternalTwap = ptr.From(num.DecimalFromFloat(1.0))
	period.InternalTwap = ptr.From(num.DecimalFromFloat(1.1))
	period.VegaTime = stores.blocks[9].VegaTime
	period.TxHash = generateTxHash()

	err = stores.fp.AddFundingPeriod(ctx, &period)
	require.NoError(t, err)

	err = pgxscan.Get(ctx, stores.fp.Connection, &dbResult, `select * from funding_period where market_id = $1 and funding_period_seq = $2`, stores.markets[0].ID, 1)
	require.NoError(t, err)
	assert.Equal(t, period, dbResult)
}

func TestFundingPeriod_AddFundingPeriodDataPoint(t *testing.T) {
	t.Run("should add data points for existing funding periods", testAddForExistingFundingPeriods)
	t.Run("should not error if the funding period does not exist", testShouldNotErrorIfNoFundingPeriod)
	t.Run("should update the data point if multiple data points for the same source is received in the same block", testShouldUpdateDataPointInSameBlock)
}

func testAddForExistingFundingPeriods(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupFundingPeriodTests(ctx, t)

	period := entities.FundingPeriod{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 1,
		StartTime:        stores.blocks[3].VegaTime,
		EndTime:          nil,
		FundingPayment:   nil,
		FundingRate:      nil,
		VegaTime:         stores.blocks[3].VegaTime,
		TxHash:           generateTxHash(),
	}

	err := stores.fp.AddFundingPeriod(ctx, &period)
	require.NoError(t, err)

	dataPoint := entities.FundingPeriodDataPoint{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 1,
		DataPointType:    entities.FundingPeriodDataPointSourceExternal,
		Price:            num.DecimalFromFloat(1.0),
		Timestamp:        stores.blocks[4].VegaTime,
		VegaTime:         stores.blocks[4].VegaTime,
		TxHash:           generateTxHash(),
	}

	err = stores.fp.AddDataPoint(ctx, &dataPoint)
	require.NoError(t, err)
}

func testShouldNotErrorIfNoFundingPeriod(t *testing.T) {
	// Note: this test was changed from should error to should not error as we can not rely on the
	// foreign key constraint to the funding_period table which has been dropped due to the
	// funding_period_data_point table being migrated to a TimescaleDB hypertable.
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupFundingPeriodTests(ctx, t)

	dataPoint := entities.FundingPeriodDataPoint{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 2,
		DataPointType:    entities.FundingPeriodDataPointSourceExternal,
		Price:            num.DecimalFromFloat(100.0),
		Timestamp:        stores.blocks[4].VegaTime,
		VegaTime:         stores.blocks[4].VegaTime,
		TxHash:           generateTxHash(),
	}

	err := stores.fp.AddDataPoint(ctx, &dataPoint)
	require.NoError(t, err)
}

func testShouldUpdateDataPointInSameBlock(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	stores := setupFundingPeriodTests(ctx, t)

	period := entities.FundingPeriod{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 1,
		StartTime:        stores.blocks[3].VegaTime,
		EndTime:          nil,
		FundingPayment:   nil,
		FundingRate:      nil,
		ExternalTwap:     nil,
		InternalTwap:     nil,
		VegaTime:         stores.blocks[3].VegaTime,
		TxHash:           generateTxHash(),
	}

	err := stores.fp.AddFundingPeriod(ctx, &period)
	require.NoError(t, err)

	dp1 := entities.FundingPeriodDataPoint{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 1,
		DataPointType:    entities.FundingPeriodDataPointSourceExternal,
		Price:            num.DecimalFromFloat(1.0),
		Twap:             num.DecimalFromFloat(1.0),
		Timestamp:        stores.blocks[4].VegaTime,
		VegaTime:         stores.blocks[4].VegaTime,
		TxHash:           generateTxHash(),
	}

	err = stores.fp.AddDataPoint(ctx, &dp1)
	require.NoError(t, err)

	var inserted []entities.FundingPeriodDataPoint
	err = pgxscan.Select(ctx, connectionSource.Connection, &inserted,
		`SELECT * FROM funding_period_data_points where market_id = $1 and funding_period_seq = $2 and data_point_type = $3 and vega_time = $4`,
		stores.markets[0].ID, 1, entities.FundingPeriodDataPointSourceExternal, stores.blocks[4].VegaTime)
	require.NoError(t, err)
	assert.Len(t, inserted, 1)
	assert.Equal(t, dp1, inserted[0])

	dp2 := entities.FundingPeriodDataPoint{
		MarketID:         stores.markets[0].ID,
		FundingPeriodSeq: 1,
		DataPointType:    entities.FundingPeriodDataPointSourceExternal,
		Price:            num.DecimalFromFloat(2.0),
		Twap:             num.DecimalFromFloat(2.0),
		Timestamp:        stores.blocks[4].VegaTime.Add(100 * time.Microsecond),
		VegaTime:         stores.blocks[4].VegaTime,
		TxHash:           generateTxHash(),
	}

	err = stores.fp.AddDataPoint(ctx, &dp2)
	require.NoError(t, err)

	err = pgxscan.Select(ctx, connectionSource.Connection, &inserted,
		`SELECT * FROM funding_period_data_points where market_id = $1 and funding_period_seq = $2 and data_point_type = $3 and vega_time = $4`,
		stores.markets[0].ID, 1, entities.FundingPeriodDataPointSourceExternal, stores.blocks[4].VegaTime)
	require.NoError(t, err)
	assert.Len(t, inserted, 1)
	assert.Equal(t, dp2, inserted[0])
}

func addFundingPeriodsAndDataPoints(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	stores.periods = []entities.FundingPeriod{
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 1,
			StartTime:        stores.blocks[1].VegaTime,
			EndTime:          nil,
			FundingPayment:   ptr.From(num.DecimalFromFloat(1)),
			FundingRate:      ptr.From(num.DecimalFromFloat(1)),
			ExternalTwap:     ptr.From(num.DecimalFromFloat(1)),
			InternalTwap:     ptr.From(num.DecimalFromFloat(1)),
			VegaTime:         stores.blocks[1].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 2,
			StartTime:        stores.blocks[3].VegaTime,
			EndTime:          nil,
			FundingPayment:   ptr.From(num.DecimalFromFloat(2)),
			FundingRate:      ptr.From(num.DecimalFromFloat(2)),
			ExternalTwap:     ptr.From(num.DecimalFromFloat(2)),
			InternalTwap:     ptr.From(num.DecimalFromFloat(2)),
			VegaTime:         stores.blocks[3].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 3,
			StartTime:        stores.blocks[5].VegaTime,
			EndTime:          nil,
			FundingPayment:   ptr.From(num.DecimalFromFloat(3)),
			FundingRate:      ptr.From(num.DecimalFromFloat(3)),
			ExternalTwap:     ptr.From(num.DecimalFromFloat(3)),
			InternalTwap:     ptr.From(num.DecimalFromFloat(3)),
			VegaTime:         stores.blocks[5].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 4,
			StartTime:        stores.blocks[7].VegaTime,
			EndTime:          nil,
			FundingPayment:   ptr.From(num.DecimalFromFloat(5)),
			FundingRate:      ptr.From(num.DecimalFromFloat(5)),
			ExternalTwap:     ptr.From(num.DecimalFromFloat(5)),
			InternalTwap:     ptr.From(num.DecimalFromFloat(5)),
			VegaTime:         stores.blocks[7].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 5,
			StartTime:        stores.blocks[9].VegaTime,
			EndTime:          nil,
			FundingPayment:   ptr.From(num.DecimalFromFloat(5)),
			FundingRate:      ptr.From(num.DecimalFromFloat(5)),
			ExternalTwap:     ptr.From(num.DecimalFromFloat(5)),
			InternalTwap:     ptr.From(num.DecimalFromFloat(5)),
			VegaTime:         stores.blocks[9].VegaTime,
			TxHash:           generateTxHash(),
		},
	}

	stores.dataPoints = []entities.FundingPeriodDataPoint{
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 1,
			DataPointType:    entities.FundingPeriodDataPointSourceExternal,
			Price:            num.DecimalFromFloat(1.0),
			Twap:             num.DecimalFromFloat(1.0),
			Timestamp:        stores.blocks[2].VegaTime,
			VegaTime:         stores.blocks[2].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 1,
			DataPointType:    entities.FundingPeriodDataPointSourceExternal,
			Price:            num.DecimalFromFloat(1.0),
			Twap:             num.DecimalFromFloat(1.0),
			Timestamp:        stores.blocks[3].VegaTime,
			VegaTime:         stores.blocks[3].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 1,
			DataPointType:    entities.FundingPeriodDataPointSourceInternal,
			Price:            num.DecimalFromFloat(1.0),
			Twap:             num.DecimalFromFloat(1.0),
			Timestamp:        stores.blocks[4].VegaTime,
			VegaTime:         stores.blocks[4].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 2,
			DataPointType:    entities.FundingPeriodDataPointSourceExternal,
			Price:            num.DecimalFromFloat(1.0),
			Twap:             num.DecimalFromFloat(1.0),
			Timestamp:        stores.blocks[5].VegaTime,
			VegaTime:         stores.blocks[5].VegaTime,
			TxHash:           generateTxHash(),
		},
		{
			MarketID:         stores.markets[0].ID,
			FundingPeriodSeq: 3,
			DataPointType:    entities.FundingPeriodDataPointSourceExternal,
			Price:            num.DecimalFromFloat(1.0),
			Twap:             num.DecimalFromFloat(1.0),
			Timestamp:        stores.blocks[6].VegaTime,
			VegaTime:         stores.blocks[6].VegaTime,
			TxHash:           generateTxHash(),
		},
	}

	for _, period := range stores.periods {
		err := stores.fp.AddFundingPeriod(ctx, &period)
		require.NoError(t, err)
	}

	for _, dataPoint := range stores.dataPoints {
		err := stores.fp.AddDataPoint(ctx, &dataPoint)
		require.NoError(t, err)
	}

	// Let's make sure the data is ordered correctly just in case we add data points, but not in order
	sort.Slice(stores.periods, func(i, j int) bool {
		return stores.periods[i].VegaTime.After(stores.periods[j].VegaTime) ||
			stores.periods[i].MarketID < stores.periods[j].MarketID ||
			stores.periods[i].FundingPeriodSeq < stores.periods[j].FundingPeriodSeq
	})

	sort.Slice(stores.dataPoints, func(i, j int) bool {
		return stores.dataPoints[i].VegaTime.After(stores.dataPoints[j].VegaTime) ||
			stores.dataPoints[i].MarketID < stores.dataPoints[j].MarketID ||
			stores.dataPoints[i].FundingPeriodSeq < stores.dataPoints[j].FundingPeriodSeq ||
			stores.dataPoints[i].DataPointType < stores.dataPoints[j].DataPointType
	})
}

func TestFundingPeriodListFundingPeriods(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	stores := setupFundingPeriodTests(ctx, t)

	addFundingPeriodsAndDataPoints(t, ctx, stores)

	t.Run("should return the first page of funding periods when no sequence number is given", func(t *testing.T) {
		testListFundingPeriodsMarketNoSequence(t, ctx, stores)
	})
	t.Run("should return the specific funding period when the market and sequence number is given", func(t *testing.T) {
		testListFundingPeriodForMarketSequence(t, ctx, stores)
	})
	t.Run("should return the page of funding periods when pagination control is provided", func(t *testing.T) {
		testListFundingPeriodPagination(t, ctx, stores)
	})
}

func testListFundingPeriodsMarketNoSequence(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	dateRange := entities.DateRange{}

	got, pageInfo, err := stores.fp.ListFundingPeriods(ctx, stores.markets[0].ID, dateRange, pagination)
	require.NoError(t, err)
	want := stores.periods

	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testListFundingPeriodForMarketSequence(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	dateRange := entities.DateRange{
		End: ptr.From(stores.periods[3].StartTime),
	}
	got, pageInfo, err := stores.fp.ListFundingPeriods(ctx, stores.markets[0].ID, dateRange, pagination)
	require.NoError(t, err)
	want := stores.periods[4:]

	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testListFundingPeriodPagination(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	var first, last int32
	var after, before string

	t.Run("should return the first page when first is specified with no cursor", func(t *testing.T) {
		first = 2
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)

		dateRange := entities.DateRange{}
		got, pageInfo, err := stores.fp.ListFundingPeriods(ctx, stores.markets[0].ID, dateRange, pagination)
		require.NoError(t, err)
		want := stores.periods[:2]

		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the first page after the cursor when first is specified with a cursor", func(t *testing.T) {
		first = 2
		after = stores.periods[1].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		dateRange := entities.DateRange{}
		got, pageInfo, err := stores.fp.ListFundingPeriods(ctx, stores.markets[0].ID, dateRange, pagination)
		require.NoError(t, err)
		want := stores.periods[2:4]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the last page when last is specified with no cursor", func(t *testing.T) {
		last = 2
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		dateRange := entities.DateRange{}
		got, pageInfo, err := stores.fp.ListFundingPeriods(ctx, stores.markets[0].ID, dateRange, pagination)
		require.NoError(t, err)
		want := stores.periods[len(stores.periods)-2:]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("should return the last page before the cursor when last is specified with a cursor", func(t *testing.T) {
		last = 2
		before = stores.periods[3].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		dateRange := entities.DateRange{}
		got, pageInfo, err := stores.fp.ListFundingPeriods(ctx, stores.markets[0].ID, dateRange, pagination)
		require.NoError(t, err)
		want := stores.periods[1:3]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestFundingPeriod_ListDataPoints(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	stores := setupFundingPeriodTests(ctx, t)

	addFundingPeriodsAndDataPoints(t, ctx, stores)

	t.Run("Should return the first page of data points for all sequences if none are specified", func(t *testing.T) {
		testListDataPointsAllSequences(t, ctx, stores)
	})
	t.Run("Should return the first page of data points for specified sequences only", func(t *testing.T) {
		testListDataPointsSpecifiedSequences(t, ctx, stores)
	})
	t.Run("should return the first page of data points when no source is specified", func(t *testing.T) {
		testListDataPointsNoSource(t, ctx, stores)
	})
	t.Run("should return the first page of data points for the specified source", func(t *testing.T) {
		testListDataPointsForSource(t, ctx, stores)
	})
	t.Run("should return the page of data points when pagination control is provided", func(t *testing.T) {
		testListDataPointsPagination(t, ctx, stores)
	})
}

func testListDataPointsAllSequences(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	dateRange := entities.DateRange{}
	got, pageInfo, err := stores.fp.ListFundingPeriodDataPoints(ctx, stores.markets[0].ID, dateRange, nil, nil, pagination)
	require.NoError(t, err)
	want := stores.dataPoints[:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testListDataPointsSpecifiedSequences(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	dateRange := entities.DateRange{}
	got, pageInfo, err := stores.fp.ListFundingPeriodDataPoints(ctx, stores.markets[0].ID, dateRange, nil, nil, pagination)
	require.NoError(t, err)
	want := stores.dataPoints[:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}, pageInfo)
}

func testListDataPointsNoSource(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	want := stores.dataPoints[2:]
	wantPageInfo := entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}
	// Seq 1
	dateRange := entities.DateRange{
		Start: ptr.From(stores.dataPoints[4].Timestamp),
		End:   ptr.From(stores.dataPoints[1].Timestamp),
	}
	testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange, nil, nil, pagination, want, wantPageInfo)
	want = stores.dataPoints[1:]
	wantPageInfo = entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[len(want)-1].Cursor().Encode(),
	}
	// Seq 1 and 2
	dateRange = entities.DateRange{
		Start: ptr.From(stores.dataPoints[4].Timestamp),
		End:   ptr.From(stores.dataPoints[0].Timestamp),
	}
	testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange, nil, nil, pagination, want, wantPageInfo)
}

func testFundingPeriodsListDataPoints(t *testing.T, ctx context.Context, fp *sqlstore.FundingPeriods, marketID entities.MarketID, dateRange entities.DateRange,
	source *entities.FundingPeriodDataPointSource, seq *uint64, pagination entities.CursorPagination, want []entities.FundingPeriodDataPoint,
	wantPageInfo entities.PageInfo,
) {
	t.Helper()
	got, pageInfo, err := fp.ListFundingPeriodDataPoints(ctx, marketID, dateRange, source, seq, pagination)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, wantPageInfo, pageInfo)
}

func testListDataPointsForSource(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	t.Run("should filter by just source", func(t *testing.T) {
		want := []entities.FundingPeriodDataPoint{}
		want = append(want, stores.dataPoints[:2]...)
		want = append(want, stores.dataPoints[3:]...)
		wantPageInfo := entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}
		dateRange := entities.DateRange{}
		testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange,
			ptr.From(entities.FundingPeriodDataPointSourceExternal), nil, pagination, want, wantPageInfo)
	})

	t.Run("should filter by date range and source", func(t *testing.T) {
		want := []entities.FundingPeriodDataPoint{}
		want = append(want, stores.dataPoints[1])
		want = append(want, stores.dataPoints[3:]...)
		wantPageInfo := entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}
		// sequence 1 and 2
		dateRange := entities.DateRange{
			Start: ptr.From(stores.dataPoints[4].Timestamp),
			End:   ptr.From(stores.dataPoints[0].Timestamp),
		}
		testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange,
			ptr.From(entities.FundingPeriodDataPointSourceExternal), nil, pagination, want, wantPageInfo)
	})
	t.Run("should filter by seq", func(t *testing.T) {
		want := []entities.FundingPeriodDataPoint{}
		want = append(want, stores.dataPoints[1])
		wantPageInfo := entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}
		dateRange := entities.DateRange{}
		// sequence 2 only, which is one point
		testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange,
			nil, ptr.From(uint64(2)), pagination, want, wantPageInfo)
	})
}

func testListDataPointsPagination(t *testing.T, ctx context.Context, stores *fundingPeriodTestStores) {
	t.Helper()
	var first, last int32
	var after, before string

	t.Run("should return the first page when first is specified with no cursor", func(t *testing.T) {
		first = 2
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)

		want := stores.dataPoints[:2]
		wantPageInfo := entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}
		dateRange := entities.DateRange{}
		testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange,
			nil, nil, pagination, want, wantPageInfo)
	})

	t.Run("should return the first page after the cursor when first is specified with a cursor", func(t *testing.T) {
		first = 2
		after = stores.dataPoints[1].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		want := stores.dataPoints[2:4]
		wantPageInfo := entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}
		dateRange := entities.DateRange{}
		testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange,
			nil, nil, pagination, want, wantPageInfo)
	})

	t.Run("should return the last page when last is specified with no cursor", func(t *testing.T) {
		last = 2
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		want := stores.dataPoints[len(stores.dataPoints)-2:]
		wantPageInfo := entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}
		dateRange := entities.DateRange{}
		testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange,
			nil, nil, pagination, want, wantPageInfo)
	})

	t.Run("should return the last page before the cursor when last is specified with a cursor", func(t *testing.T) {
		last = 2
		before = stores.dataPoints[3].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		want := stores.dataPoints[1:3]
		wantPageInfo := entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}
		dateRange := entities.DateRange{}
		testFundingPeriodsListDataPoints(t, ctx, stores.fp, stores.markets[0].ID, dateRange,
			nil, nil, pagination, want, wantPageInfo)
	})
}
