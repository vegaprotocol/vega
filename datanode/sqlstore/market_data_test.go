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

package sqlstore_test

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"github.com/jackc/pgx/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	csvColumnMarket = iota
	_
	csvColumnVegaTime
	csvColumnMarkPrice
	csvColumnBestBidPrice
	csvColumnBestBidVolume
	csvColumnBestOfferPrice
	csvColumnBestOfferVolume
	csvColumnBestStaticBidPrice
	csvColumnBestStaticBidVolume
	csvColumnBestStaticOfferPrice
	csvColumnBestStaticOfferVolume
	csvColumnMidPrice
	csvColumnStaticMidPrice
	csvColumnOpenInterest
	csvColumnAuctionEnd
	csvColumnAuctionStart
	csvColumnIndicativePrice
	csvColumnIndicativeVolume
	csvColumnMarketTradingMode
	csvColumnAuctionTrigger
	csvColumnExtensionTrigger
	csvColumnTargetStake
	csvColumnSuppliedStake
	csvColumnPriceMonitoringBounds
	csvColumnMarketValueProxy
	csvColumnLiquidityProviderFeeShares
)

func Test_MarketData(t *testing.T) {
	t.Run("Add should insert a valid market data record", shouldInsertAValidMarketDataRecord)
	t.Run("Get should return the latest market data record for a given market", getLatestMarketData)
	t.Run("GetBetweenDatesByID should return the all the market data between dates given for the specified market", getAllForMarketBetweenDates)
	t.Run("GetFromDateByID should return all market data for a given market with date greater than or equal to the given date", getForMarketFromDate)
	t.Run("GetToDateByID should return all market data for a given market with date less than or equal to the given date", getForMarketToDate)
}

func shouldInsertAValidMarketDataRecord(t *testing.T) {
	bs := sqlstore.NewBlocks(connectionSource)
	md := sqlstore.NewMarketData(connectionSource)

	DeleteEverything()

	config := sqlstore.NewDefaultConfig()
	config.ConnectionConfig.Port = testDBPort

	connStr := connectionString(config.ConnectionConfig)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	require.NoError(t, err)
	var rowCount int

	err = conn.QueryRow(ctx, `select count(*) from market_data`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)

	err = md.Add(&entities.MarketData{
		Market:            entities.NewMarketID("deadbeef"),
		MarketTradingMode: "TRADING_MODE_MONITORING_AUCTION",
		AuctionTrigger:    "AUCTION_TRIGGER_LIQUIDITY",
		ExtensionTrigger:  "AUCTION_TRIGGER_UNSPECIFIED",
		VegaTime:          block.VegaTime,
	})
	require.NoError(t, err)

	_, err = md.Flush(context.Background())
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from market_data`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func connectionString(config sqlstore.ConnectionConfig) string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database)
}

func getLatestMarketData(t *testing.T) {
	store, err := setupMarketData(t)
	if err != nil {
		t.Fatalf("could not set up test: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	marketID := entities.NewMarketID("8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8")

	want := entities.MarketData{
		MarkPrice:             mustParseDecimal(t, "999992587"),
		BestBidPrice:          mustParseDecimal(t, "1000056152"),
		BestBidVolume:         3,
		BestOfferPrice:        mustParseDecimal(t, "999945379"),
		BestOfferVolume:       1,
		BestStaticBidPrice:    mustParseDecimal(t, "1000056152"),
		BestStaticBidVolume:   3,
		BestStaticOfferPrice:  mustParseDecimal(t, "999945379"),
		BestStaticOfferVolume: 1,
		MidPrice:              mustParseDecimal(t, "1000000765"),
		StaticMidPrice:        mustParseDecimal(t, "1000000765"),
		Market:                marketID,
		OpenInterest:          27,
		AuctionEnd:            1644573937314794695,
		AuctionStart:          1644573911314794695,
		IndicativePrice:       mustParseDecimal(t, "1000026624"),
		IndicativeVolume:      3,
		MarketTradingMode:     "TRADING_MODE_MONITORING_AUCTION",
		AuctionTrigger:        "AUCTION_TRIGGER_LIQUIDITY",
		ExtensionTrigger:      "AUCTION_TRIGGER_UNSPECIFIED",
		TargetStake:           mustParseDecimal(t, "67499499622"),
		SuppliedStake:         mustParseDecimal(t, "50000000000"),
		PriceMonitoringBounds: nil,
		MarketValueProxy:      "194290093211464.7413030152957024",
		LiquidityProviderFeeShares: []*entities.LiquidityProviderFeeShare{
			{
				Party:                 "af2bb48edd738353fcd7a2b6cea4821dd2382ec95497954535278dfbfff7b5b5",
				EquityLikeShare:       1,
				AverageEntryValuation: 50000000000,
			},
		},
		VegaTime: time.Date(2022, 2, 11, 10, 5, 41, 0, time.UTC),
	}
	got, err := store.GetMarketDataByID(ctx, "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8")
	assert.NoError(t, err)

	assert.True(t, want.Equal(got))
}

func getAllForMarketBetweenDates(t *testing.T) {
	store, err := setupMarketData(t)
	if err != nil {
		t.Fatalf("could not set up test: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	market := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"

	startDate := time.Date(2022, 2, 11, 10, 5, 30, 0, time.UTC)
	endDate := time.Date(2022, 2, 11, 10, 6, 0, 0, time.UTC)

	offsetPagination := entities.OffsetPagination{}

	t.Run("should return all results if no offset pagination is provided", func(t *testing.T) {
		got, _, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, offsetPagination)
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
	})

	t.Run("should return all results if no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:31.000175Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return all results if no cursor pagination is provided - newest first", func(t *testing.T) {
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, entities.CursorPagination{NewestFirst: true})
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:31.000175Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if offset pagination is provided", func(t *testing.T) {
		offsetPagination.Skip = 5
		offsetPagination.Limit = 5

		got, _, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, offsetPagination)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(got))
	})

	t.Run("should return page of results if cursor pagination is provided with first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:31.000175Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:36.000179Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if cursor pagination is provided with first - newest first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:36.000179Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with first and after parameter", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor("2022-02-11T10:05:32.000176Z").Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:33.000177Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:39.000181Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with first and after parameter - newest first", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor("2022-02-11T10:05:40.000182Z").Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:39.000181Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:33.000177Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if cursor pagination is provided with last", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:36.000179Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if cursor pagination is provided with last - newest first", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:36.000179Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:31.000175Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with last and before parameter", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor("2022-02-11T10:05:40.000182Z").Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:33.000177Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:39.000181Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with last and before parameter - newest first", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor("2022-02-11T10:05:32.000176Z").Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetBetweenDatesByID(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:39.000181Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:33.000177Z").Encode(),
		}, pageInfo)
	})
}

func getForMarketFromDate(t *testing.T) {
	store, err := setupMarketData(t)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	startDate := time.Date(2022, 2, 11, 10, 5, 0, 0, time.UTC)

	market := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"

	pagination := entities.OffsetPagination{}

	t.Run("should return all results if no offset pagination is provided", func(t *testing.T) {
		got, _, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 32, len(got))
	})

	t.Run("should return all results if no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 32, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:00.000152Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return all results if no cursor pagination is provided - newest first", func(t *testing.T) {
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, entities.CursorPagination{NewestFirst: true})
		assert.NoError(t, err)
		assert.Equal(t, 32, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:00.000152Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if offset pagination is provided", func(t *testing.T) {
		pagination.Skip = 5
		pagination.Limit = 5
		got, _, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
	})

	t.Run("should return a page of results if cursor pagination is provided with first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:00.000152Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:05.000156Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first - newest first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:36.000179Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor("2022-02-11T10:05:09.000159Z").Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:11.00016Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:16.000164Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor("2022-02-11T10:05:09.000159Z").Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:08.000158Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:03.000154Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:36.000179Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:41.000183Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last - newest first", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:05.000156Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:00.000152Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor("2022-02-11T10:05:37.00018Z").Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:31.000175Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:36.000179Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before - newest first", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor("2022-02-11T10:05:20.000167Z").Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetFromDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:05:27.000172Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:05:22.000168Z").Encode(),
		}, pageInfo)
	})
}

func getForMarketToDate(t *testing.T) {
	store, err := setupMarketData(t)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	startDate := time.Date(2022, 2, 11, 10, 2, 0, 0, time.UTC)

	market := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"

	pagination := entities.OffsetPagination{}

	t.Run("should return all results if no pagination is provided", func(t *testing.T) {
		got, _, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 18, len(got))
	})

	t.Run("should return all results if no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 18, len(got))
		wantStartCursor := entities.NewCursor("2022-02-11T10:01:35Z").Encode()
		wantEndCursor := entities.NewCursor("2022-02-11T10:02:00.000017Z").Encode()
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     wantStartCursor,
			EndCursor:       wantEndCursor,
		}, pageInfo)
	})

	t.Run("should return all results if no cursor pagination is provided - newest first", func(t *testing.T) {
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, entities.CursorPagination{NewestFirst: true})
		assert.NoError(t, err)
		assert.Equal(t, 18, len(got))
		wantStartCursor := entities.NewCursor("2022-02-11T10:02:00.000017Z").Encode()
		wantEndCursor := entities.NewCursor("2022-02-11T10:01:35Z").Encode()
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     wantStartCursor,
			EndCursor:       wantEndCursor,
		}, pageInfo)
	})

	t.Run("should return a page of results if offset pagination is provided", func(t *testing.T) {
		pagination.Skip = 10
		pagination.Limit = 10
		got, _, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 8, len(got))
	})

	t.Run("should return a page of results if cursor pagination is provided with first", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:01:35Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:01:49.000009Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first - newest first", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:02:00.000017Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:01:47.000008Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after", func(t *testing.T) {
		first := int32(10)
		after := entities.NewCursor("2022-02-11T10:01:49.000009Z").Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 8, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:01:50.00001Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:02:00.000017Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after - newest first", func(t *testing.T) {
		first := int32(10)
		after := entities.NewCursor("2022-02-11T10:01:47.000008Z").Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 8, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:01:46.000007Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:01:35Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last", func(t *testing.T) {
		last := int32(10)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:01:47.000008Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:02:00.000017Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last - newest first", func(t *testing.T) {
		last := int32(10)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     entities.NewCursor("2022-02-11T10:01:49.000009Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:01:35Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before", func(t *testing.T) {
		last := int32(10)
		before := entities.NewCursor("2022-02-11T10:01:49.000009Z").Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:01:35Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:01:47.000008Z").Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before - newest first", func(t *testing.T) {
		last := int32(10)
		before := entities.NewCursor("2022-02-11T10:01:47.000008Z").Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetToDateByID(ctx, market, startDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     entities.NewCursor("2022-02-11T10:02:00.000017Z").Encode(),
			EndCursor:       entities.NewCursor("2022-02-11T10:01:49.000009Z").Encode(),
		}, pageInfo)
	})
}

func setupMarketData(t *testing.T) (*sqlstore.MarketData, error) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	md := sqlstore.NewMarketData(connectionSource)

	DeleteEverything()

	f, err := os.Open(filepath.Join("testdata", "marketdata.csv"))
	if err != nil {
		return nil, err
	}

	defer f.Close()

	reader := csv.NewReader(bufio.NewReader(f))

	var hash []byte
	hash, err = hex.DecodeString("deadbeef")
	assert.NoError(t, err)

	seqNum := 0
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		marketData := csvToMarketData(t, line, seqNum)
		seqNum++

		// Postgres only stores timestamps in microsecond resolution
		block := entities.Block{
			VegaTime: marketData.VegaTime,
			Height:   2,
			Hash:     hash,
		}

		// Add it to the database
		_ = bs.Add(context.Background(), block)

		err = md.Add(marketData)
		require.NoError(t, err)
	}
	_, err = md.Flush(context.Background())
	require.NoError(t, err)

	return md, nil
}

func mustParseDecimal(t *testing.T, value string) decimal.Decimal {
	d, err := decimal.NewFromString(value)
	if err != nil {
		t.Fatalf("could not parse decimal value: %s", err)
	}

	return d
}

func mustParseTimestamp(t *testing.T, value string) time.Time {
	const dbDateFormat = "2006-01-02 15:04:05.999999 -07:00"
	ts, err := time.Parse(dbDateFormat, value)
	if err != nil {
		t.Fatalf("could not parse time: %s", err)
	}

	return ts
}

func mustParseInt64(t *testing.T, value string) int64 {
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatalf("could not parse int64: %s", err)
	}

	return i
}

func mustParsePriceMonitoringBounds(t *testing.T, value string) []*entities.PriceMonitoringBound {
	if strings.ToLower(value) == "null" {
		return nil
	}

	var bounds []*entities.PriceMonitoringBound

	err := json.Unmarshal([]byte(value), &bounds)
	if err != nil {
		t.Fatalf("could not parse Price Monitoring Bounds: %s", err)
	}

	return bounds
}

func mustParseLiquidity(t *testing.T, value string) []*entities.LiquidityProviderFeeShare {
	if strings.ToLower(value) == "null" {
		return nil
	}

	var liquidity []*entities.LiquidityProviderFeeShare

	err := json.Unmarshal([]byte(value), &liquidity)
	if err != nil {
		t.Fatalf("could not parse Liquidity Provider Fee Share: %s", err)
	}

	return liquidity
}

func csvToMarketData(t *testing.T, line []string, seqNum int) *entities.MarketData {
	t.Helper()

	vegaTime := mustParseTimestamp(t, line[csvColumnVegaTime])
	syntheticTime := vegaTime.Add(time.Duration(seqNum) * time.Microsecond)

	return &entities.MarketData{
		MarkPrice:                  mustParseDecimal(t, line[csvColumnMarkPrice]),
		BestBidPrice:               mustParseDecimal(t, line[csvColumnBestBidPrice]),
		BestBidVolume:              mustParseInt64(t, line[csvColumnBestBidVolume]),
		BestOfferPrice:             mustParseDecimal(t, line[csvColumnBestOfferPrice]),
		BestOfferVolume:            mustParseInt64(t, line[csvColumnBestOfferVolume]),
		BestStaticBidPrice:         mustParseDecimal(t, line[csvColumnBestStaticBidPrice]),
		BestStaticBidVolume:        mustParseInt64(t, line[csvColumnBestStaticBidVolume]),
		BestStaticOfferPrice:       mustParseDecimal(t, line[csvColumnBestStaticOfferPrice]),
		BestStaticOfferVolume:      mustParseInt64(t, line[csvColumnBestStaticOfferVolume]),
		MidPrice:                   mustParseDecimal(t, line[csvColumnMidPrice]),
		StaticMidPrice:             mustParseDecimal(t, line[csvColumnStaticMidPrice]),
		Market:                     entities.NewMarketID(line[csvColumnMarket]),
		OpenInterest:               mustParseInt64(t, line[csvColumnOpenInterest]),
		AuctionEnd:                 mustParseInt64(t, line[csvColumnAuctionEnd]),
		AuctionStart:               mustParseInt64(t, line[csvColumnAuctionStart]),
		IndicativePrice:            mustParseDecimal(t, line[csvColumnIndicativePrice]),
		IndicativeVolume:           mustParseInt64(t, line[csvColumnIndicativeVolume]),
		MarketTradingMode:          line[csvColumnMarketTradingMode],
		AuctionTrigger:             line[csvColumnAuctionTrigger],
		ExtensionTrigger:           line[csvColumnExtensionTrigger],
		TargetStake:                mustParseDecimal(t, line[csvColumnTargetStake]),
		SuppliedStake:              mustParseDecimal(t, line[csvColumnSuppliedStake]),
		PriceMonitoringBounds:      mustParsePriceMonitoringBounds(t, line[csvColumnPriceMonitoringBounds]),
		MarketValueProxy:           line[csvColumnMarketValueProxy],
		LiquidityProviderFeeShares: mustParseLiquidity(t, line[csvColumnLiquidityProviderFeeShares]),
		VegaTime:                   vegaTime,
		SeqNum:                     uint64(seqNum),
		SyntheticTime:              syntheticTime,
	}
}
