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

package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/api/mocks"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/metadata"
)

//go:embed testdata/dummysegment.zip
var testData embed.FS

func makeFullSegment(from, to, dbVersion int64) segment.Full {
	return segment.Full{
		MetaData: segment.MetaData{
			Base: segment.Base{
				HeightFrom:      from,
				HeightTo:        to,
				DatabaseVersion: dbVersion,
				ChainID:         "test-chain-id",
			},
		},
	}
}

func TestExportNetworkHistory(t *testing.T) {
	req := &v2.ExportNetworkHistoryRequest{
		FromBlock: 1,
		ToBlock:   3000,
		Table:     v2.Table_TABLE_ORDERS,
	}

	ctrl := gomock.NewController(t)
	historyService := mocks.NewMockNetworkHistoryService(ctrl)

	testSegments := []segment.Full{
		makeFullSegment(1, 1000, 1),
		makeFullSegment(1001, 2000, 1),
		makeFullSegment(2001, 3000, 2),
	}

	historyService.EXPECT().ListAllHistorySegments().Times(1).Return(testSegments, nil)
	historyService.EXPECT().GetHistorySegmentReader(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
		func(ctx context.Context, id string) (io.ReadSeekCloser, int64, error) {
			reader, err := testData.Open("testdata/dummysegment.zip")
			require.NoError(t, err)
			info, _ := reader.Stat()
			return reader.(io.ReadSeekCloser), info.Size(), nil
		},
	)

	stream := &mockStream{}
	apiService := api.TradingDataServiceV2{
		NetworkHistoryService: historyService,
	}

	err := apiService.ExportNetworkHistory(req, stream)
	require.NoError(t, err)

	// Now check that we got a zip file with two CSV files in it; as we crossed a schema migration boundary
	require.Greater(t, len(stream.sent), 0)
	assert.Equal(t, stream.sent[0].ContentType, "application/zip")

	zipBytes := stream.sent[0].Data
	zipBuffer := bytes.NewReader(zipBytes)
	zipReader, err := zip.NewReader(zipBuffer, int64(len(zipBytes)))
	require.NoError(t, err)

	filenames := []string{}
	for _, file := range zipReader.File {
		filenames = append(filenames, file.Name)
		fileReader, err := file.Open()
		require.NoError(t, err)
		fileContents, err := io.ReadAll(fileReader)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(fileContents), "header row\nmock data, more mock data,"))
	}

	require.Equal(t, filenames, []string{
		"test-chain-id-orders-001-000001-002000.csv",
		"test-chain-id-orders-002-002001-003000.csv",
	})
}

func TestEstimatePosition(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	assetId := "assetID"
	marketId := "marketID"

	assetDecimals := 8
	marketDecimals := 3
	positionDecimalPlaces := 2
	marginFundingFactor := 0.95
	initialMarginScalingFactor := 1.5
	linearSlippageFactor := num.DecimalFromFloat(0.005)
	quadraticSlippageFactor := num.DecimalZero()
	rfLong := num.DecimalFromFloat(0.1)
	rfShort := num.DecimalFromFloat(0.2)

	markPrice := 123.456 * math.Pow10(marketDecimals)
	auctionEnd := int64(0)
	fundingPayment := 1234.56789

	asset := entities.Asset{
		Decimals: assetDecimals,
	}

	mkt := entities.Market{
		DecimalPlaces:           marketDecimals,
		PositionDecimalPlaces:   positionDecimalPlaces,
		LinearSlippageFactor:    &linearSlippageFactor,
		QuadraticSlippageFactor: &quadraticSlippageFactor,
		TradableInstrument: entities.TradableInstrument{
			TradableInstrument: &vega.TradableInstrument{
				Instrument: &vega.Instrument{
					Product: &vega.Instrument_Perpetual{
						Perpetual: &vega.Perpetual{
							SettlementAsset:     assetId,
							MarginFundingFactor: fmt.Sprintf("%f", marginFundingFactor),
						},
					},
				},
				MarginCalculator: &vega.MarginCalculator{
					ScalingFactors: &vega.ScalingFactors{
						SearchLevel:       initialMarginScalingFactor * 0.9,
						InitialMargin:     initialMarginScalingFactor,
						CollateralRelease: initialMarginScalingFactor * 1.1,
					},
				},
			},
		},
	}

	rf := entities.RiskFactor{
		Long:  rfLong,
		Short: rfShort,
	}

	assetService := mocks.NewMockAssetService(ctrl)
	marketService := mocks.NewMockMarketsService(ctrl)
	riskFactorService := mocks.NewMockRiskFactorService(ctrl)

	assetService.EXPECT().GetByID(ctx, assetId).Return(asset, nil).AnyTimes()
	marketService.EXPECT().GetByID(ctx, marketId).Return(mkt, nil).AnyTimes()
	riskFactorService.EXPECT().GetMarketRiskFactors(ctx, marketId).Return(rf, nil).AnyTimes()

	testCases := []struct {
		markPrice                         float64
		openVolume                        int64
		avgEntryPrice                     float64
		orders                            []*v2.OrderInfo
		marginAccountBalance              float64
		generalAccountBalance             float64
		orderMarginAccountBalance         float64
		marginMode                        vega.MarginMode
		marginFactor                      float64
		expectedCollIncBest               string
		expectedLiquidationBestVolumeOnly string
	}{
		{
			markPrice:     markPrice,
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      100 * math.Pow10(assetDecimals),
			generalAccountBalance:     1000 * math.Pow10(assetDecimals),
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_CROSS_MARGIN,
		},
		{
			markPrice:     markPrice,
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      100 * math.Pow10(assetDecimals),
			generalAccountBalance:     1000 * math.Pow10(assetDecimals),
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.1,
		},
		{
			markPrice:     markPrice,
			openVolume:    int64(10 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 111.1 * math.Pow10(marketDecimals),
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     1000 * math.Pow10(assetDecimals),
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_CROSS_MARGIN,
		},
		{
			markPrice:     markPrice,
			openVolume:    int64(-10 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 111.1 * math.Pow10(marketDecimals),
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     1000 * math.Pow10(assetDecimals),
			orderMarginAccountBalance: 10 * math.Pow10(assetDecimals),
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.5,
		},
		{
			markPrice:     markPrice,
			openVolume:    int64(-10 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 111.1 * math.Pow10(marketDecimals),
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(11 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(11 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
			},
			marginAccountBalance:      100 * math.Pow10(assetDecimals),
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_CROSS_MARGIN,
		},
		{
			markPrice:     markPrice,
			openVolume:    int64(-10 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 111.1 * math.Pow10(marketDecimals),
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      100 * math.Pow10(assetDecimals),
			generalAccountBalance:     1000 * math.Pow10(assetDecimals),
			orderMarginAccountBalance: 10 * math.Pow10(assetDecimals),
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
		},
		{
			markPrice:     markPrice,
			openVolume:    int64(10 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 111.1 * math.Pow10(marketDecimals),
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(3 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(101, marketDecimals),
					Remaining:     uint64(4 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(105, marketDecimals),
					Remaining:     uint64(5 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(95, marketDecimals),
					Remaining:     uint64(2 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(94, marketDecimals),
					Remaining:     uint64(3 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(90, marketDecimals),
					Remaining:     uint64(10 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
			},
			marginAccountBalance:      100 * math.Pow10(assetDecimals),
			generalAccountBalance:     1000 * math.Pow10(assetDecimals),
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_CROSS_MARGIN,
		},
		{
			markPrice:     markPrice,
			openVolume:    -int64(10 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 111.1 * math.Pow10(marketDecimals),
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(100, marketDecimals),
					Remaining:     uint64(3 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(101, marketDecimals),
					Remaining:     uint64(4 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(105, marketDecimals),
					Remaining:     uint64(5 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(95, marketDecimals),
					Remaining:     uint64(2 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(94, marketDecimals),
					Remaining:     uint64(3 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(90, marketDecimals),
					Remaining:     uint64(10 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
			},
			marginAccountBalance:      100 * math.Pow10(assetDecimals),
			generalAccountBalance:     1000 * math.Pow10(assetDecimals),
			orderMarginAccountBalance: 10 * math.Pow10(assetDecimals),
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.1,
		},
		{
			markPrice:     markPrice,
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideBuy,
					Price:         fmt.Sprintf("%f", markPrice),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:     markPrice,
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideBuy,
					Price:         "0",
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:                 markPrice,
			openVolume:                int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice:             markPrice,
			orders:                    []*v2.OrderInfo{},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:     markPrice,
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         fmt.Sprintf("%f", markPrice),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:     markPrice,
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         "0",
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:                 markPrice,
			openVolume:                -int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice:             markPrice,
			orders:                    []*v2.OrderInfo{},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:     markPrice,
			openVolume:    -int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: markPrice,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideBuy,
					Price:         "0",
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.1,
			expectedCollIncBest:       "0",
		},
		{
			markPrice:     markPrice,
			openVolume:    int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: markPrice,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         "0",
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: true,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.1,
			expectedCollIncBest:       "0",
		},
		{
			markPrice:                         67813,
			openVolume:                        10000,
			avgEntryPrice:                     68113,
			orders:                            []*v2.OrderInfo{},
			marginAccountBalance:              68389,
			generalAccountBalance:             0,
			orderMarginAccountBalance:         0,
			marginMode:                        vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:                      0.01277,
			expectedLiquidationBestVolumeOnly: "6781300000",
		},
	}
	for i, tc := range testCases {
		mktData := entities.MarketData{
			MarkPrice:  num.DecimalFromFloat(tc.markPrice),
			AuctionEnd: auctionEnd,
			ProductData: &entities.ProductData{
				ProductData: &vega.ProductData{
					Data: &vega.ProductData_PerpetualData{
						PerpetualData: &vega.PerpetualData{
							FundingPayment: fmt.Sprintf("%f", fundingPayment),
							FundingRate:    "0.05",
						},
					},
				},
			},
		}
		marketDataService := mocks.NewMockMarketDataService(ctrl)
		marketDataService.EXPECT().GetMarketDataByID(ctx, marketId).Return(mktData, nil).AnyTimes()

		apiService := api.TradingDataServiceV2{
			AssetService:      assetService,
			MarketsService:    marketService,
			MarketDataService: marketDataService,
			RiskFactorService: riskFactorService,
		}

		marginFactor := fmt.Sprintf("%f", tc.marginFactor)
		exclude := false
		dontScale := false
		req := &v2.EstimatePositionRequest{
			MarketId:                  marketId,
			OpenVolume:                tc.openVolume,
			AverageEntryPrice:         fmt.Sprintf("%f", tc.avgEntryPrice),
			Orders:                    tc.orders,
			MarginAccountBalance:      fmt.Sprintf("%f", tc.marginAccountBalance),
			GeneralAccountBalance:     fmt.Sprintf("%f", tc.generalAccountBalance),
			OrderMarginAccountBalance: fmt.Sprintf("%f", tc.orderMarginAccountBalance),
			MarginMode:                tc.marginMode,
			MarginFactor:              &marginFactor,
			IncludeRequiredPositionMarginInAvailableCollateral: &exclude,
			ScaleLiquidationPriceToMarketDecimals:              &dontScale,
		}
		include := true
		req2 := &v2.EstimatePositionRequest{
			MarketId:                  marketId,
			OpenVolume:                tc.openVolume,
			AverageEntryPrice:         fmt.Sprintf("%f", tc.avgEntryPrice),
			Orders:                    tc.orders,
			MarginAccountBalance:      fmt.Sprintf("%f", tc.marginAccountBalance),
			GeneralAccountBalance:     fmt.Sprintf("%f", tc.generalAccountBalance),
			OrderMarginAccountBalance: fmt.Sprintf("%f", tc.orderMarginAccountBalance),
			MarginMode:                tc.marginMode,
			MarginFactor:              &marginFactor,
			IncludeRequiredPositionMarginInAvailableCollateral: &include,
			ScaleLiquidationPriceToMarketDecimals:              &dontScale,
		}

		isolatedMargin := tc.marginMode == vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN

		res, err := apiService.EstimatePosition(ctx, req)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))
		require.NotNil(t, res, fmt.Sprintf("test case #%v", i+1))

		if res.Margin.WorstCase.MaintenanceMargin != "0" {
			require.NotEqual(t, "0", res.Margin.BestCase.InitialMargin, fmt.Sprintf("test case #%v", i+1))
			require.NotEqual(t, "0", res.Margin.WorstCase.InitialMargin, fmt.Sprintf("test case #%v", i+1))
			if isolatedMargin {
				require.Equal(t, "0", res.Margin.BestCase.SearchLevel, fmt.Sprintf("test case #%v", i+1))
				require.Equal(t, "0", res.Margin.BestCase.CollateralReleaseLevel, fmt.Sprintf("test case #%v", i+1))
				require.Equal(t, "0", res.Margin.WorstCase.SearchLevel, fmt.Sprintf("test case #%v", i+1))
				require.Equal(t, "0", res.Margin.WorstCase.CollateralReleaseLevel, fmt.Sprintf("test case #%v", i+1))
			} else {
				require.NotEqual(t, "0", res.Margin.BestCase.SearchLevel, fmt.Sprintf("test case #%v", i+1))
				require.NotEqual(t, "0", res.Margin.BestCase.CollateralReleaseLevel, fmt.Sprintf("test case #%v", i+1))
				require.NotEqual(t, "0", res.Margin.WorstCase.SearchLevel, fmt.Sprintf("test case #%v", i+1))
				require.NotEqual(t, "0", res.Margin.WorstCase.CollateralReleaseLevel, fmt.Sprintf("test case #%v", i+1))
			}
		}

		colIncBest, err := strconv.ParseFloat(res.CollateralIncreaseEstimate.BestCase, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))
		colIncWorst, err := strconv.ParseFloat(res.CollateralIncreaseEstimate.WorstCase, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))
		if tc.expectedCollIncBest != "" {
			require.Equal(t, tc.expectedCollIncBest, res.CollateralIncreaseEstimate.BestCase, fmt.Sprintf("test case #%v", i+1))
		}
		if tc.expectedLiquidationBestVolumeOnly != "" {
			require.Equal(t, tc.expectedLiquidationBestVolumeOnly, res.Liquidation.BestCase.OpenVolumeOnly, fmt.Sprintf("test case #%v", i+1))
		}

		if tc.openVolume == 0 {
			require.Equal(t, colIncBest, colIncWorst, fmt.Sprintf("test case #%v", i+1))
		} else {
			if isolatedMargin {
				require.Equal(t, colIncWorst, colIncBest, fmt.Sprintf("test case #%v", i+1))
			} else {
				require.GreaterOrEqual(t, colIncWorst, colIncBest, fmt.Sprintf("test case #%v", i+1))
			}
		}
		initialMarginBest, err := strconv.ParseFloat(res.Margin.BestCase.InitialMargin, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))
		initialMarginWorst, err := strconv.ParseFloat(res.Margin.WorstCase.InitialMargin, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))

		releaseMarginBest, err := strconv.ParseFloat(res.Margin.BestCase.CollateralReleaseLevel, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))
		releaseMarginWorst, err := strconv.ParseFloat(res.Margin.WorstCase.CollateralReleaseLevel, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))

		expectedCollIncBest := 0.0
		expectedCollIncWorst := 0.0
		expectedPosMarginIncrease := 0.0
		if isolatedMargin {
			priceFactor := math.Pow10(assetDecimals - marketDecimals)
			marketOrderNotional := getMarketOrderNotional(tc.markPrice, tc.orders, priceFactor, positionDecimalPlaces)
			adjNotional := (tc.avgEntryPrice*priceFactor*float64(tc.openVolume)/math.Pow10(positionDecimalPlaces) + marketOrderNotional)

			requiredPositionMargin := math.Abs(adjNotional) * tc.marginFactor
			requiredOrderMargin := getLimitOrderNotional(t, tc.orders, priceFactor, positionDecimalPlaces) * tc.marginFactor
			expectedCollIncBest = requiredPositionMargin + requiredOrderMargin - tc.marginAccountBalance - tc.orderMarginAccountBalance
			expectedCollIncWorst = expectedCollIncBest

			expectedPosMarginIncrease = max(0, requiredPositionMargin-tc.marginAccountBalance)
		} else {
			collat := tc.marginAccountBalance + tc.orderMarginAccountBalance
			bDelta := initialMarginBest - collat
			wDelta := initialMarginWorst - collat
			if bDelta > 0 || collat > releaseMarginBest {
				expectedCollIncBest = bDelta
			}
			if wDelta > 0 || collat > releaseMarginWorst {
				expectedCollIncWorst = wDelta
			}
		}

		actualCollIncBest, err := strconv.ParseFloat(res.CollateralIncreaseEstimate.BestCase, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))
		actualCollIncWorst, err := strconv.ParseFloat(res.CollateralIncreaseEstimate.WorstCase, 64)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))

		require.Equal(t, expectedCollIncBest, actualCollIncBest, fmt.Sprintf("test case #%v", i+1))
		require.Equal(t, expectedCollIncWorst, actualCollIncWorst, fmt.Sprintf("test case #%v", i+1))

		res2, err := apiService.EstimatePosition(ctx, req2)
		require.NoError(t, err, fmt.Sprintf("test case #%v", i+1))
		require.NotNil(t, res2, fmt.Sprintf("test case #%v", i+1))

		if isolatedMargin {
			if expectedPosMarginIncrease > 0 {
				if countOrders(tc.orders, entities.SideBuy) > 0 && res.Liquidation.WorstCase.IncludingBuyOrders != "0" {
					require.NotEqual(t, res.Liquidation.WorstCase.IncludingBuyOrders, res2.Liquidation.WorstCase.IncludingBuyOrders, fmt.Sprintf("test case #%v", i+1))
				}
				if countOrders(tc.orders, entities.SideSell) > 0 && res.Liquidation.WorstCase.IncludingSellOrders != "0" {
					require.NotEqual(t, res.Liquidation.WorstCase.IncludingSellOrders, res2.Liquidation.WorstCase.IncludingSellOrders, fmt.Sprintf("test case #%v", i+1))
				}
				if countOrders(tc.orders, entities.SideBuy) > 0 && res.Liquidation.BestCase.IncludingBuyOrders != "0" {
					require.NotEqual(t, res.Liquidation.BestCase.IncludingBuyOrders, res2.Liquidation.BestCase.IncludingBuyOrders, fmt.Sprintf("test case #%v", i+1))
				}
				if countOrders(tc.orders, entities.SideSell) > 0 && res.Liquidation.BestCase.IncludingSellOrders != "0" {
					require.NotEqual(t, res.Liquidation.BestCase.IncludingSellOrders, res2.Liquidation.BestCase.IncludingSellOrders, fmt.Sprintf("test case #%v", i+1))
				}
			}
		}

		scale := true
		req2.ScaleLiquidationPriceToMarketDecimals = &scale

		res3, err := apiService.EstimatePosition(ctx, req2)
		require.NoError(t, err)
		require.NotNil(t, res3)

		dp := int64(assetDecimals - marketDecimals)
		compareDps(t, res2.Liquidation.BestCase.OpenVolumeOnly, res3.Liquidation.BestCase.OpenVolumeOnly, dp)
		compareDps(t, res2.Liquidation.BestCase.IncludingBuyOrders, res3.Liquidation.BestCase.IncludingBuyOrders, dp)
		compareDps(t, res2.Liquidation.BestCase.IncludingSellOrders, res3.Liquidation.BestCase.IncludingSellOrders, dp)
		compareDps(t, res2.Liquidation.WorstCase.OpenVolumeOnly, res3.Liquidation.WorstCase.OpenVolumeOnly, dp)
		compareDps(t, res2.Liquidation.WorstCase.IncludingBuyOrders, res3.Liquidation.WorstCase.IncludingBuyOrders, dp)
		compareDps(t, res2.Liquidation.WorstCase.IncludingSellOrders, res3.Liquidation.WorstCase.IncludingSellOrders, dp)

		liqFp, err := strconv.ParseFloat(res3.Liquidation.WorstCase.OpenVolumeOnly, 64)
		require.NoError(t, err)
		effectiveOpenVolume := tc.openVolume + sumMarketOrderVolume(tc.orders)
		if tc.openVolume != 0 && effectiveOpenVolume > 0 {
			require.LessOrEqual(t, liqFp, tc.markPrice, fmt.Sprintf("test case #%v", i+1))
		}
		if tc.openVolume != 0 && effectiveOpenVolume < 0 {
			require.GreaterOrEqual(t, liqFp, tc.markPrice, fmt.Sprintf("test case #%v", i+1))
		}
	}
}

//nolint:unparam
func floatToStringWithDp(value float64, dp int) string {
	return fmt.Sprintf("%f", value*math.Pow10(dp))
}

func compareDps(t *testing.T, bigger, smaller string, dp int64) {
	t.Helper()
	b, err := strconv.ParseFloat(bigger, 64)
	require.NoError(t, err)
	s, err := strconv.ParseFloat(smaller, 64)
	require.NoError(t, err)
	if s != 0 {
		l := int64(math.Round(math.Log10(b / s)))
		require.Equal(t, dp, l)
	}
}

func countOrders(orders []*v2.OrderInfo, side vega.Side) int {
	c := 0
	for _, o := range orders {
		if o.Side == side {
			c += 1
		}
	}
	return c
}

func sumMarketOrderVolume(orders []*v2.OrderInfo) int64 {
	v := int64(0)
	for _, o := range orders {
		if !o.IsMarketOrder {
			continue
		}
		if o.Side == entities.SideBuy {
			v += int64(o.Remaining)
		}
		if o.Side == entities.SideSell {
			v -= int64(o.Remaining)
		}
	}
	return v
}

type mockStream struct {
	sent []*httpbody.HttpBody
}

func (s *mockStream) Send(b *httpbody.HttpBody) error { s.sent = append(s.sent, b); return nil }
func (s *mockStream) SetHeader(metadata.MD) error     { return nil }
func (s *mockStream) SendHeader(metadata.MD) error    { return nil }
func (s *mockStream) SetTrailer(metadata.MD)          {}
func (s *mockStream) Context() context.Context        { return context.Background() }
func (s *mockStream) SendMsg(m interface{}) error     { return nil }
func (s *mockStream) RecvMsg(m interface{}) error     { return nil }

func getLimitOrderNotional(t *testing.T, orders []*v2.OrderInfo, priceFactor float64, positionDecimals int) float64 {
	t.Helper()
	notional := 0.0
	for _, o := range orders {
		if o.IsMarketOrder {
			continue
		}
		price, err := strconv.ParseFloat(o.Price, 64)
		require.NoError(t, err)
		notional += price * priceFactor * float64(o.Remaining) / math.Pow10(positionDecimals)
	}
	return notional
}

func getMarketOrderNotional(marketObservable float64, orders []*v2.OrderInfo, priceFactor float64, positionDecimals int) float64 {
	notional := 0.0
	for _, o := range orders {
		if !o.IsMarketOrder {
			continue
		}
		size := float64(o.Remaining) / math.Pow10(positionDecimals)
		if o.Side == vega.Side_SIDE_SELL {
			size = -size
		}
		notional += marketObservable * priceFactor * size
	}
	return notional
}
