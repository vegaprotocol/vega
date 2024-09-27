// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore_test

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

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
	csvColumnMarketState
	csvColumnMarketGrowth
	csvColumnLastTradedPrice
)

func Test_MarketData(t *testing.T) {
	t.Run("Add should insert a valid market data record", shouldInsertAValidMarketDataRecord)
	t.Run("Get should return the latest market data record for a given market", getLatestMarketData)
	t.Run("GetHistoricMarketData should return the all the market data between dates given for the specified market", getAllForMarketBetweenDates)
	t.Run("GetHistoricMarketData should return all market data for a given market with date greater than or equal to the given date", getForMarketFromDate)
	t.Run("GetHistoricMarketData should return all market data for a given market with date less than or equal to the given date", getForMarketToDate)
	t.Run("GetHistoricMarketData should return all market data when no start or end is provided", TestGetAllMarketData)
	t.Run("Add should work for all valid enumerations values of auction trigger", shouldWorkForAllValuesOfAuctionTrigger)
	t.Run("Add should work for all valid enumerations values of composite price type", shouldWorkForAllValuesOfCompositePriceType)
}

func shouldWorkForAllValuesOfCompositePriceType(t *testing.T) {
	var priceType vegapb.CompositePriceType
	enums := getEnums(t, priceType)
	assert.Len(t, enums, 4)

	for e, pt := range enums {
		t.Run(pt, func(t *testing.T) {
			ctx := tempTransaction(t)

			addMarketData(t, ctx, "AUCTION_TRIGGER_LIQUIDITY", pt)
			var got entities.MarketData

			err := connectionSource.QueryRow(ctx, `select mark_price_type from market_data`).Scan(&got.MarkPriceType)
			require.NoError(t, err)

			mdProto := got.ToProto()

			assert.Equal(t, vegapb.CompositePriceType(e), mdProto.MarkPriceType)
		})
	}
}

func addMarketData(t *testing.T, ctx context.Context, trigger, priceType string) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, bs)

	md := sqlstore.NewMarketData(connectionSource)
	err := md.Add(&entities.MarketData{
		Market:            entities.MarketID("deadbeef"),
		MarketTradingMode: "TRADING_MODE_MONITORING_AUCTION",
		MarketState:       "STATE_ACTIVE",
		AuctionTrigger:    trigger,
		ExtensionTrigger:  trigger,
		MarkPriceType:     priceType,
		PriceMonitoringBounds: []*vega.PriceMonitoringBounds{
			{
				MinValidPrice: "1",
				MaxValidPrice: "2",
				Trigger: &vega.PriceMonitoringTrigger{
					Horizon:          100,
					Probability:      "0.5",
					AuctionExtension: 200,
				},
				ReferencePrice: "3",
			},
		},
		VegaTime: block.VegaTime,
	})
	require.NoError(t, err)

	_, err = md.Flush(ctx)
	require.NoError(t, err)
}

func shouldWorkForAllValuesOfAuctionTrigger(t *testing.T) {
	var auctionTrigger vegapb.AuctionTrigger
	enums := getEnums(t, auctionTrigger)
	require.Len(t, enums, 10)

	for e, trigger := range enums {
		t.Run(trigger, func(t *testing.T) {
			ctx := tempTransaction(t)

			addMarketData(t, ctx, trigger, "COMPOSITE_PRICE_TYPE_LAST_TRADE")
			var got entities.MarketData

			err := connectionSource.QueryRow(ctx, `select auction_trigger from market_data`).Scan(&got.AuctionTrigger)
			require.NoError(t, err)

			mdProto := got.ToProto()

			assert.Equal(t, vegapb.AuctionTrigger(e), mdProto.Trigger)
		})
	}
}

func shouldInsertAValidMarketDataRecord(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	md := sqlstore.NewMarketData(connectionSource)

	var rowCount int

	err := connectionSource.QueryRow(ctx, `select count(*) from market_data`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)

	err = md.Add(&entities.MarketData{
		Market:            entities.MarketID("deadbeef"),
		MarketTradingMode: "TRADING_MODE_MONITORING_AUCTION",
		MarketState:       "STATE_ACTIVE",
		AuctionTrigger:    "AUCTION_TRIGGER_LIQUIDITY",
		ExtensionTrigger:  "AUCTION_TRIGGER_UNSPECIFIED",
		MarkPriceType:     "COMPOSITE_PRICE_TYPE_UNSPECIFIED",
		PriceMonitoringBounds: []*vega.PriceMonitoringBounds{
			{
				MinValidPrice: "1",
				MaxValidPrice: "2",
				Trigger: &vega.PriceMonitoringTrigger{
					Horizon:          100,
					Probability:      "0.5",
					AuctionExtension: 200,
				},
				ReferencePrice: "3",
			},
		},
		VegaTime: block.VegaTime,
	})
	require.NoError(t, err)

	_, err = md.Flush(ctx)
	require.NoError(t, err)

	err = connectionSource.QueryRow(ctx, `select count(*) from market_data`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func getLatestMarketData(t *testing.T) {
	ctx := tempTransaction(t)

	store, err := setupMarketData(t, ctx)
	if err != nil {
		t.Fatalf("could not set up test: %s", err)
	}

	marketID := entities.MarketID("8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8")

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
		MarketState:           "STATE_ACTIVE",
		AuctionTrigger:        "AUCTION_TRIGGER_LIQUIDITY",
		ExtensionTrigger:      "AUCTION_TRIGGER_UNSPECIFIED",
		MarkPriceType:         "COMPOSITE_PRICE_TYPE_LAST_TRADE",
		TargetStake:           mustParseDecimal(t, "67499499622"),
		SuppliedStake:         mustParseDecimal(t, "50000000000"),
		PriceMonitoringBounds: []*vega.PriceMonitoringBounds{
			{
				MinValidPrice: "1",
				MaxValidPrice: "2",
				Trigger: &vega.PriceMonitoringTrigger{
					Horizon:          100,
					Probability:      "0.5",
					AuctionExtension: 200,
				},
				ReferencePrice: "3",
			},
		},
		MarketValueProxy: "194290093211464.7413030152957024",
		LiquidityProviderFeeShares: []*vega.LiquidityProviderFeeShare{
			{
				Party:                 "af2bb48edd738353fcd7a2b6cea4821dd2382ec95497954535278dfbfff7b5b5",
				EquityLikeShare:       "1",
				AverageEntryValuation: "50000000000",
				AverageScore:          "123",
			},
		},
		MarketGrowth:    num.DecimalZero(),
		LastTradedPrice: mustParseDecimal(t, "999992588"),
		ProductData: &entities.ProductData{
			ProductData: &vega.ProductData{
				Data: &vega.ProductData_PerpetualData{
					PerpetualData: &vega.PerpetualData{
						InternalCompositePrice:         "100",
						NextInternalCompositePriceCalc: 200,
						InternalCompositePriceType:     vega.CompositePriceType_COMPOSITE_PRICE_TYPE_LAST_TRADE,
					},
				},
			},
		},
	}
	got, err := store.GetMarketDataByID(ctx, "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8")
	require.NoError(t, err)

	require.Truef(t, want.Equal(got), "want: %#v\ngot: %#v\n", want, got)
}

func getAllForMarketBetweenDates(t *testing.T) {
	ctx := tempTransaction(t)

	store, err := setupMarketData(t, ctx)
	if err != nil {
		t.Fatalf("could not set up test: %s", err)
	}

	market := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"

	startDate := ptr.From(time.Date(2022, 2, 11, 10, 5, 30, 0, time.UTC))
	endDate := ptr.From(time.Date(2022, 2, 11, 10, 6, 0, 0, time.UTC))

	t.Run("should return all results if no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 2, 11, 10, 5, 31, 175000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return all results if no cursor pagination is provided - newest first", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, entities.CursorPagination{NewestFirst: true})
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 31, 175000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if cursor pagination is provided with first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 31, 175000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 36, 179000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if cursor pagination is provided with first - newest first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 36, 179000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with first and after parameter", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 32, 176000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 33, 177000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 39, 181000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with first and after parameter - newest first", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 40, 182000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 39, 181000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 33, 177000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if cursor pagination is provided with last", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 36, 179000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if cursor pagination is provided with last - newest first", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 36, 179000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 31, 175000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with last and before parameter", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor(
			entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 40, 182000, time.UTC).Local(),
			}.String()).Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 33, 177000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 39, 181000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return page of results if forward cursor pagination is provided with last and before parameter - newest first", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor(
			entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 32, 176000, time.UTC).Local(),
			}.String()).Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, endDate, pagination)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 39, 181000, time.UTC).Local(),
				}.String()).Encode(),
			EndCursor: entities.NewCursor(
				entities.MarketDataCursor{
					SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 33, 177000, time.UTC).Local(),
				}.String()).Encode(),
		}, pageInfo)
	})
}

func getForMarketFromDate(t *testing.T) {
	ctx := tempTransaction(t)

	store, err := setupMarketData(t, ctx)
	require.NoError(t, err)

	startDate := ptr.From(time.Date(2022, 2, 11, 10, 5, 0, 0, time.UTC))

	market := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"

	t.Run("should return all results if no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 32, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 0o0, 152000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return all results if no cursor pagination is provided - newest first", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, entities.CursorPagination{NewestFirst: true})
		assert.NoError(t, err)
		assert.Equal(t, 32, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 0o0, 152000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 0o0, 152000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 0o5, 156000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first - newest first", func(t *testing.T) {
		first := int32(5)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 2, 11, 10, 5, 41, 183000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 2, 11, 10, 5, 36, 179000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 2, 11, 10, 5, 9, 159000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 11, 160000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 16, 164000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after", func(t *testing.T) {
		first := int32(5)
		after := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 2, 11, 10, 5, 9, 159000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 2, 11, 10, 5, 8, 158000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 0o3, 154000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 36, 179000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last - newest first", func(t *testing.T) {
		last := int32(5)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 0o5, 156000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 0o0, 152000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 37, 180000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 31, 175000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 36, 179000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before - newest first", func(t *testing.T) {
		last := int32(5)
		before := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 20, 167000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 27, 172000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 22, 168000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})
}

func getForMarketToDate(t *testing.T) {
	ctx := tempTransaction(t)

	store, err := setupMarketData(t, ctx)
	require.NoError(t, err)

	endDate := ptr.From(time.Date(2022, 2, 11, 10, 2, 0, 0, time.UTC))

	market := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"

	t.Run("should return all results if no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 18, len(got))
		wantStartCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
		}.String()).Encode()
		wantEndCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o2, 0o0, 17000, time.UTC).Local(),
		}.String()).Encode()
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     wantStartCursor,
			EndCursor:       wantEndCursor,
		}, pageInfo)
	})

	t.Run("should return all results if no cursor pagination is provided - newest first", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, entities.CursorPagination{NewestFirst: true})
		assert.NoError(t, err)
		assert.Equal(t, 18, len(got))
		wantStartCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o2, 0o0, 17000, time.UTC).Local(),
		}.String()).Encode()
		wantEndCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
		}.String()).Encode()
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     wantStartCursor,
			EndCursor:       wantEndCursor,
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 49, 9000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first - newest first", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o2, 0o0, 17000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 47, 8000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after", func(t *testing.T) {
		first := int32(10)
		after := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 49, 9000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 8, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 50, 10000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o2, 0o0, 17000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first and after - newest first", func(t *testing.T) {
		first := int32(10)
		after := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 47, 8000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 8, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 46, 7000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last", func(t *testing.T) {
		last := int32(10)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 47, 8000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o2, 0o0, 17000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last - newest first", func(t *testing.T) {
		last := int32(10)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 49, 9000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before", func(t *testing.T) {
		last := int32(10)
		before := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 49, 9000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 47, 8000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with last and before - newest first", func(t *testing.T) {
		last := int32(10)
		before := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 47, 8000, time.UTC).Local(),
		}.String()).Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, endDate, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 9, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o2, 0o0, 17000, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 49, 9000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})
}

func TestGetAllMarketData(t *testing.T) {
	ctx := tempTransaction(t)

	store, err := setupMarketData(t, ctx)
	require.NoError(t, err)
	market := "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8"
	startDate := time.Date(2022, 2, 11, 0, 0, 0, 0, time.UTC)

	t.Run("should return all results if no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, &startDate, nil, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 184, len(got))
		wantStartCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
		}.String()).Encode()
		wantEndCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
		}.String()).Encode()
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     wantStartCursor,
			EndCursor:       wantEndCursor,
		}, pageInfo)
	})

	t.Run("should return a page of results if cursor pagination is provided with first", func(t *testing.T) {
		first := int32(10)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, &startDate, nil, pagination)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(got))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 35, 0, time.UTC).Local(),
			}.String()).Encode(),
			EndCursor: entities.NewCursor(entities.MarketDataCursor{
				SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o1, 49, 9000, time.UTC).Local(),
			}.String()).Encode(),
		}, pageInfo)
	})

	t.Run("should return the most recent record if no dates and no cursor pagination is provided", func(t *testing.T) {
		got, pageInfo, err := store.GetHistoricMarketData(ctx, market, nil, nil, entities.CursorPagination{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(got))
		wantStartCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
		}.String()).Encode()
		wantEndCursor := entities.NewCursor(entities.MarketDataCursor{
			SyntheticTime: time.Date(2022, 0o2, 11, 10, 0o5, 41, 183000, time.UTC).Local(),
		}.String()).Encode()
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     wantStartCursor,
			EndCursor:       wantEndCursor,
		}, pageInfo)
	})
}

func setupMarketData(t *testing.T, ctx context.Context) (*sqlstore.MarketData, error) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	md := sqlstore.NewMarketData(connectionSource)

	f, err := os.Open(filepath.Join("testdata", "marketdata.csv"))
	if err != nil {
		return nil, err
	}

	defer f.Close()

	reader := csv.NewReader(bufio.NewReader(f))

	var hash []byte
	hash, err = hex.DecodeString("deadbeef")
	assert.NoError(t, err)

	addedBlocksAt := make(map[int64]struct{})
	seqNum := 0
	for {
		line, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if len(line) == 0 {
			continue
		}

		marketData := csvToMarketData(t, line, seqNum)
		seqNum++

		if _, alreadyAdded := addedBlocksAt[marketData.VegaTime.UnixNano()]; !alreadyAdded {
			// Postgres only stores timestamps in microsecond resolution
			block := entities.Block{
				VegaTime: marketData.VegaTime,
				Height:   2,
				Hash:     hash,
			}

			// Add it to the database
			err = bs.Add(ctx, block)
			require.NoError(t, err)
			addedBlocksAt[marketData.VegaTime.UnixNano()] = struct{}{}
		}

		err = md.Add(marketData)
		require.NoError(t, err)
	}
	_, err = md.Flush(ctx)
	require.NoError(t, err)

	return md, nil
}

func mustParseDecimal(t *testing.T, value string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(value)
	if err != nil {
		t.Fatalf("could not parse decimal value: %s", err)
	}

	return d
}

func mustParseTimestamp(t *testing.T, value string) time.Time {
	t.Helper()
	const dbDateFormat = "2006-01-02 15:04:05.999999 -07:00"
	ts, err := time.Parse(dbDateFormat, value)
	if err != nil {
		t.Fatalf("could not parse time: %s", err)
	}

	return ts
}

func mustParseUInt64(t *testing.T, value string) uint64 {
	t.Helper()
	i, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		t.Fatalf("could not parse int64: %s", err)
	}

	return i
}

func mustParseInt64(t *testing.T, value string) int64 {
	t.Helper()
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatalf("could not parse int64: %s", err)
	}

	return i
}

func mustParsePriceMonitoringBounds(t *testing.T, value string) []*vega.PriceMonitoringBounds {
	t.Helper()
	if strings.ToLower(value) == "null" {
		return nil
	}

	var bounds []*vega.PriceMonitoringBounds

	err := json.Unmarshal([]byte(value), &bounds)
	if err != nil {
		t.Fatalf("could not parse Price Monitoring Bounds: %s", err)
	}

	return bounds
}

func mustParseLiquidity(t *testing.T, value string) []*vega.LiquidityProviderFeeShare {
	t.Helper()
	if strings.ToLower(value) == "null" {
		return nil
	}

	var liquidity []*vega.LiquidityProviderFeeShare

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
		BestBidVolume:              mustParseUInt64(t, line[csvColumnBestBidVolume]),
		BestOfferPrice:             mustParseDecimal(t, line[csvColumnBestOfferPrice]),
		BestOfferVolume:            mustParseUInt64(t, line[csvColumnBestOfferVolume]),
		BestStaticBidPrice:         mustParseDecimal(t, line[csvColumnBestStaticBidPrice]),
		BestStaticBidVolume:        mustParseUInt64(t, line[csvColumnBestStaticBidVolume]),
		BestStaticOfferPrice:       mustParseDecimal(t, line[csvColumnBestStaticOfferPrice]),
		BestStaticOfferVolume:      mustParseUInt64(t, line[csvColumnBestStaticOfferVolume]),
		MidPrice:                   mustParseDecimal(t, line[csvColumnMidPrice]),
		StaticMidPrice:             mustParseDecimal(t, line[csvColumnStaticMidPrice]),
		Market:                     entities.MarketID(line[csvColumnMarket]),
		OpenInterest:               mustParseUInt64(t, line[csvColumnOpenInterest]),
		AuctionEnd:                 mustParseInt64(t, line[csvColumnAuctionEnd]),
		AuctionStart:               mustParseInt64(t, line[csvColumnAuctionStart]),
		IndicativePrice:            mustParseDecimal(t, line[csvColumnIndicativePrice]),
		IndicativeVolume:           mustParseUInt64(t, line[csvColumnIndicativeVolume]),
		MarketTradingMode:          line[csvColumnMarketTradingMode],
		AuctionTrigger:             line[csvColumnAuctionTrigger],
		ExtensionTrigger:           line[csvColumnExtensionTrigger],
		TargetStake:                mustParseDecimal(t, line[csvColumnTargetStake]),
		SuppliedStake:              mustParseDecimal(t, line[csvColumnSuppliedStake]),
		PriceMonitoringBounds:      mustParsePriceMonitoringBounds(t, line[csvColumnPriceMonitoringBounds]),
		MarketValueProxy:           line[csvColumnMarketValueProxy],
		LiquidityProviderFeeShares: mustParseLiquidity(t, line[csvColumnLiquidityProviderFeeShares]),
		MarketState:                line[csvColumnMarketState],
		VegaTime:                   vegaTime,
		SeqNum:                     uint64(seqNum),
		SyntheticTime:              syntheticTime,
		MarketGrowth:               mustParseDecimal(t, line[csvColumnMarketGrowth]),
		LastTradedPrice:            mustParseDecimal(t, line[csvColumnLastTradedPrice]),
		ProductData: &entities.ProductData{
			ProductData: &vega.ProductData{
				Data: &vega.ProductData_PerpetualData{
					PerpetualData: &vega.PerpetualData{
						InternalCompositePrice:         "100",
						NextInternalCompositePriceCalc: 200,
						InternalCompositePriceType:     vega.CompositePriceType_COMPOSITE_PRICE_TYPE_LAST_TRADE,
					},
				},
			},
		},
		MarkPriceType: "COMPOSITE_PRICE_TYPE_LAST_TRADE",
	}
}
