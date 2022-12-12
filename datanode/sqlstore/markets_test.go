// (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkets_Add(t *testing.T) {
	t.Run("Add should insert a valid market record", shouldInsertAValidMarketRecord)
	t.Run("Add should update a valid market record if the block number already exists", shouldUpdateAValidMarketRecord)
}

func TestMarkets_Get(t *testing.T) {
	t.Run("GetByID should return the request market if it exists", getByIDShouldReturnTheRequestedMarketIfItExists)
	t.Run("GetByID should return error if the market does not exist", getByIDShouldReturnErrorIfTheMarketDoesNotExist)
	t.Run("GetAll should not include rejected markets", getAllShouldNotIncludeRejectedMarkets)
	t.Run("GetAllPaged should not include rejected markets", getAllPagedShouldNotIncludeRejectedMarkets)
}

func getByIDShouldReturnTheRequestedMarketIfItExists(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()
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

func getByIDShouldReturnErrorIfTheMarketDoesNotExist(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()
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

func getAllShouldNotIncludeRejectedMarkets(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()
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

	markets, err := md.GetAll(ctx, entities.OffsetPagination{})
	require.NoError(t, err)
	assert.Len(t, markets, 1)
	assert.Equal(t, market.ID, markets[0].ID)
	assert.Equal(t, market.TxHash, markets[0].TxHash)
	assert.Equal(t, market.VegaTime, markets[0].VegaTime)
	assert.Equal(t, market.State, markets[0].State)
}

func getAllPagedShouldNotIncludeRejectedMarkets(t *testing.T) {
	bs, md := setupMarketsTest(t)

	ctx, rollback := tempTransaction(t)
	defer rollback()
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

	markets, pageInfo, err := md.GetAllPaged(ctx, "", entities.CursorPagination{})
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

	ctx, rollback := tempTransaction(t)
	defer rollback()

	conn := connectionSource.Connection
	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)

	marketProto := getTestMarket()

	market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = md.Upsert(ctx, market)
	require.NoError(t, err, "Saving market entity to database")
	err = conn.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	conn := connectionSource.Connection
	var rowCount int

	t.Run("should have no markets in the database", func(t *testing.T) {
		err := conn.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
		require.NoError(t, err)
		assert.Equal(t, 0, rowCount)
	})

	var block entities.Block
	var marketProto *vega.Market

	t.Run("should insert a valid market record to the database", func(t *testing.T) {
		block = addTestBlock(t, ctx, bs)
		marketProto = getTestMarket()

		market, err := entities.NewMarketFromProto(marketProto, generateTxHash(), block.VegaTime)
		require.NoError(t, err, "Converting market proto to database entity")

		err = md.Upsert(ctx, market)
		require.NoError(t, err, "Saving market entity to database")

		var got entities.Market
		err = pgxscan.Get(ctx, conn, &got, `select * from markets where id = $1 and vega_time = $2`, market.ID, market.VegaTime)
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
		err = pgxscan.Get(ctx, conn, &got, `select * from markets where id = $1 and vega_time = $2`, market.ID, market.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)

		assert.Equal(t, marketProto.TradableInstrument, got.TradableInstrument.ToProto())
	})

	t.Run("should add the updated market record to the database if the block number has changed", func(t *testing.T) {
		newMarketProto := marketProto.DeepClone()
		newMarketProto.TradableInstrument.Instrument.Metadata.Tags = append(newMarketProto.TradableInstrument.Instrument.Metadata.Tags, "DDD")
		time.Sleep(time.Second)
		newBlock := addTestBlock(t, ctx, bs)

		market, err := entities.NewMarketFromProto(newMarketProto, generateTxHash(), newBlock.VegaTime)
		require.NoError(t, err, "Converting market proto to database entity")

		err = md.Upsert(ctx, market)
		require.NoError(t, err, "Saving market entity to database")

		err = conn.QueryRow(ctx, `select count(*) from markets`).Scan(&rowCount)
		require.NoError(t, err)
		assert.Equal(t, 2, rowCount)

		var gotFirstBlock, gotSecondBlock entities.Market

		err = pgxscan.Get(ctx, conn, &gotFirstBlock, `select * from markets where id = $1 and vega_time = $2`, market.ID, block.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)

		assert.Equal(t, marketProto.TradableInstrument, gotFirstBlock.TradableInstrument.ToProto())

		err = pgxscan.Get(ctx, conn, &gotSecondBlock, `select * from markets where id = $1 and vega_time = $2`, market.ID, newBlock.VegaTime)
		assert.NoError(t, err)
		assert.Equal(t, "TEST_INSTRUMENT", market.InstrumentID)

		assert.Equal(t, newMarketProto.TradableInstrument, gotSecondBlock.TradableInstrument.ToProto())
	})
}

func getTestMarket() *vega.Market {
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
								vega.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&vega.DataSourceSpecConfiguration{
									Signers: nil,
									Filters: nil,
								},
							),
							Status: 0,
						},
						DataSourceSpecForTradingTermination: &vega.DataSourceSpec{
							Id:        "",
							CreatedAt: 0,
							UpdatedAt: 0,
							Data: vega.NewDataSourceDefinition(
								vega.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&vega.DataSourceSpecConfiguration{
									Signers: nil,
									Filters: nil,
								},
							),
							Status: 0,
						},
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
			TriggeringRatio:  0,
			AuctionExtension: 0,
		},
		TradingMode: vega.Market_TRADING_MODE_CONTINUOUS,
		State:       vega.Market_STATE_ACTIVE,
		MarketTimestamps: &vega.MarketTimestamps{
			Proposed: 0,
			Pending:  0,
			Open:     0,
			Close:    0,
		},
		PositionDecimalPlaces: 8,
		LpPriceRange:          "0.95",
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

	for _, market := range markets {
		block := addTestBlock(t, ctx, bs)
		market.VegaTime = block.VegaTime
		blockTimes[market.ID.String()] = block.VegaTime
		err := md.Upsert(ctx, &market)
		require.NoError(t, err)
		time.Sleep(time.Microsecond * 100)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, md := setupMarketsTest(t)

	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "c612300d", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
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

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
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

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "c612300d", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
	blockTimes := make(map[string]time.Time)
	populateTestMarkets(ctx, t, bs, md, blockTimes)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
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

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()
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

	got, pageInfo, err := md.GetAllPaged(ctx, "", pagination)
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
