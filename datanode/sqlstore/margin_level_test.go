// Copyright (c) 2022 Gobalsky Labs Limited
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
	"github.com/jackc/pgx/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAssetId = "deadbeef"

func TestMarginLevels(t *testing.T) {
	t.Run("Add should insert margin levels that don't exist in the current block", testInsertMarginLevels)
	t.Run("Add should insert margin levels that already exist in the same block", testDuplicateMarginLevelInSameBlock)
	t.Run("GetMarginLevelsByID should return the latest state of margin levels for all markets if only party ID is provided", testGetMarginLevelsByPartyID)
	t.Run("GetMarginLevelsByID should return the latest state of margin levels for all parties if only market ID is provided", testGetMarginLevelsByMarketID)
	t.Run("GetMarginLevelsByID should return the latest state of margin levels for the given party/market ID", testGetMarginLevelsByID)

	t.Run("GetMarginLevelsByIDWithCursorPagination should return all margin levels for a given party if no pagination is provided", testGetMarginLevelsByIDPaginationWithPartyNoCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return all margin levels for a given market if no pagination is provided", testGetMarginLevelsByIDPaginationWithMarketNoCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the first page of margin levels for a given party if first is set with no after cursor", testGetMarginLevelsByIDPaginationWithPartyFirstNoAfterCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the first page of margin levels for a given market if first is set with no after cursor", testGetMarginLevelsByIDPaginationWithMarketFirstNoAfterCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the last page of margin levels for a given party if last is set with no before cursor", testGetMarginLevelsByIDPaginationWithPartyLastNoBeforeCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the last page of margin levels for a given market if last is set with no before cursor", testGetMarginLevelsByIDPaginationWithMarketLastNoBeforeCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given party if first is set with after cursor", testGetMarginLevelsByIDPaginationWithPartyFirstAndAfterCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given market if first is set with after cursor", testGetMarginLevelsByIDPaginationWithMarketFirstAndAfterCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given party if last is set with before cursor", testGetMarginLevelsByIDPaginationWithPartyLastAndBeforeCursor)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given market if last is set with before cursor", testGetMarginLevelsByIDPaginationWithMarketLastAndBeforeCursor)

	t.Run("GetMarginLevelsByIDWithCursorPagination should return all margin levels for a given party if no pagination is provided - Newest First", testGetMarginLevelsByIDPaginationWithPartyNoCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return all margin levels for a given market if no pagination is provided - Newest First", testGetMarginLevelsByIDPaginationWithMarketNoCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the first page of margin levels for a given party if first is set with no after cursor - Newest First", testGetMarginLevelsByIDPaginationWithPartyFirstNoAfterCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the first page of margin levels for a given market if first is set with no after cursor - Newest First", testGetMarginLevelsByIDPaginationWithMarketFirstNoAfterCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the last page of margin levels for a given party if last is set with no before cursor - Newest First", testGetMarginLevelsByIDPaginationWithPartyLastNoBeforeCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the last page of margin levels for a given market if last is set with no before cursor - Newest First", testGetMarginLevelsByIDPaginationWithMarketLastNoBeforeCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given party if first is set with after cursor - Newest First", testGetMarginLevelsByIDPaginationWithPartyFirstAndAfterCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given market if first is set with after cursor - Newest First", testGetMarginLevelsByIDPaginationWithMarketFirstAndAfterCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given party if last is set with before cursor - Newest First", testGetMarginLevelsByIDPaginationWithPartyLastAndBeforeCursorNewestFirst)
	t.Run("GetMarginLevelsByIDWithCursorPagination should return the requested page of margin levels for a given market if last is set with before cursor - Newest First", testGetMarginLevelsByIDPaginationWithMarketLastAndBeforeCursorNewestFirst)
}

type testBlockSource struct {
	blockStore *sqlstore.Blocks
	blockTime  time.Time
}

func (bs *testBlockSource) getNextBlock(t *testing.T) entities.Block {
	t.Helper()
	bs.blockTime = bs.blockTime.Add(1 * time.Second)
	return addTestBlockForTime(t, bs.blockStore, bs.blockTime)
}

func setupMarginLevelTests(t *testing.T, ctx context.Context) (*testBlockSource, *sqlstore.MarginLevels, *sqlstore.Accounts, *pgx.Conn) {
	t.Helper()
	DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	testBlockSource := &testBlockSource{bs, time.Now()}

	block := testBlockSource.getNextBlock(t)

	assets := sqlstore.NewAssets(connectionSource)

	testAsset := entities.Asset{
		ID:            testAssetId,
		Name:          "testAssetName",
		Symbol:        "tan",
		TotalSupply:   decimal.NewFromInt(20),
		Decimals:      1,
		Quantum:       decimal.NewFromInt(1),
		Source:        "TS",
		ERC20Contract: "ET",
		VegaTime:      block.VegaTime,
	}

	err := assets.Add(context.Background(), testAsset)
	if err != nil {
		t.Fatalf("failed to add test asset:%s", err)
	}

	accountStore := sqlstore.NewAccounts(connectionSource)
	ml := sqlstore.NewMarginLevels(connectionSource)
	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, config.ConnectionConfig.GetConnectionString())
	require.NoError(t, err)

	return testBlockSource, ml, accountStore, conn
}

func testInsertMarginLevels(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	blockSource, ml, accountStore, conn := setupMarginLevelTests(t, ctx)
	block := blockSource.getNextBlock(t)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)

	marginLevelProto := getMarginLevelProto()
	marginLevel, err := entities.MarginLevelsFromProto(context.Background(), marginLevelProto, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	_, err = ml.Flush(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testDuplicateMarginLevelInSameBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	blockSource, ml, accountStore, conn := setupMarginLevelTests(t, ctx)
	block := blockSource.getNextBlock(t)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)

	marginLevelProto := getMarginLevelProto()
	marginLevel, err := entities.MarginLevelsFromProto(context.Background(), marginLevelProto, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel)
	require.NoError(t, err)

	err = ml.Add(marginLevel)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	_, err = ml.Flush(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func getMarginLevelProto() *vega.MarginLevels {
	return getMarginLevelWithMaintenanceProto("1000", "deadbeef", "deadbeef", time.Now().UnixNano())
}

func getMarginLevelWithMaintenanceProto(maintenanceMargin, partyId, marketId string, timestamp int64) *vega.MarginLevels {
	return &vega.MarginLevels{
		MaintenanceMargin:      maintenanceMargin,
		SearchLevel:            "1000",
		InitialMargin:          "1000",
		CollateralReleaseLevel: "1000",
		PartyId:                partyId,
		MarketId:               marketId,
		Asset:                  testAssetId,
		Timestamp:              timestamp,
	}
}

func testGetMarginLevelsByPartyID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	blockSource, ml, accountStore, conn := setupMarginLevelTests(t, ctx)
	block := blockSource.getNextBlock(t)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	ml1 := getMarginLevelProto()
	ml2 := getMarginLevelProto()
	ml3 := getMarginLevelProto()
	ml4 := getMarginLevelProto()

	ml2.MarketId = "deadbaad"

	ml3.Timestamp = ml2.Timestamp + 1000000000
	ml3.MaintenanceMargin = "2000"
	ml3.SearchLevel = "2000"

	ml4.Timestamp = ml2.Timestamp + 1000000000
	ml4.MaintenanceMargin = "2000"
	ml4.SearchLevel = "2000"
	ml4.MarketId = "deadbaad"

	marginLevel1, err := entities.MarginLevelsFromProto(context.Background(), ml1, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	marginLevel2, err := entities.MarginLevelsFromProto(context.Background(), ml2, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel1)
	require.NoError(t, err)
	err = ml.Add(marginLevel2)
	require.NoError(t, err)

	_, err = ml.Flush(ctx)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	block = blockSource.getNextBlock(t)
	marginLevel3, err := entities.MarginLevelsFromProto(context.Background(), ml3, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	marginLevel4, err := entities.MarginLevelsFromProto(context.Background(), ml4, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel3)
	require.NoError(t, err)
	err = ml.Add(marginLevel4)
	require.NoError(t, err)

	_, err = ml.Flush(ctx)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 4, rowCount)

	got, err := ml.GetMarginLevelsByID(ctx, "DEADBEEF", "", entities.OffsetPagination{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(got))

	// We have to truncate the time because Postgres only supports time to microsecond granularity.
	want1 := marginLevel3
	want1.Timestamp = want1.Timestamp.Truncate(time.Microsecond)
	want1.VegaTime = want1.VegaTime.Truncate(time.Microsecond)

	want2 := marginLevel4
	want2.Timestamp = want2.Timestamp.Truncate(time.Microsecond)
	want2.VegaTime = want2.VegaTime.Truncate(time.Microsecond)

	want := []entities.MarginLevels{want1, want2}

	assert.ElementsMatch(t, want, got)
}

func testGetMarginLevelsByMarketID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	blockSource, ml, accountStore, conn := setupMarginLevelTests(t, ctx)
	block := blockSource.getNextBlock(t)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	ml1 := getMarginLevelProto()
	ml2 := getMarginLevelProto()
	ml3 := getMarginLevelProto()
	ml4 := getMarginLevelProto()

	ml2.PartyId = "deadbaad"

	ml3.Timestamp = ml2.Timestamp + 1000000000
	ml3.MaintenanceMargin = "2000"
	ml3.SearchLevel = "2000"

	ml4.Timestamp = ml2.Timestamp + 1000000000
	ml4.MaintenanceMargin = "2000"
	ml4.SearchLevel = "2000"
	ml4.PartyId = "deadbaad"

	marginLevel1, err := entities.MarginLevelsFromProto(context.Background(), ml1, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	marginLevel2, err := entities.MarginLevelsFromProto(context.Background(), ml2, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel1)
	require.NoError(t, err)
	err = ml.Add(marginLevel2)
	require.NoError(t, err)

	_, err = ml.Flush(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	time.Sleep(time.Second)

	block = blockSource.getNextBlock(t)
	marginLevel3, err := entities.MarginLevelsFromProto(context.Background(), ml3, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	marginLevel4, err := entities.MarginLevelsFromProto(context.Background(), ml4, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel3)
	require.NoError(t, err)
	err = ml.Add(marginLevel4)
	require.NoError(t, err)

	_, err = ml.Flush(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 4, rowCount)

	got, err := ml.GetMarginLevelsByID(ctx, "", "DEADBEEF", entities.OffsetPagination{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(got))

	// We have to truncate the time because Postgres only supports time to microsecond granularity.
	want1 := marginLevel3
	want1.Timestamp = want1.Timestamp.Truncate(time.Microsecond)
	want1.VegaTime = want1.VegaTime.Truncate(time.Microsecond)

	want2 := marginLevel4
	want2.Timestamp = want2.Timestamp.Truncate(time.Microsecond)
	want2.VegaTime = want2.VegaTime.Truncate(time.Microsecond)

	want := []entities.MarginLevels{want1, want2}

	assert.ElementsMatch(t, want, got)
}

func testGetMarginLevelsByID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	blockSource, ml, accountStore, conn := setupMarginLevelTests(t, ctx)
	block := blockSource.getNextBlock(t)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	ml1 := getMarginLevelProto()
	ml2 := getMarginLevelProto()
	ml3 := getMarginLevelProto()
	ml4 := getMarginLevelProto()

	ml2.PartyId = "DEADBAAD"

	ml3.Timestamp = ml2.Timestamp + 1000000000
	ml3.MaintenanceMargin = "2000"
	ml3.SearchLevel = "2000"

	ml4.Timestamp = ml2.Timestamp + 1000000000
	ml4.MaintenanceMargin = "2000"
	ml4.SearchLevel = "2000"
	ml4.PartyId = "DEADBAAD"

	marginLevel1, err := entities.MarginLevelsFromProto(context.Background(), ml1, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	marginLevel2, err := entities.MarginLevelsFromProto(context.Background(), ml2, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel1)
	require.NoError(t, err)
	err = ml.Add(marginLevel2)
	require.NoError(t, err)

	_, err = ml.Flush(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	time.Sleep(time.Second)

	block = blockSource.getNextBlock(t)
	marginLevel3, err := entities.MarginLevelsFromProto(context.Background(), ml3, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	marginLevel4, err := entities.MarginLevelsFromProto(context.Background(), ml4, accountStore, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	err = ml.Add(marginLevel3)
	require.NoError(t, err)
	err = ml.Add(marginLevel4)
	require.NoError(t, err)

	_, err = ml.Flush(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 4, rowCount)

	got, err := ml.GetMarginLevelsByID(ctx, "DEADBEEF", "DEADBEEF", entities.OffsetPagination{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(got))

	// We have to truncate the time because Postgres only supports time to microsecond granularity.
	want1 := marginLevel3
	want1.Timestamp = want1.Timestamp.Truncate(time.Microsecond)
	want1.VegaTime = want1.VegaTime.Truncate(time.Microsecond)

	want := []entities.MarginLevels{want1}

	assert.ElementsMatch(t, want, got)
}

func populateMarginLevelPaginationTestData(t *testing.T, ctx context.Context) (*sqlstore.MarginLevels, map[int]entities.Block, map[int]entities.MarginLevels) {
	t.Helper()
	DeleteEverything()

	blockSource, mlStore, accountStore, _ := setupMarginLevelTests(t, ctx)

	margins := []struct {
		maintenanceMargin string
		partyID           string
		marketID          string
	}{
		{
			// 0
			maintenanceMargin: "1000",
			partyID:           "DEADBEEF",
			marketID:          "DEADBAAD",
		},
		{
			// 1
			maintenanceMargin: "1001",
			partyID:           "DEADBEEF",
			marketID:          "0FF1CE",
		},
		{
			// 2
			maintenanceMargin: "1002",
			partyID:           "DEADBAAD",
			marketID:          "DEADBEEF",
		},
		{
			// 3
			maintenanceMargin: "1003",
			partyID:           "DEADBAAD",
			marketID:          "DEADBAAD",
		},
		{
			// 4
			maintenanceMargin: "1004",
			partyID:           "DEADBEEF",
			marketID:          "DEADC0DE",
		},
		{
			// 5
			maintenanceMargin: "1005",
			partyID:           "0FF1CE",
			marketID:          "DEADBEEF",
		},
		{
			// 6
			maintenanceMargin: "1006",
			partyID:           "DEADC0DE",
			marketID:          "DEADBEEF",
		},
		{
			// 7
			maintenanceMargin: "1007",
			partyID:           "DEADBEEF",
			marketID:          "CAFED00D",
		},
		{
			// 8
			maintenanceMargin: "1008",
			partyID:           "CAFED00D",
			marketID:          "DEADBEEF",
		},
		{
			// 9
			maintenanceMargin: "1009",
			partyID:           "DEADBAAD",
			marketID:          "DEADBAAD",
		},
		{
			// 10
			maintenanceMargin: "1010",
			partyID:           "DEADBEEF",
			marketID:          "CAFEB0BA",
		},
		{
			// 11
			maintenanceMargin: "1011",
			partyID:           "CAFEB0BA",
			marketID:          "DEADBEEF",
		},
		{
			// 12
			maintenanceMargin: "1012",
			partyID:           "DEADBAAD",
			marketID:          "DEADBAAD",
		},
		{
			// 13
			maintenanceMargin: "1013",
			partyID:           "0D15EA5E",
			marketID:          "DEADBEEF",
		},
		{
			// 14
			maintenanceMargin: "1014",
			partyID:           "DEADBEEF",
			marketID:          "0D15EA5E",
		},
	}

	blocks := make(map[int]entities.Block)
	marginLevels := make(map[int]entities.MarginLevels)

	for i, ml := range margins {
		block := blockSource.getNextBlock(t)
		mlProto := getMarginLevelWithMaintenanceProto(ml.maintenanceMargin, ml.partyID, ml.marketID, block.VegaTime.UnixNano())
		mlEntity, err := entities.MarginLevelsFromProto(ctx, mlProto, accountStore, block.VegaTime)
		require.NoError(t, err, "Converting margin levels proto to database entity")
		err = mlStore.Add(mlEntity)
		require.NoError(t, err)

		blocks[i] = block
		marginLevels[i] = mlEntity
	}

	_, err := mlStore.Flush(ctx)
	require.NoError(t, err)

	return mlStore, blocks, marginLevels
}

func testGetMarginLevelsByIDPaginationWithPartyNoCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 6)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[0],
		marginLevels[1],
		marginLevels[4],
		marginLevels[7],
		marginLevels[10],
		marginLevels[14],
	}
	assert.Equal(t, wantMarginLevels, got)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[0].VegaTime,
		AccountID: marginLevels[0].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[14].VegaTime,
		AccountID: marginLevels[14].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyNoCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 6)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[14],
		marginLevels[10],
		marginLevels[7],
		marginLevels[4],
		marginLevels[1],
		marginLevels[0],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[14].VegaTime,
		AccountID: marginLevels[14].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[0].VegaTime,
		AccountID: marginLevels[0].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketNoCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 6)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[2],
		marginLevels[5],
		marginLevels[6],
		marginLevels[8],
		marginLevels[11],
		marginLevels[13],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[2].VegaTime,
		AccountID: marginLevels[2].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[13].VegaTime,
		AccountID: marginLevels[13].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketNoCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 6)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[13],
		marginLevels[11],
		marginLevels[8],
		marginLevels[6],
		marginLevels[5],
		marginLevels[2],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[13].VegaTime,
		AccountID: marginLevels[13].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[2].VegaTime,
		AccountID: marginLevels[2].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyFirstNoAfterCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[0],
		marginLevels[1],
		marginLevels[4],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[0].VegaTime,
		AccountID: marginLevels[0].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[4].VegaTime,
		AccountID: marginLevels[4].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyFirstNoAfterCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[14],
		marginLevels[10],
		marginLevels[7],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[14].VegaTime,
		AccountID: marginLevels[14].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[7].VegaTime,
		AccountID: marginLevels[7].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketFirstNoAfterCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[2],
		marginLevels[5],
		marginLevels[6],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[2].VegaTime,
		AccountID: marginLevels[2].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[6].VegaTime,
		AccountID: marginLevels[6].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketFirstNoAfterCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[13],
		marginLevels[11],
		marginLevels[8],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[13].VegaTime,
		AccountID: marginLevels[13].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[8].VegaTime,
		AccountID: marginLevels[8].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyLastNoBeforeCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[7],
		marginLevels[10],
		marginLevels[14],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[7].VegaTime,
		AccountID: marginLevels[7].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[14].VegaTime,
		AccountID: marginLevels[14].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyLastNoBeforeCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[4],
		marginLevels[1],
		marginLevels[0],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[4].VegaTime,
		AccountID: marginLevels[4].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[0].VegaTime,
		AccountID: marginLevels[0].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketLastNoBeforeCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[8],
		marginLevels[11],
		marginLevels[13],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[8].VegaTime,
		AccountID: marginLevels[8].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[13].VegaTime,
		AccountID: marginLevels[13].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketLastNoBeforeCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[6],
		marginLevels[5],
		marginLevels[2],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[6].VegaTime,
		AccountID: marginLevels[6].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[2].VegaTime,
		AccountID: marginLevels[2].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyFirstAndAfterCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	after := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[1].VegaTime,
		AccountID: marginLevels[1].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[4],
		marginLevels[7],
		marginLevels[10],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[4].VegaTime,
		AccountID: marginLevels[4].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[10].VegaTime,
		AccountID: marginLevels[10].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyFirstAndAfterCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	after := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[10].VegaTime,
		AccountID: marginLevels[10].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[7],
		marginLevels[4],
		marginLevels[1],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[7].VegaTime,
		AccountID: marginLevels[7].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[1].VegaTime,
		AccountID: marginLevels[1].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketFirstAndAfterCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	after := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[5].VegaTime,
		AccountID: marginLevels[5].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[6],
		marginLevels[8],
		marginLevels[11],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[6].VegaTime,
		AccountID: marginLevels[6].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[11].VegaTime,
		AccountID: marginLevels[11].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketFirstAndAfterCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	after := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[11].VegaTime,
		AccountID: marginLevels[11].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[8],
		marginLevels[6],
		marginLevels[5],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[8].VegaTime,
		AccountID: marginLevels[8].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[5].VegaTime,
		AccountID: marginLevels[5].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyLastAndBeforeCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	before := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[10].VegaTime,
		AccountID: marginLevels[10].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[1],
		marginLevels[4],
		marginLevels[7],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[1].VegaTime,
		AccountID: marginLevels[1].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[7].VegaTime,
		AccountID: marginLevels[7].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithPartyLastAndBeforeCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	before := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[1].VegaTime,
		AccountID: marginLevels[1].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[10],
		marginLevels[7],
		marginLevels[4],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[10].VegaTime,
		AccountID: marginLevels[10].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[4].VegaTime,
		AccountID: marginLevels[4].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketLastAndBeforeCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	before := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[11].VegaTime,
		AccountID: marginLevels[11].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[5],
		marginLevels[6],
		marginLevels[8],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[5].VegaTime,
		AccountID: marginLevels[5].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[8].VegaTime,
		AccountID: marginLevels[8].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}

func testGetMarginLevelsByIDPaginationWithMarketLastAndBeforeCursorNewestFirst(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	before := entities.NewCursor(entities.MarginCursor{
		VegaTime:  blocks[5].VegaTime,
		AccountID: marginLevels[5].AccountID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[11],
		marginLevels[8],
		marginLevels[6],
	}
	assert.Equal(t, wantMarginLevels, got)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[11].VegaTime,
		AccountID: marginLevels[11].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[6].VegaTime,
		AccountID: marginLevels[6].AccountID,
	}
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(wantStartCursor.String()).Encode(),
		EndCursor:       entities.NewCursor(wantEndCursor.String()).Encode(),
	}, pageInfo)
}
