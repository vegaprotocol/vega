package sqlstore_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
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
	t.Run("GetMarginLevelsByID should return use conflated data where raw data is not available", testMarginLevelsDataRetention)

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
}

type testBlockSource struct {
	blockStore *sqlstore.Blocks
	blockTime  time.Time
}

func (bs *testBlockSource) getNextBlock(t *testing.T) entities.Block {
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
		ID:            entities.AssetID{ID: entities.ID(testAssetId)},
		Name:          "testAssetName",
		Symbol:        "tan",
		TotalSupply:   decimal.NewFromInt(20),
		Decimals:      1,
		Quantum:       1,
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

func testMarginLevelsDataRetention(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	bs := sqlstore.NewBlocks(connectionSource)
	blockSource := &testBlockSource{bs, yesterday.Add(1 * time.Hour)}
	_, ml, accountStore, conn := setupMarginLevelTests(t, ctx)

	marginLevels := addMarginLevels(t, 300, blockSource, accountStore, ml)

	_, err := conn.Exec(context.Background(), fmt.Sprintf("CALL refresh_continuous_aggregate('conflated_margin_levels', '%s', '%s');",
		yesterday.Format("2006-01-02"),
		time.Now().Format("2006-01-02")))

	assert.NoError(t, err)

	_, err = conn.Exec(context.Background(), "delete from margin_levels")
	assert.NoError(t, err)

	levels, err := ml.GetMarginLevelsByID(context.Background(), "", "deadbeef", entities.OffsetPagination{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(levels))

	// The conflated data should be used to obtain the latest margin level when raw margin level data is not available
	lastMarginLevelInserted := marginLevels[len(marginLevels)-1]
	assert.Equal(t, lastMarginLevelInserted, levels[0])

	marginLevels = addMarginLevels(t, 150, blockSource, accountStore, ml)

	levels, err = ml.GetMarginLevelsByID(context.Background(), "", "deadbeef", entities.OffsetPagination{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(levels))

	// The latest raw (non-conflated) margin data should be used to get the current margin level when it is available
	lastMarginLevelInserted = marginLevels[len(marginLevels)-1]
	assert.Equal(t, lastMarginLevelInserted, levels[0])
}

func addMarginLevels(t *testing.T, numMarginLevels int, blockSource *testBlockSource, accountStore *sqlstore.Accounts, ml *sqlstore.MarginLevels) []entities.MarginLevels {
	var marginLevels []entities.MarginLevels
	block := blockSource.getNextBlock(t)
	for i := 0; i < numMarginLevels; i++ {

		marginLevelProto := getMarginLevelWithMaintenanceProto(strconv.Itoa(i), "deadbeef",
			"deadbeef", 0)
		marginLevelProto2 := getMarginLevelWithMaintenanceProto(strconv.Itoa(i), "deadbeef",
			"deadbead", 0)
		marginLevel, err := entities.MarginLevelsFromProto(context.Background(), marginLevelProto, accountStore, block.VegaTime)
		marginLevels = append(marginLevels, marginLevel)
		require.NoError(t, err, "Converting margin levels proto to database entity")
		err = ml.Add(marginLevel)
		require.NoError(t, err)

		marginLevel2, err := entities.MarginLevelsFromProto(context.Background(), marginLevelProto2, accountStore, block.VegaTime)
		marginLevels = append(marginLevels, marginLevel)
		require.NoError(t, err, "Converting margin levels proto to database entity")
		err = ml.Add(marginLevel2)
		require.NoError(t, err)

		ml.Flush(context.Background())
		block = blockSource.getNextBlock(t)
	}
	return marginLevels
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

	err = ml.Flush(ctx)
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

	err = ml.Flush(ctx)
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

	err = ml.Flush(ctx)
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

	err = ml.Flush(ctx)
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

	err = ml.Flush(ctx)
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

	err = ml.Flush(ctx)
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

	err = ml.Flush(ctx)
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

	err = ml.Flush(ctx)
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

	err := mlStore.Flush(ctx)
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
	pagination, err := entities.PaginationFromProto(&v2.Pagination{})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 6)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[0],
		marginLevels[1],
		marginLevels[4],
		marginLevels[7],
		marginLevels[10],
		marginLevels[14],
	}
	assert.Equal(t, wantMarginLevels, results)
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
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
}

func testGetMarginLevelsByIDPaginationWithMarketNoCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	pagination, err := entities.PaginationFromProto(&v2.Pagination{})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 6)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[2],
		marginLevels[5],
		marginLevels[6],
		marginLevels[8],
		marginLevels[11],
		marginLevels[13],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[2].VegaTime,
		AccountID: marginLevels[2].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[13].VegaTime,
		AccountID: marginLevels[13].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
}

func testGetMarginLevelsByIDPaginationWithPartyFirstNoAfterCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[0],
		marginLevels[1],
		marginLevels[4],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[0].VegaTime,
		AccountID: marginLevels[0].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[4].VegaTime,
		AccountID: marginLevels[4].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
}

func testGetMarginLevelsByIDPaginationWithMarketFirstNoAfterCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	first := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[2],
		marginLevels[5],
		marginLevels[6],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[2].VegaTime,
		AccountID: marginLevels[2].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[6].VegaTime,
		AccountID: marginLevels[6].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
}

func testGetMarginLevelsByIDPaginationWithPartyLastNoBeforeCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last: &last,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[7],
		marginLevels[10],
		marginLevels[14],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[7].VegaTime,
		AccountID: marginLevels[7].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[14].VegaTime,
		AccountID: marginLevels[14].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
}

func testGetMarginLevelsByIDPaginationWithMarketLastNoBeforeCursor(t *testing.T) {
	testTimeout := time.Second * 5
	testCtx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	defer DeleteEverything()

	t.Logf("DB Port: %d", testDBPort)
	mls, blocks, marginLevels := populateMarginLevelPaginationTestData(t, context.Background())
	last := int32(3)
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last: &last,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[8],
		marginLevels[11],
		marginLevels[13],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[8].VegaTime,
		AccountID: marginLevels[8].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[13].VegaTime,
		AccountID: marginLevels[13].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
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
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
		After: &after,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[4],
		marginLevels[7],
		marginLevels[10],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[4].VegaTime,
		AccountID: marginLevels[4].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[10].VegaTime,
		AccountID: marginLevels[10].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
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
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		First: &first,
		After: &after,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[6],
		marginLevels[8],
		marginLevels[11],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[6].VegaTime,
		AccountID: marginLevels[6].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[11].VegaTime,
		AccountID: marginLevels[11].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
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
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last:   &last,
		Before: &before,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "DEADBEEF", "", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[1],
		marginLevels[4],
		marginLevels[7],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[1].VegaTime,
		AccountID: marginLevels[1].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[7].VegaTime,
		AccountID: marginLevels[7].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
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
	pagination, err := entities.PaginationFromProto(&v2.Pagination{
		Last:   &last,
		Before: &before,
	})
	require.NoError(t, err)
	results, pageInfo, err := mls.GetMarginLevelsByIDWithCursorPagination(testCtx, "", "DEADBEEF", pagination)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	wantMarginLevels := []entities.MarginLevels{
		marginLevels[5],
		marginLevels[6],
		marginLevels[8],
	}
	assert.Equal(t, wantMarginLevels, results)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
	wantStartCursor := entities.MarginCursor{
		VegaTime:  blocks[5].VegaTime,
		AccountID: marginLevels[5].AccountID,
	}
	wantEndCursor := entities.MarginCursor{
		VegaTime:  blocks[8].VegaTime,
		AccountID: marginLevels[8].AccountID,
	}
	assert.Equal(t, entities.NewCursor(wantStartCursor.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(wantEndCursor.String()).Encode(), pageInfo.EndCursor)
}
