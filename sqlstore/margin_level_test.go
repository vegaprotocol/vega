package sqlstore_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
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

	levels, err := ml.GetMarginLevelsByID(context.Background(), "", "deadbeef", entities.Pagination{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(levels))

	// The conflated data should be used to obtain the latest margin level when raw margin level data is not available
	lastMarginLevelInserted := marginLevels[len(marginLevels)-1]
	assert.Equal(t, lastMarginLevelInserted, levels[0])

	marginLevels = addMarginLevels(t, 150, blockSource, accountStore, ml)

	levels, err = ml.GetMarginLevelsByID(context.Background(), "", "deadbeef", entities.Pagination{})
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

	got, err := ml.GetMarginLevelsByID(ctx, "DEADBEEF", "", entities.Pagination{})
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

	got, err := ml.GetMarginLevelsByID(ctx, "", "DEADBEEF", entities.Pagination{})
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

	got, err := ml.GetMarginLevelsByID(ctx, "DEADBEEF", "DEADBEEF", entities.Pagination{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(got))

	// We have to truncate the time because Postgres only supports time to microsecond granularity.
	want1 := marginLevel3
	want1.Timestamp = want1.Timestamp.Truncate(time.Microsecond)
	want1.VegaTime = want1.VegaTime.Truncate(time.Microsecond)

	want := []entities.MarginLevels{want1}

	assert.ElementsMatch(t, want, got)
}
