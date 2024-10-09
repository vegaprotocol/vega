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
	"context"
	"testing"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMarkets_Add(t *testing.T) {
	t.Run("Add should insert a valid market record", shouldInsertAValidMarketRecord)
	t.Run("Add should update a valid market record if the block number already exists", shouldUpdateAValidMarketRecord)
	t.Run("Add should insert a valid spot market record", shouldInsertAValidSpotMarketRecord)
	t.Run("Add should insert a valid perpetual market record", shouldInsertAValidPerpetualMarketRecord)
}

func TestMarkets_Get(t *testing.T) {
	t.Run("GetByID should return the request market if it exists", getByIDShouldReturnTheRequestedMarketIfItExists)
	t.Run("GetByID should return error if the market does not exist", getByIDShouldReturnErrorIfTheMarketDoesNotExist)
	t.Run("GetAllPaged should not include rejected markets", getAllPagedShouldNotIncludeRejectedMarkets)
	t.Run("GetByTxHash", getByTxHashReturnsMatchingMarkets)
	t.Run("GetByID should return a spot market if it exists", getByIDShouldReturnASpotMarketIfItExists)
	t.Run("GetByID should return a perpetual market if it exists", getByIDShouldReturnAPerpetualMarketIfItExists)
}

func TestGetAllFees(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)

	market := entities.Market{
		ID:       "deadbeef",
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
		State:    entities.MarketStateActive,
		Fees: entities.Fees{
			Factors: &entities.FeeFactors{
				MakerFee:          "0.1",
				InfrastructureFee: "0.2",
				LiquidityFee:      "0.3",
				BuyBackFee:        "0.4",
				TreasuryFee:       "0.5",
			},
		},
	}
	err := md.Upsert(ctx, &market)
	require.NoError(t, err)

	market2 := entities.Market{
		ID:       "beefdead",
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
		State:    entities.MarketStateActive,
		Fees: entities.Fees{
			Factors: &entities.FeeFactors{
				MakerFee:          "0.5",
				InfrastructureFee: "0.4",
				LiquidityFee:      "0.3",
				BuyBackFee:        "0.2",
				TreasuryFee:       "0.1",
			},
		},
	}
	err = md.Upsert(ctx, &market2)
	require.NoError(t, err, "Saving market entity to database")

	mkts, err := md.GetAllFees(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(mkts))

	require.Equal(t, "beefdead", mkts[0].ID.String())
	require.Equal(t, "0.5", mkts[0].Fees.Factors.MakerFee)
	require.Equal(t, "0.4", mkts[0].Fees.Factors.InfrastructureFee)
	require.Equal(t, "0.3", mkts[0].Fees.Factors.LiquidityFee)
	require.Equal(t, "0.2", mkts[0].Fees.Factors.BuyBackFee)
	require.Equal(t, "0.1", mkts[0].Fees.Factors.TreasuryFee)
	require.Equal(t, "deadbeef", mkts[1].ID.String())
	require.Equal(t, "0.1", mkts[1].Fees.Factors.MakerFee)
	require.Equal(t, "0.2", mkts[1].Fees.Factors.InfrastructureFee)
	require.Equal(t, "0.3", mkts[1].Fees.Factors.LiquidityFee)
	require.Equal(t, "0.4", mkts[1].Fees.Factors.BuyBackFee)
	require.Equal(t, "0.5", mkts[1].Fees.Factors.TreasuryFee)
}

func getByIDShouldReturnTheRequestedMarketIfItExists(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)

	market := entities.Market{
		ID:       "deadbeef",
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
		State:    entities.MarketStateActive,
	}
	err := md.Upsert(ctx, &market)
	require.NoError(t, err, "Saving market entity to database")

	marketFromDB, err := md.GetByID(ctx, market.ID.String())
	require.NoError(t, err)
	assert.Equal(t, market.ID, marketFromDB.ID)
	assert.Equal(t, market.TxHash, marketFromDB.TxHash)
	assert.Equal(t, market.VegaTime, marketFromDB.VegaTime)
	assert.Equal(t, market.State, marketFromDB.State)
}

func getByIDShouldReturnASpotMarketIfItExists(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)

	marketProto := getTestSpotMarket()

	market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")
	err = md.Upsert(ctx, market)
	require.NoError(t, err, "Saving market entity to database")

	marketFromDB, err := md.GetByID(ctx, market.ID.String())
	require.NoError(t, err)
	marketToProto := marketFromDB.ToProto()
	assert.IsType(t, &vega.Spot{}, marketToProto.TradableInstrument.GetInstrument().GetSpot())
}

func getByIDShouldReturnAPerpetualMarketIfItExists(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)

	marketProto := getTestSpotMarket()

	market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")
	err = md.Upsert(ctx, market)
	require.NoError(t, err, "Saving market entity to database")

	marketFromDB, err := md.GetByID(ctx, market.ID.String())
	require.NoError(t, err)
	marketToProto := marketFromDB.ToProto()
	assert.IsType(t, &vega.Perpetual{}, marketToProto.TradableInstrument.GetInstrument().GetPerpetual())
}

func getByTxHashReturnsMatchingMarkets(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)

	market := entities.Market{
		ID:       "deadbeef",
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
		State:    entities.MarketStateActive,
	}
	err := md.Upsert(ctx, &market)
	require.NoError(t, err, "Saving market entity to database")

	foundMarkets, err := md.GetByTxHash(ctx, market.TxHash)
	require.NoError(t, err)
	require.Len(t, foundMarkets, 1)
	assert.Equal(t, market.ID, foundMarkets[0].ID)
	assert.Equal(t, market.TxHash, foundMarkets[0].TxHash)
	assert.Equal(t, market.VegaTime, foundMarkets[0].VegaTime)
	assert.Equal(t, market.State, foundMarkets[0].State)
}

func getByIDShouldReturnErrorIfTheMarketDoesNotExist(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)

	market := entities.Market{
		ID:       "deadbeef",
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
		State:    entities.MarketStateActive,
	}
	err := md.Upsert(ctx, &market)
	require.NoError(t, err, "Saving market entity to database")

	_, err = md.GetByID(ctx, "not-a-market")
	require.Error(t, err)
}

func getAllPagedShouldNotIncludeRejectedMarkets(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)

	market := entities.Market{
		ID:       "deadbeef",
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
		State:    entities.MarketStateActive,
	}
	err := md.Upsert(ctx, &market)
	require.NoError(t, err, "Saving market entity to database")

	rejected := entities.Market{
		ID:       "DEADBAAD",
		TxHash:   generateTxHash(),
		VegaTime: block.VegaTime,
		State:    entities.MarketStateRejected,
	}
	err = md.Upsert(ctx, &rejected)
	require.NoError(t, err, "Saving market entity to database")

	markets, pageInfo, err := md.GetAllPaged(ctx, "", entities.CursorPagination{}, true)
	require.NoError(t, err)
	assert.Len(t, markets, 1)
	assert.Equal(t, market.ID, markets[0].ID)
	assert.Equal(t, market.TxHash, markets[0].TxHash)
	assert.Equal(t, market.VegaTime, markets[0].VegaTime)
	assert.Equal(t, market.State, markets[0].State)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     market.Cursor().Encode(),
		EndCursor:       market.Cursor().Encode(),
	}, pageInfo)
}

func shouldInsertAValidMarketRecord(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	var rowCount int

	err := connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)

	marketProto := getTestFutureMarket(true)

	market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = md.Upsert(ctx, market)
	require.NoError(t, err, "Saving market entity to database")
	err = connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func setupMarketsTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Markets) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	md := sqlstore.NewMarkets(connectionSource)
	return bs, md
}

func shouldUpdateAValidMarketRecord(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	var rowCount int

	t.Run("should have no markets in the database", func(t *testing.T) {
		err := connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
		require.NoError(t, err)
		assert.Equal(t, 0, rowCount)
	})

	var block entities.Block
	var marketProto *vega.Market

	t.Run("should insert a valid market record to the database with liquidation strategy", func(t *testing.T) {
		block = addTestBlock(t, ctx, bs)
		marketProto = getTestFutureMarketWithLiquidationStrategy(false)

		market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
		require.NoError(t, err, "Converting market proto to database entity")

		err = md.Upsert(ctx, market)
		require.NoError(t, err, "Saving market entity to database")

		var got entities.Market
		err = pgxscan.Get(ctx, connectionSource, &got, `select * from markets where id = $1 and vega_time = $2`, market.ID, market.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)
		assert.NotNil(t, got.LiquidationStrategy)

		assert.Equal(t, marketProto.TradableInstrument, got.TradableInstrument.ToProto())
		assert.Equal(t, marketProto.LiquidationStrategy, got.LiquidationStrategy.IntoProto())
	})

	t.Run("should insert a valid market record to the database", func(t *testing.T) {
		block = addTestBlock(t, ctx, bs)
		marketProto = getTestFutureMarket(false)

		market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
		require.NoError(t, err, "Converting market proto to database entity")

		err = md.Upsert(ctx, market)
		require.NoError(t, err, "Saving market entity to database")

		var got entities.Market
		err = pgxscan.Get(ctx, connectionSource, &got, `select * from markets where id = $1 and vega_time = $2`, market.ID, market.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)

		assert.Equal(t, marketProto.TradableInstrument, got.TradableInstrument.ToProto())
	})

	marketProto.TradableInstrument.Instrument.Name = "Updated Test Instrument"
	marketProto.TradableInstrument.Instrument.Metadata.Tags = append(marketProto.TradableInstrument.Instrument.Metadata.Tags, "CCC")

	t.Run("should update a valid market record to the database if the block number already exists", func(t *testing.T) {
		market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)

		require.NoError(t, err, "Converting market proto to database entity")

		err = md.Upsert(ctx, market)
		require.NoError(t, err, "Saving market entity to database")

		var got entities.Market
		err = pgxscan.Get(ctx, connectionSource, &got, `select * from markets where id = $1 and vega_time = $2`, market.ID, market.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)

		assert.Equal(t, marketProto.TradableInstrument, got.TradableInstrument.ToProto())
	})

	t.Run("should add the updated market record to the database if the block number has changed", func(t *testing.T) {
		newMarketProto := proto.Clone(marketProto).(*vega.Market)
		newMarketProto.TradableInstrument.Instrument.Metadata.Tags = append(newMarketProto.TradableInstrument.Instrument.Metadata.Tags, "DDD")
		newBlock := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Second))

		market, err := entities.NewMarketFromProto(newMarketProto, generateTxHash(), newBlock.VegaTime)
		require.NoError(t, err, "Converting market proto to database entity")

		err = md.Upsert(ctx, market)
		require.NoError(t, err, "Saving market entity to database")

		err = connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
		require.NoError(t, err)
		assert.Equal(t, 3, rowCount)

		var gotFirstBlock, gotSecondBlock entities.Market

		err = pgxscan.Get(ctx, connectionSource, &gotFirstBlock, `select * from markets where id = $1 and vega_time = $2`, market.ID, block.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)

		assert.Equal(t, marketProto.TradableInstrument, gotFirstBlock.TradableInstrument.ToProto())

		err = pgxscan.Get(ctx, connectionSource, &gotSecondBlock, `select * from markets where id = $1 and vega_time = $2`, market.ID, newBlock.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)

		assert.Equal(t, newMarketProto.TradableInstrument, gotSecondBlock.TradableInstrument.ToProto())
	})
}

func shouldInsertAValidSpotMarketRecord(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	var rowCount int

	err := connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)

	marketProto := getTestSpotMarket()

	market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = md.Upsert(ctx, market)
	require.NoError(t, err, "Saving market entity to database")
	err = connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func shouldInsertAValidPerpetualMarketRecord(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx := tempTransaction(t)

	var rowCount int

	err := connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)

	marketProto := getTestPerpetualMarket()

	market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = md.Upsert(ctx, market)
	require.NoError(t, err, "Saving market entity to database")
	err = connectionSource.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func getTestSpotMarket() *vega.Market {
	mkt := getTestMarket()

	mkt.TradableInstrument.Instrument.Product = &vega.Instrument_Spot{
		Spot: &vega.Spot{
			BaseAsset:  "Ethereum",
			QuoteAsset: "USD",
		},
	}

	return mkt
}

func getTestPerpetualMarket() *vega.Market {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	mkt := getTestMarket()
	mkt.TradableInstrument.Instrument.Product = &vega.Instrument_Perpetual{
		Perpetual: &vega.Perpetual{
			SettlementAsset:     "Ethereum/Ether",
			QuoteName:           "ETH-230929",
			MarginFundingFactor: "0.5",
			InterestRate:        "0.012",
			ClampLowerBound:     "0.2",
			ClampUpperBound:     "0.8",
			DataSourceSpecForSettlementSchedule: &vega.DataSourceSpec{
				Id:        "test-settlement-schedule",
				CreatedAt: time.Now().UnixNano(),
				UpdatedAt: time.Now().UnixNano(),
				Data: vega.NewDataSourceDefinition(
					vega.DataSourceContentTypeOracle,
				).SetOracleConfig(
					&vega.DataSourceDefinitionExternal_Oracle{
						Oracle: &vega.DataSourceSpecConfiguration{
							Signers: []*v1.Signer{pk.IntoProto()},
							Filters: []*v1.Filter{
								{
									Key: &v1.PropertyKey{
										Name: "prices.ETH.value",
										Type: v1.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*v1.Condition{},
								},
							},
						},
					},
				),
				Status: vega.DataSourceSpec_STATUS_ACTIVE,
			},
			DataSourceSpecForSettlementData: &vega.DataSourceSpec{
				Id:        "test-settlement-data",
				CreatedAt: time.Now().UnixNano(),
				UpdatedAt: time.Now().UnixNano(),
				Data: vega.NewDataSourceDefinition(
					vega.DataSourceContentTypeOracle,
				).SetOracleConfig(
					&vega.DataSourceDefinitionExternal_Oracle{
						Oracle: &vega.DataSourceSpecConfiguration{
							Signers: []*v1.Signer{pk.IntoProto()},
							Filters: []*v1.Filter{
								{
									Key: &v1.PropertyKey{
										Name: "prices.ETH.value",
										Type: v1.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*v1.Condition{},
								},
							},
						},
					},
				),
				Status: vega.DataSourceSpec_STATUS_ACTIVE,
			},
			DataSourceSpecBinding: &vega.DataSourceSpecToPerpetualBinding{
				SettlementDataProperty:     "prices.ETH.value",
				SettlementScheduleProperty: "2023-09-29T00:00:00.000000000Z",
			},
		},
	}
	return mkt
}

func getTestMarket() *vega.Market {
	return &vega.Market{
		Id: GenerateID(),
		TradableInstrument: &vega.TradableInstrument{
			Instrument: &vega.Instrument{
				Id:   "Crypto/BTCUSD/Futures/Dec19",
				Code: "FX:BTCUSD/DEC19",
				Name: "December 2019 BTC vs USD future",
				Metadata: &vega.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
			},
			MarginCalculator: &vega.MarginCalculator{
				ScalingFactors: &vega.ScalingFactors{
					SearchLevel:       1.1,
					InitialMargin:     1.2,
					CollateralRelease: 1.4,
				},
			},
			RiskModel: &vega.TradableInstrument_LogNormalRiskModel{
				LogNormalRiskModel: &vega.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   1.0 / 365.25 / 24,
					Params: &vega.LogNormalModelParams{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
		},
		Fees: &vega.Fees{
			Factors: &vega.FeeFactors{
				MakerFee:          "",
				InfrastructureFee: "",
				LiquidityFee:      "",
			},
			LiquidityFeeSettings: &vega.LiquidityFeeSettings{
				Method: vega.LiquidityFeeSettings_METHOD_MARGINAL_COST,
			},
		},
		OpeningAuction: &vega.AuctionDuration{
			Duration: 0,
			Volume:   0,
		},
		PriceMonitoringSettings: &vega.PriceMonitoringSettings{
			Parameters: &vega.PriceMonitoringParameters{
				Triggers: []*vega.PriceMonitoringTrigger{
					{
						Horizon:          0,
						Probability:      "",
						AuctionExtension: 0,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &vega.LiquidityMonitoringParameters{
			TargetStakeParameters: &vega.TargetStakeParameters{
				TimeWindow:    0,
				ScalingFactor: 0,
			},
		},
		TradingMode: vega.Market_TRADING_MODE_CONTINUOUS,
		State:       vega.Market_STATE_ACTIVE,
		MarketTimestamps: &vega.MarketTimestamps{
			Proposed: 0,
			Pending:  0,
			Open:     0,
			Close:    0,
		},
		PositionDecimalPlaces:   8,
		LpPriceRange:            "0.95",
		LinearSlippageFactor:    "1.23",
		QuadraticSlippageFactor: "5.67",
	}
}

func getTestFutureMarketWithLiquidationStrategy(termInt bool) *vega.Market {
	mkt := getTestFutureMarket(termInt)
	mkt.LiquidationStrategy = &vega.LiquidationStrategy{
		DisposalTimeStep:      10,
		DisposalFraction:      "0.1",
		FullDisposalSize:      20,
		MaxFractionConsumed:   "0.01",
		DisposalSlippageRange: "0.1",
	}
	return mkt
}

func getTestFutureMarket(termInt bool) *vega.Market {
	term := &vega.DataSourceSpec{
		Id:        "",
		CreatedAt: 0,
		UpdatedAt: 0,
		Data: vega.NewDataSourceDefinition(
			vega.DataSourceContentTypeOracle,
		).SetOracleConfig(
			&vega.DataSourceDefinitionExternal_Oracle{
				Oracle: &vega.DataSourceSpecConfiguration{
					Signers: nil,
					Filters: nil,
				},
			},
		),
		Status: 0,
	}

	if termInt {
		term = &vega.DataSourceSpec{
			Id:        "",
			CreatedAt: 0,
			UpdatedAt: 0,
			Data: vega.NewDataSourceDefinition(
				vega.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*v1.Condition{
					{
						Operator: v1.Condition_OPERATOR_GREATER_THAN,
						Value:    "test-value",
					},
				},
			),
			Status: 0,
		}
	}

	return &vega.Market{
		Id: "DEADBEEF",
		TradableInstrument: &vega.TradableInstrument{
			Instrument: &vega.Instrument{
				Id:   "TEST_INSTRUMENT",
				Code: "TEST",
				Name: "Test Instrument",
				Metadata: &vega.InstrumentMetadata{
					Tags: []string{"AAA", "BBB"},
				},
				Product: &vega.Instrument_Future{
					Future: &vega.Future{
						SettlementAsset: "Test Asset",
						QuoteName:       "Test Quote",
						DataSourceSpecForSettlementData: &vega.DataSourceSpec{
							Id:        "",
							CreatedAt: 0,
							UpdatedAt: 0,
							Data: vega.NewDataSourceDefinition(
								vega.DataSourceContentTypeOracle,
							).SetOracleConfig(
								&vega.DataSourceDefinitionExternal_Oracle{
									Oracle: &vega.DataSourceSpecConfiguration{
										Signers: nil,
										Filters: nil,
									},
								},
							),
							Status: 0,
						},
						DataSourceSpecForTradingTermination: term,
						DataSourceSpecBinding: &vega.DataSourceSpecToFutureBinding{
							SettlementDataProperty:     "",
							TradingTerminationProperty: "",
						},
					},
				},
			},
			MarginCalculator: &vega.MarginCalculator{
				ScalingFactors: &vega.ScalingFactors{
					SearchLevel:       0,
					InitialMargin:     0,
					CollateralRelease: 0,
				},
			},
			RiskModel: &vega.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vega.SimpleRiskModel{
					Params: &vega.SimpleModelParams{
						FactorLong:           0,
						FactorShort:          0,
						MaxMoveUp:            0,
						MinMoveDown:          0,
						ProbabilityOfTrading: 0,
					},
				},
			},
		},
		DecimalPlaces: 16,
		Fees: &vega.Fees{
			Factors: &vega.FeeFactors{
				MakerFee:          "",
				InfrastructureFee: "",
				LiquidityFee:      "",
			},
			LiquidityFeeSettings: &vega.LiquidityFeeSettings{
				Method: vega.LiquidityFeeSettings_METHOD_MARGINAL_COST,
			},
		},
		OpeningAuction: &vega.AuctionDuration{
			Duration: 0,
			Volume:   0,
		},
		PriceMonitoringSettings: &vega.PriceMonitoringSettings{
			Parameters: &vega.PriceMonitoringParameters{
				Triggers: []*vega.PriceMonitoringTrigger{
					{
						Horizon:          0,
						Probability:      "",
						AuctionExtension: 0,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &vega.LiquidityMonitoringParameters{
			TargetStakeParameters: &vega.TargetStakeParameters{
				TimeWindow:    0,
				ScalingFactor: 0,
			},
		},
		TradingMode: vega.Market_TRADING_MODE_CONTINUOUS,
		State:       vega.Market_STATE_ACTIVE,
		MarketTimestamps: &vega.MarketTimestamps{
			Proposed: 0,
			Pending:  0,
			Open:     0,
			Close:    0,
		},
		PositionDecimalPlaces:   8,
		LpPriceRange:            "0.95",
		LinearSlippageFactor:    "1.23",
		QuadraticSlippageFactor: "5.67",
		LiquiditySlaParams: &vega.LiquiditySLAParameters{
			PriceRange:                  "0.75",
			CommitmentMinTimeFraction:   "0.5",
			PerformanceHysteresisEpochs: 0,
			SlaCompetitionFactor:        "1.0",
		},
	}
}

func populateTestMarkets(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, md *sqlstore.Markets, blockTimes map[string]time.Time) {
	t.Helper()

	markets := []entities.Market{
		{
			ID:           entities.MarketID("02a16077"),
			InstrumentID: "AAA",
		},
		{
			ID:           entities.MarketID("44eea1bc"),
			InstrumentID: "BBB",
		},
		{
			ID:           entities.MarketID("65be62cd"),
			InstrumentID: "CCC",
		},
		{
			ID:           entities.MarketID("7a797e0e"),
			InstrumentID: "DDD",
		},
		{
			ID:           entities.MarketID("7bb2356e"),
			InstrumentID: "EEE",
		},
		{
			ID:           entities.MarketID("b7c84b8e"),
			InstrumentID: "FFF",
		},
		{
			ID:           entities.MarketID("c612300d"),
			InstrumentID: "GGG",
		},
		{
			ID:           entities.MarketID("c8744329"),
			InstrumentID: "HHH",
		},
		{
			ID:           entities.MarketID("da8d1803"),
			InstrumentID: "III",
		},
		{
			ID:           entities.MarketID("fb1528a5"),
			InstrumentID: "JJJ",
		},
	}

	source := &testBlockSource{bs, time.Now()}
	for _, market := range markets {
		block := source.getNextBlock(t, ctx)
		market.VegaTime = block.VegaTime
		blockTimes[market.ID.String()] = block.VegaTime
		err := md.Upsert(ctx, &market)
		require.NoError(t, err)
	}
}

func TestMarketsCursorPagination(t *testing.T) {
	t.Run("Should return the market if Market ID is provided", testCursorPaginationReturnsTheSpecifiedMarket)
	t.Run("Should return all markets if no market ID and no cursor is provided", testCursorPaginationReturnsAllMarkets)
	t.Run("Should return the first page when first limit is provided with no after cursor", testCursorPaginationReturnsFirstPage)
	t.Run("Should return the last page when last limit is provided with first before cursor", testCursorPaginationReturnsLastPage)
	t.Run("Should return the page specified by the first limit and after cursor", testCursorPaginationReturnsPageTraversingForward)
	t.Run("Should return the page specified by the last limit and before cursor", testCursorPaginationReturnsPageTraversingBackward)

	t.Run("Should return the market if Market ID is provided - newest first", testCursorPaginationReturnsTheSpecifiedMarketNewestFirst)
	t.Run("Should return all markets if no market ID and no cursor is provided - newest first", testCursorPaginationReturnsAllMarketsNewestFirst)
	t.Run("Should return the first page when first limit is provided with no after cursor - newest first", testCursorPaginationReturnsFirstPageNewestFirst)
	t.Run("Should return the last page when last limit is provided with first before cursor - newest first", testCursorPaginationReturnsLastPageNewestFirst)
	t.Run("Should return the page specified by the first limit and after cursor - newest first", testCursorPaginationReturnsPageTraversingForwardNewestFirst)
	t.Run("Should return the page specified by the last limit and before cursor - newest first", testCursorPaginationReturnsPageTraversingBackwardNewestFirst)
}

func testCursorPaginationReturnsTheSpecifiedMarket(t *testing.T) {
	ctx := tempTransaction(t)

	bs, md := setupMarketsTest(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "c612300d", pagination, true)
	require.NoError(t, err)
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "c612300d", got[0].ID.String())
	assert.Equal(t, "GGG", got[0].InstrumentID)

	mc := entities.MarketCursor{
		VegaTime: blockTimes["c612300d"],
		ID:       "c612300d",
	}

	wantStartCursor := entities.NewCursor(mc.String()).Encode()
	wantEndCursor := entities.NewCursor(mc.String()).Encode()

	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsAllMarkets(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)
	assert.Equal(t, 10, len(got))
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "fb1528a5", got[9].ID.String())
	assert.Equal(t, "AAA", got[0].InstrumentID)
	assert.Equal(t, "JJJ", got[9].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["02a16077"],
			ID:       "02a16077",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["fb1528a5"],
			ID:       "fb1528a5",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsFirstPage(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "02a16077", got[0].ID.String())
	assert.Equal(t, "65be62cd", got[2].ID.String())
	assert.Equal(t, "AAA", got[0].InstrumentID)
	assert.Equal(t, "CCC", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["02a16077"],
			ID:       "02a16077",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["65be62cd"],
			ID:       "65be62cd",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsLastPage(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "c8744329", got[0].ID.String())
	assert.Equal(t, "fb1528a5", got[2].ID.String())
	assert.Equal(t, "HHH", got[0].InstrumentID)
	assert.Equal(t, "JJJ", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c8744329"],
			ID:       "c8744329",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["fb1528a5"],
			ID:       "fb1528a5",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsPageTraversingForward(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	first := int32(3)
	after := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["65be62cd"],
			ID:       "65be62cd",
		}.String(),
	).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "7a797e0e", got[0].ID.String())
	assert.Equal(t, "b7c84b8e", got[2].ID.String())
	assert.Equal(t, "DDD", got[0].InstrumentID)
	assert.Equal(t, "FFF", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["7a797e0e"],
			ID:       "7a797e0e",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["b7c84b8e"],
			ID:       "b7c84b8e",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsPageTraversingBackward(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	last := int32(3)
	before := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c8744329"],
			ID:       "c8744329",
		}.String(),
	).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "7bb2356e", got[0].ID.String())
	assert.Equal(t, "c612300d", got[2].ID.String())
	assert.Equal(t, "EEE", got[0].InstrumentID)
	assert.Equal(t, "GGG", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["7bb2356e"],
			ID:       "7bb2356e",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c612300d"],
			ID:       "c612300d",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsTheSpecifiedMarketNewestFirst(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "c612300d", pagination, true)
	require.NoError(t, err)
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "c612300d", got[0].ID.String())
	assert.Equal(t, "GGG", got[0].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c612300d"],
			ID:       "c612300d",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c612300d"],
			ID:       "c612300d",
		}.String(),
	).Encode()

	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsAllMarketsNewestFirst(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)
	assert.Equal(t, 10, len(got))
	assert.Equal(t, "fb1528a5", got[0].ID.String())
	assert.Equal(t, "02a16077", got[9].ID.String())
	assert.Equal(t, "JJJ", got[0].InstrumentID)
	assert.Equal(t, "AAA", got[9].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["fb1528a5"],
			ID:       "fb1528a5",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["02a16077"],
			ID:       "02a16077",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsFirstPageNewestFirst(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "fb1528a5", got[0].ID.String())
	assert.Equal(t, "c8744329", got[2].ID.String())
	assert.Equal(t, "JJJ", got[0].InstrumentID)
	assert.Equal(t, "HHH", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["fb1528a5"],
			ID:       "fb1528a5",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c8744329"],
			ID:       "c8744329",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsLastPageNewestFirst(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "65be62cd", got[0].ID.String())
	assert.Equal(t, "02a16077", got[2].ID.String())
	assert.Equal(t, "CCC", got[0].InstrumentID)
	assert.Equal(t, "AAA", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["65be62cd"],
			ID:       "65be62cd",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["02a16077"],
			ID:       "02a16077",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsPageTraversingForwardNewestFirst(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	first := int32(3)
	after := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c8744329"],
			ID:       "c8744329",
		}.String(),
	).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "c612300d", got[0].ID.String())
	assert.Equal(t, "7bb2356e", got[2].ID.String())
	assert.Equal(t, "GGG", got[0].InstrumentID)
	assert.Equal(t, "EEE", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["c612300d"],
			ID:       "c612300d",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["7bb2356e"],
			ID:       "7bb2356e",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func testCursorPaginationReturnsPageTraversingBackwardNewestFirst(t *testing.T) {
	bs, md := setupMarketsTest(t)
	ctx := tempTransaction(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	last := int32(3)
	before := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["65be62cd"],
			ID:       "65be62cd",
		}.String(),
	).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination, true)
	require.NoError(t, err)

	assert.Equal(t, 3, len(got))
	assert.Equal(t, "b7c84b8e", got[0].ID.String())
	assert.Equal(t, "7a797e0e", got[2].ID.String())
	assert.Equal(t, "FFF", got[0].InstrumentID)
	assert.Equal(t, "DDD", got[2].InstrumentID)

	wantStartCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["b7c84b8e"],
			ID:       "b7c84b8e",
		}.String(),
	).Encode()
	wantEndCursor := entities.NewCursor(
		entities.MarketCursor{
			VegaTime: blockTimes["7a797e0e"],
			ID:       "7a797e0e",
		}.String(),
	).Encode()
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     wantStartCursor,
		EndCursor:       wantEndCursor,
	}, pageInfo)
}

func TestSuccessorMarkets(t *testing.T) {
	t.Run("should create a market lineage record when a successor market proposal is approved", testMarketLineageCreated)
	t.Run("ListSuccessorMarkets should return the market lineage", testListSuccessorMarkets)
	t.Run("GetMarket should return the market with its parent and successor if they exist", testGetMarketWithParentAndSuccessor)
}

func testMarketLineageCreated(t *testing.T) {
	ctx := tempTransaction(t)

	bs, md := setupMarketsTest(t)
	parentMarket := entities.Market{
		ID:           entities.MarketID("deadbeef01"),
		InstrumentID: "deadbeef01",
	}

	successorMarketA := entities.Market{
		ID:             entities.MarketID("deadbeef02"),
		InstrumentID:   "deadbeef02",
		ParentMarketID: parentMarket.ID,
	}

	successorMarketB := entities.Market{
		ID:             entities.MarketID("deadbeef03"),
		InstrumentID:   "deadbeef03",
		ParentMarketID: successorMarketA.ID,
	}

	var rowCount int64

	source := &testBlockSource{bs, time.Now()}
	block := source.getNextBlock(t, ctx)
	t.Run("parent market should create a market lineage record with no parent market id", func(t *testing.T) {
		parentMarket.VegaTime = block.VegaTime
		parentMarket.State = entities.MarketStateProposed
		err := md.Upsert(ctx, &parentMarket)
		require.NoError(t, err)
		err = connectionSource.QueryRow(ctx, `select count(*) from market_lineage where market_id = $1`, parentMarket.ID).Scan(&rowCount)
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowCount)

		block = source.getNextBlock(t, ctx)
		parentMarket.State = entities.MarketStatePending
		parentMarket.TradingMode = entities.MarketTradingModeOpeningAuction
		parentMarket.VegaTime = block.VegaTime
		err = md.Upsert(ctx, &parentMarket)
		require.NoError(t, err)

		block = source.getNextBlock(t, ctx)
		parentMarket.State = entities.MarketStateActive
		parentMarket.TradingMode = entities.MarketTradingModeContinuous
		parentMarket.VegaTime = block.VegaTime
		err = md.Upsert(ctx, &parentMarket)
		require.NoError(t, err)

		var marketID, parentMarketID, rootID entities.MarketID
		err = connectionSource.QueryRow(ctx,
			`select market_id, parent_market_id, root_id from market_lineage where market_id = $1`,
			parentMarket.ID,
		).Scan(&marketID, &parentMarketID, &rootID)
		require.NoError(t, err)
		assert.Equal(t, parentMarket.ID, marketID)
		assert.Equal(t, entities.MarketID(""), parentMarketID)
		assert.Equal(t, parentMarket.ID, rootID)
	})

	block = source.getNextBlock(t, ctx)
	t.Run("successor market should create a market lineage record pointing to the parent market and the root market", func(t *testing.T) {
		successorMarketA.VegaTime = block.VegaTime
		successorMarketA.State = entities.MarketStateProposed
		err := md.Upsert(ctx, &successorMarketA)
		require.NoError(t, err)
		// proposed market successor only, so it should not create a lineage record yet
		err = connectionSource.QueryRow(ctx, `select count(*) from market_lineage where market_id = $1`, successorMarketA.ID).Scan(&rowCount)
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowCount)

		block = source.getNextBlock(t, ctx)
		successorMarketA.State = entities.MarketStatePending
		successorMarketA.TradingMode = entities.MarketTradingModeOpeningAuction
		successorMarketA.VegaTime = block.VegaTime
		err = md.Upsert(ctx, &successorMarketA)
		require.NoError(t, err)

		block = source.getNextBlock(t, ctx)
		successorMarketA.State = entities.MarketStateActive
		successorMarketA.TradingMode = entities.MarketTradingModeContinuous
		successorMarketA.VegaTime = block.VegaTime
		err = md.Upsert(ctx, &successorMarketA)
		require.NoError(t, err)
		// proposed market successor has been accepted and is pending, so we should now have a lineage record pointing to the parent
		var marketID, parentMarketID, rootID entities.MarketID
		err = connectionSource.QueryRow(ctx,
			`select market_id, parent_market_id, root_id from market_lineage where market_id = $1`,
			successorMarketA.ID,
		).Scan(&marketID, &parentMarketID, &rootID)
		require.NoError(t, err)
		assert.Equal(t, successorMarketA.ID, marketID)
		assert.Equal(t, parentMarket.ID, parentMarketID)
		assert.Equal(t, parentMarket.ID, rootID)
	})

	block = source.getNextBlock(t, ctx)
	t.Run("second successor market should create a lineage record pointing to the parent market and the root market", func(t *testing.T) {
		successorMarketB.VegaTime = block.VegaTime
		successorMarketB.State = entities.MarketStateProposed
		err := md.Upsert(ctx, &successorMarketB)
		require.NoError(t, err)
		// proposed market successor only, so it should not create a lineage record yet
		err = connectionSource.QueryRow(ctx, `select count(*) from market_lineage where market_id = $1`, successorMarketB.ID).Scan(&rowCount)
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowCount)

		block = source.getNextBlock(t, ctx)
		successorMarketB.State = entities.MarketStatePending
		successorMarketB.TradingMode = entities.MarketTradingModeOpeningAuction
		successorMarketB.VegaTime = block.VegaTime
		err = md.Upsert(ctx, &successorMarketB)
		require.NoError(t, err)
		// proposed market successor has been accepted and is pending, so we should now have a lineage record pointing to the parent
		block = source.getNextBlock(t, ctx)
		successorMarketB.State = entities.MarketStateActive
		successorMarketB.TradingMode = entities.MarketTradingModeContinuous
		successorMarketB.VegaTime = block.VegaTime
		err = md.Upsert(ctx, &successorMarketB)
		require.NoError(t, err)
		var marketID, parentMarketID, rootID entities.MarketID
		err = connectionSource.QueryRow(ctx,
			`select market_id, parent_market_id, root_id from market_lineage where market_id = $1`,
			successorMarketB.ID,
		).Scan(&marketID, &parentMarketID, &rootID)
		require.NoError(t, err)
		assert.Equal(t, successorMarketB.ID, marketID)
		assert.Equal(t, successorMarketA.ID, parentMarketID)
		assert.Equal(t, parentMarket.ID, rootID)
	})
}

func testListSuccessorMarkets(t *testing.T) {
	ctx := tempTransaction(t)

	md, markets, proposals := setupSuccessorMarkets(t, ctx)

	successors := []entities.SuccessorMarket{
		{
			Market: markets[5],
			Proposals: []*entities.Proposal{
				&proposals[1],
				&proposals[2],
			},
		},
		{
			Market: markets[6],
			Proposals: []*entities.Proposal{
				&proposals[3],
				&proposals[4],
			},
		},
		{
			Market: markets[9],
		},
	}

	t.Run("should list the full history if children only is false", func(t *testing.T) {
		got, _, err := md.ListSuccessorMarkets(ctx, "deadbeef02", true, entities.CursorPagination{})
		require.NoError(t, err)
		want := successors[:]
		assert.Equal(t, want, got)
	})

	t.Run("should list only the successor markets if children only is true", func(t *testing.T) {
		got, _, err := md.ListSuccessorMarkets(ctx, "deadbeef02", false, entities.CursorPagination{})
		require.NoError(t, err)
		want := successors[1:]

		assert.Equal(t, want, got)
	})

	t.Run("should paginate results if pagination is provided", func(t *testing.T) {
		first := int32(2)
		after := entities.NewCursor(
			entities.MarketCursor{
				VegaTime: markets[5].VegaTime,
				ID:       markets[5].ID,
			}.String(),
		).Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		got, pageInfo, err := md.ListSuccessorMarkets(ctx, "deadbeef01", true, pagination)
		require.NoError(t, err)
		want := successors[1:]

		assert.Equal(t, want, got, "paged successor markets do not match")
		wantStartCursor := entities.NewCursor(
			entities.MarketCursor{
				VegaTime: markets[6].VegaTime,
				ID:       markets[6].ID,
			}.String(),
		).Encode()
		wantEndCursor := entities.NewCursor(
			entities.MarketCursor{
				VegaTime: markets[9].VegaTime,
				ID:       markets[9].ID,
			}.String(),
		).Encode()
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     wantStartCursor,
			EndCursor:       wantEndCursor,
		}, pageInfo)
	})

	t.Run("should list the parent market even if it has not entered continuous trading and has no successors", func(t *testing.T) {
		got, _, err := md.ListSuccessorMarkets(ctx, "deadbeef04", false, entities.CursorPagination{})
		require.NoError(t, err)
		want := []entities.SuccessorMarket{
			{
				Market: markets[10],
			},
		}
		assert.Equal(t, want, got)
	})
}

func testGetMarketWithParentAndSuccessor(t *testing.T) {
	ctx := tempTransaction(t)

	md, _, _ := setupSuccessorMarkets(t, ctx)

	t.Run("should return successor market id only if the first market in a succession line", func(t *testing.T) {
		got, err := md.GetByID(ctx, "deadbeef01")
		require.NoError(t, err)
		assert.Equal(t, "", got.ParentMarketID.String())
		assert.Equal(t, "deadbeef02", got.SuccessorMarketID.String())
	})

	t.Run("should return parent and successor market id if the market is within a succession line", func(t *testing.T) {
		got, err := md.GetByID(ctx, "deadbeef02")
		require.NoError(t, err)
		assert.Equal(t, "deadbeef01", got.ParentMarketID.String())
		assert.Equal(t, "deadbeef03", got.SuccessorMarketID.String())
	})

	t.Run("should return parent market id only if the last market in a succession line", func(t *testing.T) {
		got, err := md.GetByID(ctx, "deadbeef03")
		require.NoError(t, err)
		assert.Equal(t, "deadbeef02", got.ParentMarketID.String())
		assert.Equal(t, "", got.SuccessorMarketID.String())
	})
}

func setupSuccessorMarkets(t *testing.T, ctx context.Context) (*sqlstore.Markets, []entities.Market, []entities.Proposal) {
	t.Helper()

	bs, md := setupMarketsTest(t)
	ps := sqlstore.NewProposals(connectionSource)
	ts := sqlstore.NewParties(connectionSource)

	emptyLS := &vega.LiquidationStrategy{
		DisposalTimeStep:      0,
		DisposalFraction:      "0",
		FullDisposalSize:      0,
		MaxFractionConsumed:   "0",
		DisposalSlippageRange: "0",
	}
	liquidationStrat := entities.LiquidationStrategyFromProto(emptyLS)
	parentMarket := entities.Market{
		ID:           entities.MarketID("deadbeef01"),
		InstrumentID: "deadbeef01",
		TradableInstrument: entities.TradableInstrument{
			TradableInstrument: &vega.TradableInstrument{},
		},
		LiquiditySLAParameters: entities.LiquiditySLAParameters{
			PriceRange:                  num.NewDecimalFromFloat(0),
			CommitmentMinTimeFraction:   num.NewDecimalFromFloat(0),
			PerformanceHysteresisEpochs: 0,
			SlaCompetitionFactor:        num.NewDecimalFromFloat(0),
		},
		LiquidationStrategy: liquidationStrat,
	}

	successorMarketA := entities.Market{
		ID:           entities.MarketID("deadbeef02"),
		InstrumentID: "deadbeef02",
		TradableInstrument: entities.TradableInstrument{
			TradableInstrument: &vega.TradableInstrument{},
		},
		ParentMarketID: parentMarket.ID,
		LiquiditySLAParameters: entities.LiquiditySLAParameters{
			PriceRange:                  num.NewDecimalFromFloat(0),
			CommitmentMinTimeFraction:   num.NewDecimalFromFloat(0),
			PerformanceHysteresisEpochs: 0,
			SlaCompetitionFactor:        num.NewDecimalFromFloat(0),
		},
		LiquidationStrategy: liquidationStrat,
	}

	parentMarket.SuccessorMarketID = successorMarketA.ID

	successorMarketB := entities.Market{
		ID:           entities.MarketID("deadbeef03"),
		InstrumentID: "deadbeef03",
		TradableInstrument: entities.TradableInstrument{
			TradableInstrument: &vega.TradableInstrument{},
		},
		ParentMarketID: successorMarketA.ID,
		LiquiditySLAParameters: entities.LiquiditySLAParameters{
			PriceRange:                  num.NewDecimalFromFloat(0),
			CommitmentMinTimeFraction:   num.NewDecimalFromFloat(0),
			PerformanceHysteresisEpochs: 0,
			SlaCompetitionFactor:        num.NewDecimalFromFloat(0),
		},
		LiquidationStrategy: liquidationStrat,
	}

	parentMarket2 := entities.Market{
		ID:           entities.MarketID("deadbeef04"),
		InstrumentID: "deadbeef04",
		TradableInstrument: entities.TradableInstrument{
			TradableInstrument: &vega.TradableInstrument{},
		},
		LiquiditySLAParameters: entities.LiquiditySLAParameters{
			PriceRange:                  num.NewDecimalFromFloat(0),
			CommitmentMinTimeFraction:   num.NewDecimalFromFloat(0),
			PerformanceHysteresisEpochs: 0,
			SlaCompetitionFactor:        num.NewDecimalFromFloat(0),
		},
		LiquidationStrategy: liquidationStrat,
	}

	successorMarketA.SuccessorMarketID = successorMarketB.ID

	source := &testBlockSource{bs, time.Now()}

	block := source.getNextBlock(t, ctx)

	pt1 := addTestParty(t, ctx, ts, block)
	pt2 := addTestParty(t, ctx, ts, block)

	proposals := []struct {
		id        string
		party     entities.Party
		reference string
		block     entities.Block
		state     entities.ProposalState
		rationale entities.ProposalRationale
		terms     entities.ProposalTerms
		reason    entities.ProposalError
	}{
		{
			id:        "deadbeef01",
			party:     pt1,
			reference: "deadbeef01",
			block:     source.getNextBlock(t, ctx),
			state:     entities.ProposalStateEnacted,
			rationale: entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "mydescription1"}},
			terms: entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{
				Changes: &vega.NewMarketConfiguration{
					LiquidationStrategy: emptyLS,
				},
			}}}},
			reason: entities.ProposalErrorUnspecified,
		},
		{
			id:        "deadbeef02",
			party:     pt1,
			reference: "deadbeef02",
			block:     source.getNextBlock(t, ctx),
			state:     entities.ProposalStateEnacted,
			rationale: entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "mydescription1"}},
			terms: entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{
				Changes: &vega.NewMarketConfiguration{
					Successor: &vega.SuccessorConfiguration{
						ParentMarketId:        "deadbeef01",
						InsurancePoolFraction: "1.0",
					},
					LiquidationStrategy: emptyLS,
				},
			}}}},
			reason: entities.ProposalErrorUnspecified,
		},
		{
			id:        "deadbeefaa",
			party:     pt2,
			reference: "deadbeefaa",
			block:     source.getNextBlock(t, ctx),
			state:     entities.ProposalStateEnacted,
			rationale: entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "mydescription1"}},
			terms: entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{
				Changes: &vega.NewMarketConfiguration{
					Successor: &vega.SuccessorConfiguration{
						ParentMarketId:        "deadbeef01",
						InsurancePoolFraction: "1.0",
					},
					LiquidationStrategy: emptyLS,
				},
			}}}},
			reason: entities.ProposalErrorParticipationThresholdNotReached,
		},
		{
			id:        "deadbeef03",
			party:     pt1,
			reference: "deadbeef03",
			block:     source.getNextBlock(t, ctx),
			state:     entities.ProposalStateEnacted,
			rationale: entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "mydescription1"}},
			terms: entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{
				Changes: &vega.NewMarketConfiguration{
					Successor: &vega.SuccessorConfiguration{
						ParentMarketId:        "deadbeef02",
						InsurancePoolFraction: "1.0",
					},
					LiquidationStrategy: emptyLS,
				},
			}}}},
			reason: entities.ProposalErrorUnspecified,
		},
		{
			id:        "deadbeefbb",
			party:     pt2,
			reference: "deadbeefbb",
			block:     source.getNextBlock(t, ctx),
			state:     entities.ProposalStateEnacted,
			rationale: entities.ProposalRationale{ProposalRationale: &vega.ProposalRationale{Title: "myurl1.com", Description: "mydescription1"}},
			terms: entities.ProposalTerms{ProposalTerms: &vega.ProposalTerms{Change: &vega.ProposalTerms_NewMarket{NewMarket: &vega.NewMarket{
				Changes: &vega.NewMarketConfiguration{
					Successor: &vega.SuccessorConfiguration{
						ParentMarketId:        "deadbeef02",
						InsurancePoolFraction: "1.0",
					},
					LiquidationStrategy: emptyLS,
				},
			}}}},
			reason: entities.ProposalErrorParticipationThresholdNotReached,
		},
	}

	props := []entities.Proposal{}
	for _, p := range proposals {
		p := addTestProposal(t, ctx, ps, p.id, p.party, p.reference, p.block, p.state,
			p.rationale, p.terms, p.reason, nil, entities.BatchProposalTerms{})

		props = append(props, p)
	}

	markets := []struct {
		market      entities.Market
		state       entities.MarketState
		tradingMode entities.MarketTradingMode
	}{
		{
			market:      parentMarket,
			state:       entities.MarketStateProposed,
			tradingMode: entities.MarketTradingModeOpeningAuction,
		},
		{
			market:      parentMarket,
			state:       entities.MarketStatePending,
			tradingMode: entities.MarketTradingModeOpeningAuction,
		},
		{
			market:      parentMarket,
			state:       entities.MarketStateActive,
			tradingMode: entities.MarketTradingModeContinuous,
		},
		{
			market:      successorMarketA,
			state:       entities.MarketStateProposed,
			tradingMode: entities.MarketTradingModeOpeningAuction,
		},
		{
			market:      successorMarketA,
			state:       entities.MarketStatePending,
			tradingMode: entities.MarketTradingModeOpeningAuction,
		},
		{
			market:      parentMarket,
			state:       entities.MarketStateSettled,
			tradingMode: entities.MarketTradingModeNoTrading,
		},
		{
			market:      successorMarketA,
			state:       entities.MarketStateActive,
			tradingMode: entities.MarketTradingModeContinuous,
		},
		{
			market:      successorMarketB,
			state:       entities.MarketStateProposed,
			tradingMode: entities.MarketTradingModeOpeningAuction,
		},
		{
			market:      successorMarketB,
			state:       entities.MarketStatePending,
			tradingMode: entities.MarketTradingModeOpeningAuction,
		},
		{
			market:      successorMarketB,
			state:       entities.MarketStateActive,
			tradingMode: entities.MarketTradingModeContinuous,
		},
		{
			market:      parentMarket2,
			state:       entities.MarketStatePending,
			tradingMode: entities.MarketTradingModeOpeningAuction,
		},
	}

	entries := make([]entities.Market, 0, len(markets))

	for _, u := range markets {
		block := source.getNextBlock(t, ctx)
		u.market.VegaTime = block.VegaTime
		u.market.State = u.state
		u.market.TradingMode = u.tradingMode
		err := md.Upsert(ctx, &u.market)
		entries = append(entries, u.market)
		require.NoError(t, err)
	}

	return md, entries, props
}

func TestMarketsEnums(t *testing.T) {
	t.Run("All proto market states should be supported", testMarketState)
	t.Run("All proto market trading modes should be supported", testMarketTradingMode)
}

func testMarketState(t *testing.T) {
	var marketState vega.Market_State
	states := getEnums(t, marketState)
	assert.Len(t, states, 11)
	for s, state := range states {
		t.Run(state, func(t *testing.T) {
			bs, md := setupMarketsTest(t)

			ctx := tempTransaction(t)

			block := addTestBlock(t, ctx, bs)

			marketProto := getTestFutureMarket(true)
			marketProto.State = vega.Market_State(s)

			market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
			require.NoError(t, err, "Converting market proto to database entity")
			require.NoError(t, md.Upsert(ctx, market))
			got, err := md.GetByID(ctx, market.ID.String())
			require.NoError(t, err)
			assert.Equal(t, market.State, got.State)
		})
	}
}

func testMarketTradingMode(t *testing.T) {
	var marketTradingMode vega.Market_TradingMode
	modes := getEnums(t, marketTradingMode)
	assert.Len(t, modes, 9)
	for m, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			bs, md := setupMarketsTest(t)

			ctx := tempTransaction(t)

			block := addTestBlock(t, ctx, bs)

			marketProto := getTestFutureMarket(true)
			marketProto.TradingMode = vega.Market_TradingMode(m)

			market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
			require.NoError(t, err, "Converting market proto to database entity")
			require.NoError(t, md.Upsert(ctx, market))
			got, err := md.GetByID(ctx, market.ID.String())
			require.NoError(t, err)
			assert.Equal(t, market.TradingMode, got.TradingMode)
		})
	}
}
