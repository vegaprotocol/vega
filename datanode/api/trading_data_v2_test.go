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
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/api/mocks"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/service"
	smocks "code.vegaprotocol.io/vega/datanode/service/mocks"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
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

type dummyReferralService struct{}

func (*dummyReferralService) GetReferralSetStats(ctx context.Context, setID *entities.ReferralSetID, atEpoch *uint64, referee *entities.PartyID, pagination entities.CursorPagination) ([]entities.FlattenReferralSetStats, entities.PageInfo, error) {
	return []entities.FlattenReferralSetStats{
		{
			DiscountFactors: &vega.DiscountFactors{
				MakerDiscountFactor:          "0.001",
				InfrastructureDiscountFactor: "0.002",
				LiquidityDiscountFactor:      "0.003",
			},
		},
	}, entities.PageInfo{}, nil
}

func (*dummyReferralService) ListReferralSets(ctx context.Context, referralSetID *entities.ReferralSetID, referrer, referee *entities.PartyID, pagination entities.CursorPagination) ([]entities.ReferralSet, entities.PageInfo, error) {
	return nil, entities.PageInfo{}, nil
}

func (*dummyReferralService) ListReferralSetReferees(ctx context.Context, referralSetID *entities.ReferralSetID, referrer, referee *entities.PartyID, pagination entities.CursorPagination, aggregationEpochs uint32) ([]entities.ReferralSetRefereeStats, entities.PageInfo, error) {
	return nil, entities.PageInfo{}, nil
}

func TestEstimateFees(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	assetDecimals := 8
	marketDecimals := 3
	positionDecimalPlaces := 2
	marginFundingFactor := 0.95
	initialMarginScalingFactor := 1.5
	linearSlippageFactor := num.DecimalFromFloat(0.005)
	quadraticSlippageFactor := num.DecimalZero()
	rfLong := num.DecimalFromFloat(0.1)
	rfShort := num.DecimalFromFloat(0.2)

	asset := entities.Asset{
		Decimals: assetDecimals,
	}

	tickSize := num.DecimalOne()

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
							SettlementAsset:     crypto.RandomHash(),
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
		TickSize: &tickSize,
		Fees: entities.Fees{
			Factors: &entities.FeeFactors{
				MakerFee:          "0.1",
				InfrastructureFee: "0.02",
				LiquidityFee:      "0.03",
				BuyBackFee:        "0.04",
				TreasuryFee:       "0.05",
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
	vdService := mocks.NewMockVolumeDiscountService(ctrl)
	epochService := mocks.NewMockEpochService(ctrl)

	assetService.EXPECT().GetByID(ctx, gomock.Any()).Return(asset, nil).AnyTimes()
	marketService.EXPECT().GetByID(ctx, gomock.Any()).Return(mkt, nil).AnyTimes()
	riskFactorService.EXPECT().GetMarketRiskFactors(ctx, gomock.Any()).Return(rf, nil).AnyTimes()
	epochService.EXPECT().GetCurrent(gomock.Any()).Return(entities.Epoch{ID: 1}, nil).AnyTimes()
	vdService.EXPECT().Stats(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]entities.FlattenVolumeDiscountStats{
		{
			DiscountFactors: &vega.DiscountFactors{
				MakerDiscountFactor:          "0.0001",
				InfrastructureDiscountFactor: "0.0002",
				LiquidityDiscountFactor:      "0.0003",
			},
		},
	}, entities.PageInfo{}, nil)

	apiService := api.TradingDataServiceV2{
		AssetService:               assetService,
		MarketsService:             marketService,
		RiskFactorService:          riskFactorService,
		VolumeDiscountStatsService: vdService,
		ReferralSetsService:        &dummyReferralService{},
		EpochService:               epochService,
	}

	estimate, err := apiService.EstimateFee(ctx, &v2.EstimateFeeRequest{
		MarketId: crypto.RandomHash(),
		Price:    "100",
		Size:     10,
		Party:    nil,
	})
	require.NoError(t, err)
	// no party was passed so the calculation is returned without discounts
	require.Equal(t, "100000", estimate.Fee.MakerFee)
	require.Equal(t, "20000", estimate.Fee.InfrastructureFee)
	require.Equal(t, "30000", estimate.Fee.LiquidityFee)
	require.Equal(t, "40000", estimate.Fee.BuyBackFee)
	require.Equal(t, "50000", estimate.Fee.TreasuryFee)
	require.Equal(t, "", estimate.Fee.MakerFeeReferrerDiscount)
	require.Equal(t, "", estimate.Fee.MakerFeeVolumeDiscount)
	require.Equal(t, "", estimate.Fee.InfrastructureFeeReferrerDiscount)
	require.Equal(t, "", estimate.Fee.InfrastructureFeeVolumeDiscount)
	require.Equal(t, "", estimate.Fee.LiquidityFeeReferrerDiscount)
	require.Equal(t, "", estimate.Fee.LiquidityFeeVolumeDiscount)

	party := "party"
	estimate, err = apiService.EstimateFee(ctx, &v2.EstimateFeeRequest{
		MarketId: crypto.RandomHash(),
		Price:    "100",
		Size:     10,
		Party:    &party,
	})
	require.NoError(t, err)
	// before discount makerFee = 100000
	// ref discount = 0.001 * 100000 = 100
	// vol discount = 0.0001 * (100000 - 100) = 9.99 => 9
	// 100000 - 100 - 9 = 99,891
	require.Equal(t, "99891", estimate.Fee.MakerFee)
	require.Equal(t, "100", estimate.Fee.MakerFeeReferrerDiscount)
	require.Equal(t, "9", estimate.Fee.MakerFeeVolumeDiscount)

	// before discount infraFee = 20000
	// ref discount = 0.002 * 20000 = 40
	// vol discount = 0.0002 * (20000 - 40) = 3.992 => 3
	// 20000 - 40 - 3 = 19,957
	require.Equal(t, "19957", estimate.Fee.InfrastructureFee)
	require.Equal(t, "40", estimate.Fee.InfrastructureFeeReferrerDiscount)
	require.Equal(t, "3", estimate.Fee.InfrastructureFeeVolumeDiscount)

	// before discount liqFee = 30000
	// ref discount = 0.003 * 30000 = 90
	// vol discount = 0.0003 * (30000 - 90) = 8.973 => 8
	// 30000 - 90 - 8 = 29,902
	require.Equal(t, "29902", estimate.Fee.LiquidityFee)
	require.Equal(t, "90", estimate.Fee.LiquidityFeeReferrerDiscount)
	require.Equal(t, "8", estimate.Fee.LiquidityFeeVolumeDiscount)

	// no discount on buy back and treasury
	require.Equal(t, "40000", estimate.Fee.BuyBackFee)
	require.Equal(t, "50000", estimate.Fee.TreasuryFee)
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

	auctionEnd := int64(0)
	fundingPayment := 1234.56789

	asset := entities.Asset{
		Decimals: assetDecimals,
	}

	tickSize := num.DecimalOne()

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
		TickSize: &tickSize,
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(123.456, marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:                 123.456 * math.Pow10(marketDecimals),
			openVolume:                int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice:             123.456 * math.Pow10(marketDecimals),
			orders:                    []*v2.OrderInfo{},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:     123.456 * math.Pow10(marketDecimals),
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(123.456, marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
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
			markPrice:                 123.456 * math.Pow10(marketDecimals),
			openVolume:                -int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice:             123.456 * math.Pow10(marketDecimals),
			orders:                    []*v2.OrderInfo{},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.3,
			expectedCollIncBest:       "3703680000",
		},
		{
			markPrice:     123.456 * math.Pow10(marketDecimals),
			openVolume:    -int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 123.456 * math.Pow10(marketDecimals),
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
			markPrice:     123.456 * math.Pow10(marketDecimals),
			openVolume:    int64(1 * math.Pow10(positionDecimalPlaces)),
			avgEntryPrice: 123.456 * math.Pow10(marketDecimals),
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
		{
			markPrice:     3225 * math.Pow10(marketDecimals),
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(5000, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 0,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.1,
			expectedCollIncBest:       "50000000000",
		},
		{
			markPrice:     3225 * math.Pow10(marketDecimals),
			openVolume:    0,
			avgEntryPrice: 0,
			orders: []*v2.OrderInfo{
				{
					Side:          entities.SideSell,
					Price:         floatToStringWithDp(5000, marketDecimals),
					Remaining:     uint64(1 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
				{
					Side:          entities.SideBuy,
					Price:         floatToStringWithDp(2500, marketDecimals),
					Remaining:     uint64(2 * math.Pow10(positionDecimalPlaces)),
					IsMarketOrder: false,
				},
			},
			marginAccountBalance:      0,
			generalAccountBalance:     0,
			orderMarginAccountBalance: 50000000000,
			marginMode:                vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
			marginFactor:              0.1,
			expectedCollIncBest:       "0",
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
			adjNotional := tc.avgEntryPrice*priceFactor*float64(tc.openVolume)/math.Pow10(positionDecimalPlaces) + marketOrderNotional

			requiredPositionMargin := math.Abs(adjNotional) * tc.marginFactor
			requiredBuyOrderMargin, requireSellOrderMargin := getLimitOrderNotionalScaledByMarginFactorAndNetOfPosition(t, tc.openVolume, tc.orders, priceFactor, positionDecimalPlaces, tc.marginFactor)
			expectedCollIncBest = requiredPositionMargin + max(requiredBuyOrderMargin, requireSellOrderMargin) - tc.marginAccountBalance - tc.orderMarginAccountBalance
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

func TestListAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	accountStore := smocks.NewMockAccountStore(ctrl)
	balanceStore := smocks.NewMockBalanceStore(ctrl)
	ammSvc := mocks.NewMockAMMService(ctrl)

	apiService := api.TradingDataServiceV2{
		AccountService: service.NewAccount(accountStore, balanceStore, logging.NewTestLogger()),
		AMMPoolService: ammSvc,
	}

	ctx := context.Background()

	req := &v2.ListAccountsRequest{
		Filter: &v2.AccountFilter{
			AssetId:   "asset1",
			PartyIds:  []string{"90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d", "03f49799559c8fd87859edba4b95d40a22e93dedee64f9d7bdc586fa6bbb90e9"},
			MarketIds: []string{"a7878862705cf303cae4ecc9e6cc60781672a9eb5b29eb62bb88b880821340ea", "af56a491ee1dc0576d8bf28e11d936eb744e9976ae0046c2ec824e2beea98ea0"},
		},
	}

	// without derived keys
	{
		expect := []entities.AccountBalance{
			{
				Account: &entities.Account{
					PartyID: "90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d",
					AssetID: "asset1",
				},
			},
			{
				Account: &entities.Account{
					PartyID: "03f49799559c8fd87859edba4b95d40a22e93dedee64f9d7bdc586fa6bbb90e9",
					AssetID: "asset1",
				},
			},
		}

		accountFilter := entities.AccountFilter{
			AssetID:   entities.AssetID(req.Filter.AssetId),
			PartyIDs:  entities.NewPartyIDSlice(req.Filter.PartyIds...),
			MarketIDs: entities.NewMarketIDSlice(req.Filter.MarketIds...),
		}

		// ammSvc.EXPECT().GetSubKeysForParties(gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1).Return(nil, nil)
		accountStore.EXPECT().QueryBalances(gomock.Any(), accountFilter, gomock.Any()).Times(1).Return(expect, entities.PageInfo{}, nil)

		resp, err := apiService.ListAccounts(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Accounts.Edges, 2)
		require.Equal(t, expect[0].ToProto(), resp.Accounts.Edges[0].Node)
		require.Equal(t, expect[1].ToProto(), resp.Accounts.Edges[1].Node)
	}

	// now test with derived keys
	{
		req.IncludeDerivedParties = ptr.From(true)

		expect := []entities.AccountBalance{
			{
				Account: &entities.Account{
					PartyID: "90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d",
					AssetID: "asset1",
				},
			},
			{
				Account: &entities.Account{
					PartyID: "03f49799559c8fd87859edba4b95d40a22e93dedee64f9d7bdc586fa6bbb90e9",
					AssetID: "asset1",
				},
			},
		}

		partyPerDerivedKey := map[string]string{
			"653f9a9850852ca541f20464893536e7986be91c4c364788f6d273fb452778ba": "90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d",
			"79b3aaa5ff0933408cf8f1bcb0b1006cd7bc259d76d400721744e8edc12f2929": "03f49799559c8fd87859edba4b95d40a22e93dedee64f9d7bdc586fa6bbb90e9",
			"35c2dc44b391a5f27ace705b554cd78ba42412c3d2597ceba39642f49ebf5d2b": "90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d",
			"161c1c424215cff4f32154871c225dc9760bcac1d4d6783deeaacf7f8b6861ab": "03f49799559c8fd87859edba4b95d40a22e93dedee64f9d7bdc586fa6bbb90e9",
		}

		ammSvc.EXPECT().GetSubKeysForParties(gomock.Any(), gomock.Any(), gomock.Any()).Times(len(expect)).DoAndReturn(func(_ context.Context, partyIDs []string, _ []string) ([]string, error) {
			if len(partyIDs) == 0 {
				return nil, nil
			}
			ret := make([]string, 0, 2)
			for dk, pid := range partyPerDerivedKey {
				if pid == partyIDs[0] {
					ret = append(ret, dk)
				}
			}
			return ret, nil
		})
		for derivedKey := range partyPerDerivedKey {
			expect = append(expect, entities.AccountBalance{
				Account: &entities.Account{
					PartyID: entities.PartyID(derivedKey),
					AssetID: "asset1",
				},
			})
		}

		accountStore.EXPECT().QueryBalances(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, filter entities.AccountFilter, pageInfo entities.CursorPagination) {
				var expectPartyIDs []string
				for _, e := range expect {
					expectPartyIDs = append(expectPartyIDs, e.PartyID.String())
				}

				var gotPartyIDs []string
				for _, p := range filter.PartyIDs {
					gotPartyIDs = append(gotPartyIDs, p.String())
				}

				slices.Sort(expectPartyIDs)
				slices.Sort(gotPartyIDs)
				require.Zero(t, slices.Compare(expectPartyIDs, gotPartyIDs))
			}).Times(1).Return(expect, entities.PageInfo{}, nil)

		resp, err := apiService.ListAccounts(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Accounts.Edges, 6)

		for i := range expect {
			require.Equal(t, expect[i].ToProto().Owner, resp.Accounts.Edges[i].Node.Owner)

			if party, ok := partyPerDerivedKey[expect[i].PartyID.String()]; ok {
				require.NotNil(t, resp.Accounts.Edges[i].Node.ParentPartyId)
				require.Equal(t, party, *resp.Accounts.Edges[i].Node.ParentPartyId)
			}
		}
	}
}

func TestObserveAccountBalances(t *testing.T) {
	ctrl := gomock.NewController(t)
	accountStore := smocks.NewMockAccountStore(ctrl)
	balanceStore := smocks.NewMockBalanceStore(ctrl)
	ammSvc := mocks.NewMockAMMService(ctrl)

	apiService := api.TradingDataServiceV2{
		AccountService: service.NewAccount(accountStore, balanceStore, logging.NewTestLogger()),
		AMMPoolService: ammSvc,
	}

	apiService.SetLogger(logging.NewTestLogger())

	ctx := context.Background()

	req := &v2.ObserveAccountsRequest{
		MarketId: "a7878862705cf303cae4ecc9e6cc60781672a9eb5b29eb62bb88b880821340ea",
		PartyId:  "90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d",
		Asset:    "asset1",
	}

	// without derived keys
	{
		expect := []entities.AccountBalance{
			{
				Account: &entities.Account{
					PartyID:  entities.PartyID(req.PartyId),
					AssetID:  entities.AssetID(req.Asset),
					MarketID: entities.MarketID(req.MarketId),
					Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
				},
			},
			{
				Account: &entities.Account{
					PartyID:  entities.PartyID(req.PartyId),
					AssetID:  entities.AssetID(req.Asset),
					MarketID: entities.MarketID(req.MarketId),
					Type:     vega.AccountType_ACCOUNT_TYPE_MARGIN,
				},
			},
			{
				Account: &entities.Account{
					PartyID:  entities.PartyID(req.PartyId),
					AssetID:  entities.AssetID(req.Asset),
					MarketID: entities.MarketID(req.MarketId),
					Type:     vega.AccountType_ACCOUNT_TYPE_BOND,
				},
			},
		}

		balanceStore.EXPECT().Flush(gomock.Any()).Return(expect[1:], nil).Times(1)

		accountFilter := entities.AccountFilter{
			AssetID:   "asset1",
			PartyIDs:  entities.NewPartyIDSlice(req.PartyId),
			MarketIDs: entities.NewMarketIDSlice(req.MarketId),
		}

		accountStore.EXPECT().QueryBalances(gomock.Any(), accountFilter, gomock.Any()).Times(1).Return(expect[:1], entities.PageInfo{}, nil)

		srvCtx, cancel := context.WithCancel(ctx)
		res := mockObserveAccountServer{
			mockServerStream: mockServerStream{ctx: srvCtx},
			send: func(oar *v2.ObserveAccountsResponse) error {
				switch res := oar.Response.(type) {
				case *v2.ObserveAccountsResponse_Snapshot:
					require.Len(t, res.Snapshot.Accounts, 1)
					require.Equal(t, expect[0].ToProto(), res.Snapshot.Accounts[0])
				case *v2.ObserveAccountsResponse_Updates:
					require.Equal(t, len(expect[1:]), len(res.Updates.Accounts))
					require.Equal(t, expect[1].ToProto().Owner, res.Updates.Accounts[0].Owner)
					require.Equal(t, expect[1].ToProto().Type, res.Updates.Accounts[0].Type)
					require.Equal(t, expect[2].ToProto().Owner, res.Updates.Accounts[1].Owner)
					require.Equal(t, expect[2].ToProto().Type, res.Updates.Accounts[1].Type)
					cancel()
				default:
					t.Fatalf("unexpected response type: %T", oar.Response)
				}
				return nil
			},
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			apiService.ObserveAccounts(req, res)
		}()

		time.Sleep(1 * time.Second)
		err := apiService.AccountService.Flush(ctx)
		require.NoError(t, err)
		wg.Wait()
	}

	// now test with derived keys
	{
		req.IncludeDerivedParties = ptr.From(true)

		expect := []entities.AccountBalance{
			{
				Account: &entities.Account{
					PartyID:  entities.PartyID(req.PartyId),
					AssetID:  entities.AssetID(req.Asset),
					MarketID: entities.MarketID(req.MarketId),
					Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
				},
			},
			{
				Account: &entities.Account{
					PartyID:  entities.PartyID(req.PartyId),
					AssetID:  entities.AssetID(req.Asset),
					MarketID: entities.MarketID(req.MarketId),
					Type:     vega.AccountType_ACCOUNT_TYPE_MARGIN,
				},
			},
			{
				Account: &entities.Account{
					PartyID:  entities.PartyID(req.PartyId),
					AssetID:  entities.AssetID(req.Asset),
					MarketID: entities.MarketID(req.MarketId),
					Type:     vega.AccountType_ACCOUNT_TYPE_BOND,
				},
			},
		}

		partyPerDerivedKey := map[string]string{
			"653f9a9850852ca541f20464893536e7986be91c4c364788f6d273fb452778ba": req.PartyId,
		}
		ammSvc.EXPECT().GetSubKeysForParties(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(maps.Keys(partyPerDerivedKey), nil)

		for derivedKey := range partyPerDerivedKey {
			expect = append(expect, entities.AccountBalance{
				Account: &entities.Account{
					PartyID:  entities.PartyID(derivedKey),
					AssetID:  entities.AssetID(req.Asset),
					MarketID: entities.MarketID(req.MarketId),
					Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
				},
			})
		}
		balanceStore.EXPECT().Flush(gomock.Any()).Return(expect[3:], nil).Times(1)

		accountFilter := entities.AccountFilter{
			AssetID:   "asset1",
			PartyIDs:  entities.NewPartyIDSlice(append(maps.Keys(partyPerDerivedKey), req.PartyId)...),
			MarketIDs: entities.NewMarketIDSlice(req.MarketId),
		}

		accountStore.EXPECT().QueryBalances(gomock.Any(), accountFilter, gomock.Any()).Times(1).Return(expect[:3], entities.PageInfo{}, nil)

		srvCtx, cancel := context.WithCancel(ctx)
		res := mockObserveAccountServer{
			mockServerStream: mockServerStream{ctx: srvCtx},
			send: func(oar *v2.ObserveAccountsResponse) error {
				switch res := oar.Response.(type) {
				case *v2.ObserveAccountsResponse_Snapshot:
					require.Len(t, res.Snapshot.Accounts, 3)
					require.Equal(t, expect[0].ToProto(), res.Snapshot.Accounts[0])
					require.Equal(t, expect[1].ToProto(), res.Snapshot.Accounts[1])
					require.Equal(t, expect[2].ToProto(), res.Snapshot.Accounts[2])
				case *v2.ObserveAccountsResponse_Updates:
					require.Equal(t, len(expect[3:]), len(res.Updates.Accounts))
					require.Equal(t, expect[3].ToProto().Owner, res.Updates.Accounts[0].Owner)
					require.Equal(t, expect[3].ToProto().Type, res.Updates.Accounts[0].Type)
					cancel()
				default:
					t.Fatalf("unexpected response type: %T", oar.Response)
				}
				return nil
			},
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			apiService.ObserveAccounts(req, res)
		}()

		time.Sleep(1 * time.Second)
		err := apiService.AccountService.Flush(ctx)
		require.NoError(t, err)
		wg.Wait()
	}
}

func TestListRewards(t *testing.T) {
	ctrl := gomock.NewController(t)
	marketStore := smocks.NewMockMarketStore(ctrl)
	rewardStore := smocks.NewMockRewardStore(ctrl)
	ammSvc := mocks.NewMockAMMService(ctrl)

	apiService := api.TradingDataServiceV2{
		MarketsService: service.NewMarkets(marketStore),
		RewardService:  service.NewReward(rewardStore, logging.NewTestLogger()),
		AMMPoolService: ammSvc,
	}

	ctx := context.Background()

	req := &v2.ListRewardsRequest{
		PartyId: "90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d",
	}

	// without derived keys
	{
		expect := []entities.Reward{
			{
				PartyID: entities.PartyID(req.PartyId),
			},
		}

		pagination := entities.DefaultCursorPagination(true)

		rewardStore.EXPECT().GetByCursor(ctx,
			[]string{req.PartyId}, req.AssetId, req.FromEpoch, req.ToEpoch, pagination, req.TeamId, req.GameId, req.MarketId).
			Times(1).Return(expect, entities.PageInfo{}, nil)

		resp, err := apiService.ListRewards(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Rewards.Edges, 1)
		require.Equal(t, expect[0].ToProto().PartyId, resp.Rewards.Edges[0].Node.PartyId)
	}

	// now test with derived keys
	{
		req.IncludeDerivedParties = ptr.From(true)

		expect := []entities.Reward{
			{
				PartyID: entities.PartyID(req.PartyId),
			},
			{
				PartyID: entities.PartyID("653f9a9850852ca541f20464893536e7986be91c4c364788f6d273fb452778ba"),
			},
			{
				PartyID: entities.PartyID("35c2dc44b391a5f27ace705b554cd78ba42412c3d2597ceba39642f49ebf5d2b"),
			},
		}

		pagination := entities.DefaultCursorPagination(true)

		ammSvc.EXPECT().GetSubKeysForParties(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]string{
			"653f9a9850852ca541f20464893536e7986be91c4c364788f6d273fb452778ba",
			"35c2dc44b391a5f27ace705b554cd78ba42412c3d2597ceba39642f49ebf5d2b",
		}, nil)

		rewardStore.EXPECT().GetByCursor(ctx, gomock.Any(), req.AssetId, req.FromEpoch, req.ToEpoch,
			pagination, req.TeamId, req.GameId, req.MarketId).
			Do(func(_ context.Context, gotPartyIDs []string, _ *string, _, _ *uint64, _ entities.CursorPagination, _, _, _ *string) {
				expectPartyIDs := []string{expect[0].PartyID.String(), expect[1].PartyID.String(), expect[2].PartyID.String()}

				slices.Sort(expectPartyIDs)
				slices.Sort(gotPartyIDs)
				require.Zero(t, slices.Compare(expectPartyIDs, gotPartyIDs))
			}).
			Times(1).Return(expect, entities.PageInfo{}, nil)

		resp, err := apiService.ListRewards(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Rewards.Edges, 3)
		require.Equal(t, expect[0].ToProto().PartyId, resp.Rewards.Edges[0].Node.PartyId)
		require.Equal(t, expect[1].ToProto().PartyId, resp.Rewards.Edges[1].Node.PartyId)
		require.Equal(t, expect[2].ToProto().PartyId, resp.Rewards.Edges[2].Node.PartyId)
	}
}

func TestListRewardSummaries(t *testing.T) {
	ctrl := gomock.NewController(t)
	marketStore := smocks.NewMockMarketStore(ctrl)
	rewardStore := smocks.NewMockRewardStore(ctrl)
	ammSvc := mocks.NewMockAMMService(ctrl)

	apiService := api.TradingDataServiceV2{
		MarketsService: service.NewMarkets(marketStore),
		RewardService:  service.NewReward(rewardStore, logging.NewTestLogger()),
		AMMPoolService: ammSvc,
	}

	ctx := context.Background()

	t.Run("without party id", func(t *testing.T) {
		req := &v2.ListRewardSummariesRequest{
			AssetId: ptr.From("asset1"),
		}

		expect := []entities.RewardSummary{
			{
				PartyID: entities.PartyID("random-party"),
				AssetID: entities.AssetID(*req.AssetId),
				Amount:  num.NewDecimalFromFloat(200),
			},
		}

		rewardStore.EXPECT().GetSummaries(ctx, []string{}, req.AssetId).
			Times(1).Return(expect, nil)

		resp, err := apiService.ListRewardSummaries(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Summaries, 1)
		require.Equal(t, expect[0].ToProto(), resp.Summaries[0])
	})

	t.Run("with derived keys without party", func(t *testing.T) {
		req := &v2.ListRewardSummariesRequest{
			AssetId:               ptr.From("asset1"),
			IncludeDerivedParties: ptr.From(true),
		}

		expect := []entities.RewardSummary{
			{
				PartyID: entities.PartyID("random-party"),
				AssetID: entities.AssetID(*req.AssetId),
				Amount:  num.NewDecimalFromFloat(200),
			},
		}

		ammSvc.EXPECT().GetSubKeysForParties(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
		rewardStore.EXPECT().GetSummaries(ctx, []string{}, req.AssetId).
			Times(1).Return(expect, nil)

		resp, err := apiService.ListRewardSummaries(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Summaries, 1)
		require.Equal(t, expect[0].ToProto(), resp.Summaries[0])
	})

	t.Run("without derived keys with party", func(t *testing.T) {
		req := &v2.ListRewardSummariesRequest{
			PartyId: ptr.From("90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d"),
			AssetId: ptr.From("asset1"),
		}

		expect := []entities.RewardSummary{
			{
				PartyID: entities.PartyID(*req.PartyId),
				AssetID: entities.AssetID(*req.AssetId),
				Amount:  num.NewDecimalFromFloat(200),
			},
		}

		rewardStore.EXPECT().GetSummaries(ctx, []string{*req.PartyId}, req.AssetId).
			Times(1).Return(expect, nil)

		resp, err := apiService.ListRewardSummaries(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Summaries, 1)
		require.Equal(t, expect[0].ToProto(), resp.Summaries[0])
	})

	t.Run("with derived keys and party", func(t *testing.T) {
		req := &v2.ListRewardSummariesRequest{
			PartyId:               ptr.From("90421f905ab72919671caca4ffb891ba8b253a4d506e1c0223745268edf4416d"),
			AssetId:               ptr.From("asset1"),
			IncludeDerivedParties: ptr.From(true),
		}

		expect := []entities.RewardSummary{
			{
				PartyID: entities.PartyID(*req.PartyId),
				AssetID: entities.AssetID(*req.AssetId),
				Amount:  num.NewDecimalFromFloat(200),
			},
			{
				PartyID: entities.PartyID("653f9a9850852ca541f20464893536e7986be91c4c364788f6d273fb452778ba"),
				AssetID: entities.AssetID(*req.AssetId),
				Amount:  num.NewDecimalFromFloat(150),
			},
			{
				PartyID: entities.PartyID("35c2dc44b391a5f27ace705b554cd78ba42412c3d2597ceba39642f49ebf5d2b"),
				AssetID: entities.AssetID(*req.AssetId),
				Amount:  num.NewDecimalFromFloat(130),
			},
		}

		ammSvc.EXPECT().GetSubKeysForParties(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]string{
			"653f9a9850852ca541f20464893536e7986be91c4c364788f6d273fb452778ba",
			"35c2dc44b391a5f27ace705b554cd78ba42412c3d2597ceba39642f49ebf5d2b",
		}, nil)

		rewardStore.EXPECT().GetSummaries(ctx, gomock.Any(), req.AssetId).
			Do(func(_ context.Context, gotPartyIDs []string, _ *string) {
				var expectPartyIDs []string
				for _, e := range expect {
					expectPartyIDs = append(expectPartyIDs, e.PartyID.String())
				}

				slices.Sort(expectPartyIDs)
				slices.Sort(gotPartyIDs)
				require.Zero(t, slices.Compare(expectPartyIDs, gotPartyIDs))
			}).Times(1).Return(expect, nil)

		resp, err := apiService.ListRewardSummaries(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Summaries, 3)
		require.Equal(t, expect[0].ToProto(), resp.Summaries[0])
		require.Equal(t, expect[1].ToProto(), resp.Summaries[1])
		require.Equal(t, expect[2].ToProto(), resp.Summaries[2])
	})
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

func getLimitOrderNotionalScaledByMarginFactorAndNetOfPosition(t *testing.T, positionSize int64, orders []*v2.OrderInfo, priceFactor float64, positionDecimals int, marginFactor float64) (float64, float64) {
	t.Helper()
	buyNotional, sellNotional := 0.0, 0.0
	buyOrders, sellOrders := make([]*v2.OrderInfo, 0), make([]*v2.OrderInfo, 0)
	for _, o := range orders {
		if o.Side == entities.SideBuy {
			if o.IsMarketOrder {
				positionSize += int64(o.Remaining)
				continue
			}
			buyOrders = append(buyOrders, o)
		}
		if o.Side == entities.SideSell {
			if o.IsMarketOrder {
				positionSize -= int64(o.Remaining)
				continue
			}
			sellOrders = append(sellOrders, o)
		}
	}

	// sort orders from best to worst
	sort.Slice(buyOrders, func(i, j int) bool {
		price_i, err := strconv.ParseFloat(buyOrders[i].Price, 64)
		require.NoError(t, err)
		price_j, err := strconv.ParseFloat(buyOrders[j].Price, 64)
		require.NoError(t, err)

		return price_i > price_j
	})
	sort.Slice(sellOrders, func(i, j int) bool {
		price_i, err := strconv.ParseFloat(sellOrders[i].Price, 64)
		require.NoError(t, err)
		price_j, err := strconv.ParseFloat(sellOrders[j].Price, 64)
		require.NoError(t, err)

		return price_i < price_j
	})

	remainingCovered := uint64(math.Abs(float64(positionSize)))
	for _, o := range buyOrders {
		size := o.Remaining
		if remainingCovered != 0 && (positionSize < 0) {
			if size >= remainingCovered { // part of the order doesn't require margin
				size = size - remainingCovered
				remainingCovered = 0
			} else { // the entire order doesn't require margin
				remainingCovered -= size
				size = 0
			}
		}
		if size > 0 {
			price, err := strconv.ParseFloat(o.Price, 64)
			require.NoError(t, err)
			buyNotional += price * priceFactor * float64(size) / math.Pow10(positionDecimals)
		}
	}

	remainingCovered = uint64(math.Abs(float64(positionSize)))
	for _, o := range sellOrders {
		size := o.Remaining
		if remainingCovered != 0 && (positionSize > 0) {
			if size >= remainingCovered { // part of the order doesn't require margin
				size = size - remainingCovered
				remainingCovered = 0
			} else { // the entire order doesn't require margin
				remainingCovered -= size
				size = 0
			}
		}
		if size > 0 {
			price, err := strconv.ParseFloat(o.Price, 64)
			require.NoError(t, err)
			sellNotional += price * priceFactor * float64(size) / math.Pow10(positionDecimals)
		}
	}

	return buyNotional * marginFactor, sellNotional * marginFactor
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

type mockObserveAccountServer struct {
	mockServerStream
	send func(*v2.ObserveAccountsResponse) error
}

func (m mockObserveAccountServer) Send(resp *v2.ObserveAccountsResponse) error {
	if m.send != nil {
		return m.send(resp)
	}
	return nil
}

type mockServerStream struct {
	ctx        context.Context
	recvMsg    func(m interface{}) error
	sendMsg    func(m interface{}) error
	setHeader  func(md metadata.MD) error
	sendHeader func(md metadata.MD) error
	setTrailer func(md metadata.MD)
}

func (m mockServerStream) Context() context.Context {
	return m.ctx
}

func (m mockServerStream) SendMsg(msg interface{}) error {
	if m.sendMsg != nil {
		return m.sendMsg(msg)
	}
	return nil
}

func (m mockServerStream) RecvMsg(msg interface{}) error {
	if m.recvMsg != nil {
		return m.recvMsg(msg)
	}
	return nil
}

func (m mockServerStream) SetHeader(md metadata.MD) error {
	if m.setHeader != nil {
		return m.setHeader(md)
	}
	return nil
}

func (m mockServerStream) SendHeader(md metadata.MD) error {
	if m.sendHeader != nil {
		return m.sendHeader(md)
	}
	return nil
}

func (m mockServerStream) SetTrailer(md metadata.MD) {
	if m.setTrailer != nil {
		m.setTrailer(md)
	}
}
