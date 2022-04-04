package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/protos/vega"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarginLevels(t *testing.T) {
	t.Run("Add should insert margin levels that don't exist in the current block", testInsertMarginLevels)
	t.Run("Add should insert margin levels that already exist in the same block", testDuplicateMarginLevelInSameBlock)
	t.Run("GetMarginLevelsByID should return the latest state of margin levels for all markets if only party ID is provided", testGetMarginLevelsByPartyID)
	t.Run("GetMarginLevelsByID should return the latest state of margin levels for all parties if only market ID is provided", testGetMarginLevelsByMarketID)
	t.Run("GetMarginLevelsByID should return the latest state of margin levels for the given party/market ID", testGetMarginLevelsByID)
}

func setupMarginLevelTests(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.MarginLevels, *pgx.Conn) {
	t.Helper()
	err := testStore.DeleteEverything()
	require.NoError(t, err)

	bs := sqlstore.NewBlocks(testStore)
	ml := sqlstore.NewMarginLevels(testStore)
	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, connectionString(config))
	require.NoError(t, err)

	return bs, ml, conn
}

func testInsertMarginLevels(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ml, conn := setupMarginLevelTests(t, ctx)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)

	seqNum := uint64(1)
	block := addTestBlock(t, bs)
	marginLevelProto := getMarginLevelProto()
	marginLevel, err := entities.MarginLevelsFromProto(marginLevelProto, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	marginLevel.SeqNum = seqNum
	marginLevel.SyntheticTime = marginLevel.VegaTime.Add(time.Duration(marginLevel.SeqNum) * time.Microsecond)

	err = ml.Add(marginLevel)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testDuplicateMarginLevelInSameBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ml, conn := setupMarginLevelTests(t, ctx)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)

	block := addTestBlock(t, bs)
	marginLevelProto := getMarginLevelProto()
	marginLevel, err := entities.MarginLevelsFromProto(marginLevelProto, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")

	seqNum := uint64(1)
	marginLevel.SeqNum = seqNum
	marginLevel.SyntheticTime = marginLevel.VegaTime.Add(time.Duration(marginLevel.SeqNum) * time.Microsecond)
	err = ml.Add(marginLevel)
	require.NoError(t, err)

	seqNum += 1
	marginLevel.SeqNum = seqNum
	marginLevel.SyntheticTime = marginLevel.VegaTime.Add(time.Duration(marginLevel.SeqNum) * time.Microsecond)
	err = ml.Add(marginLevel)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)
}

func getMarginLevelProto() *vega.MarginLevels {
	return &vega.MarginLevels{
		MaintenanceMargin:      "1000",
		SearchLevel:            "1000",
		InitialMargin:          "1000",
		CollateralReleaseLevel: "1000",
		PartyId:                "deadbeef",
		MarketId:               "deadbeef",
		Asset:                  "deadbeef",
		Timestamp:              time.Now().UnixNano(),
	}
}

func testGetMarginLevelsByPartyID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ml, conn := setupMarginLevelTests(t, ctx)

	var rowCount int
	err := conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 0, rowCount)
	var seqNum uint64

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

	block := addTestBlock(t, bs)
	marginLevel1, err := entities.MarginLevelsFromProto(ml1, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel1.SeqNum = seqNum
	marginLevel1.SyntheticTime = marginLevel1.VegaTime.Add(time.Duration(marginLevel1.SeqNum) * time.Microsecond)

	marginLevel2, err := entities.MarginLevelsFromProto(ml2, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel2.SeqNum = seqNum
	marginLevel2.SyntheticTime = marginLevel2.VegaTime.Add(time.Duration(marginLevel2.SeqNum) * time.Microsecond)

	err = ml.Add(marginLevel1)
	require.NoError(t, err)
	err = ml.Add(marginLevel2)
	require.NoError(t, err)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	marginLevel3, err := entities.MarginLevelsFromProto(ml3, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel3.SeqNum = seqNum
	marginLevel3.SyntheticTime = marginLevel3.VegaTime.Add(time.Duration(marginLevel3.SeqNum) * time.Microsecond)

	marginLevel4, err := entities.MarginLevelsFromProto(ml4, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel4.SeqNum = seqNum
	marginLevel4.SyntheticTime = marginLevel4.VegaTime.Add(time.Duration(marginLevel4.SeqNum) * time.Microsecond)

	err = ml.Add(marginLevel3)
	require.NoError(t, err)
	err = ml.Add(marginLevel4)
	require.NoError(t, err)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 4, rowCount)

	got, err := ml.GetMarginLevelsByID(ctx, "DEADBEEF", "", entities.Pagination{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(got))

	// We have to truncate the time because Postgres only supports time to microsecond granularity.
	want1 := *marginLevel3
	want1.Timestamp = want1.Timestamp.Truncate(time.Microsecond)
	want1.VegaTime = want1.VegaTime.Truncate(time.Microsecond)

	want2 := *marginLevel4
	want2.Timestamp = want2.Timestamp.Truncate(time.Microsecond)
	want2.VegaTime = want2.VegaTime.Truncate(time.Microsecond)

	want := []entities.MarginLevels{want1, want2}

	assert.ElementsMatch(t, want, got)
}

func testGetMarginLevelsByMarketID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ml, conn := setupMarginLevelTests(t, ctx)

	var rowCount int
	var seqNum uint64
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

	block := addTestBlock(t, bs)
	marginLevel1, err := entities.MarginLevelsFromProto(ml1, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel1.SeqNum = seqNum
	marginLevel1.SyntheticTime = marginLevel1.VegaTime.Add(time.Duration(marginLevel1.SeqNum) * time.Microsecond)

	marginLevel2, err := entities.MarginLevelsFromProto(ml2, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel2.SeqNum = seqNum
	marginLevel2.SyntheticTime = marginLevel2.VegaTime.Add(time.Duration(marginLevel2.SeqNum) * time.Microsecond)

	err = ml.Add(marginLevel1)
	require.NoError(t, err)
	err = ml.Add(marginLevel2)
	require.NoError(t, err)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	marginLevel3, err := entities.MarginLevelsFromProto(ml3, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel3.SeqNum = seqNum
	marginLevel3.SyntheticTime = marginLevel3.VegaTime.Add(time.Duration(marginLevel3.SeqNum) * time.Microsecond)

	marginLevel4, err := entities.MarginLevelsFromProto(ml4, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel4.SeqNum = seqNum
	marginLevel4.SyntheticTime = marginLevel4.VegaTime.Add(time.Duration(marginLevel4.SeqNum) * time.Microsecond)

	err = ml.Add(marginLevel3)
	require.NoError(t, err)
	err = ml.Add(marginLevel4)
	require.NoError(t, err)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 4, rowCount)

	got, err := ml.GetMarginLevelsByID(ctx, "", "DEADBEEF", entities.Pagination{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(got))

	// We have to truncate the time because Postgres only supports time to microsecond granularity.
	want1 := *marginLevel3
	want1.Timestamp = want1.Timestamp.Truncate(time.Microsecond)
	want1.VegaTime = want1.VegaTime.Truncate(time.Microsecond)

	want2 := *marginLevel4
	want2.Timestamp = want2.Timestamp.Truncate(time.Microsecond)
	want2.VegaTime = want2.VegaTime.Truncate(time.Microsecond)

	want := []entities.MarginLevels{want1, want2}

	assert.ElementsMatch(t, want, got)
}

func testGetMarginLevelsByID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ml, conn := setupMarginLevelTests(t, ctx)

	var rowCount int
	var seqNum uint64
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

	block := addTestBlock(t, bs)
	marginLevel1, err := entities.MarginLevelsFromProto(ml1, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel1.SeqNum = seqNum
	marginLevel1.SyntheticTime = marginLevel1.VegaTime.Add(time.Duration(marginLevel1.SeqNum) * time.Microsecond)

	marginLevel2, err := entities.MarginLevelsFromProto(ml2, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel2.SeqNum = seqNum
	marginLevel2.SyntheticTime = marginLevel2.VegaTime.Add(time.Duration(marginLevel2.SeqNum) * time.Microsecond)

	err = ml.Add(marginLevel1)
	require.NoError(t, err)
	err = ml.Add(marginLevel2)
	require.NoError(t, err)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	marginLevel3, err := entities.MarginLevelsFromProto(ml3, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel3.SeqNum = seqNum
	marginLevel3.SyntheticTime = marginLevel3.VegaTime.Add(time.Duration(marginLevel3.SeqNum) * time.Microsecond)

	marginLevel4, err := entities.MarginLevelsFromProto(ml4, block.VegaTime)
	require.NoError(t, err, "Converting margin levels proto to database entity")
	seqNum += 1
	marginLevel4.SeqNum = seqNum
	marginLevel4.SyntheticTime = marginLevel4.VegaTime.Add(time.Duration(marginLevel4.SeqNum) * time.Microsecond)

	err = ml.Add(marginLevel3)
	require.NoError(t, err)
	err = ml.Add(marginLevel4)
	require.NoError(t, err)

	err = ml.OnTimeUpdateEvent(ctx)
	assert.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from margin_levels`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 4, rowCount)

	got, err := ml.GetMarginLevelsByID(ctx, "DEADBEEF", "DEADBEEF", entities.Pagination{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(got))

	// We have to truncate the time because Postgres only supports time to microsecond granularity.
	want1 := *marginLevel3
	want1.Timestamp = want1.Timestamp.Truncate(time.Microsecond)
	want1.VegaTime = want1.VegaTime.Truncate(time.Microsecond)

	want := []entities.MarginLevels{want1}

	assert.ElementsMatch(t, want, got)
}
