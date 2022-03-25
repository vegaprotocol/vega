package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleSpec(t *testing.T) {
	t.Run("Upsert should insert an OracleSpec when the id does not exist in the current block", testInsertIntoNewBlock)
	t.Run("Upsert should update an OracleSpec when the id already exists in the current block", testUpdateExistingInBlock)
	t.Run("GetSpecByID should retrieve the latest version of the specified OracleSpec", testGetSpecByID)
	t.Run("GetSpecs should retrieve the latest versions of all OracleSpecs", testGetSpecs)
}

func setupOracleSpecTest(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.OracleSpec, *pgx.Conn) {
	t.Helper()
	err := testStore.DeleteEverything()
	require.NoError(t, err)

	bs := sqlstore.NewBlocks(testStore)
	os := sqlstore.NewOracleSpec(testStore)

	config := NewTestConfig(testDBPort)
	conn, err := pgx.Connect(ctx, connectionString(config))
	require.NoError(t, err)

	return bs, os, conn
}

func testInsertIntoNewBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	proto := specProtos[0]
	data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
	require.NoError(t, err)
	assert.NoError(t, os.Upsert(data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)
}

func testUpdateExistingInBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	proto := specProtos[0]
	data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
	require.NoError(t, err)
	assert.NoError(t, os.Upsert(data))

	data.Status = entities.OracleSpecDeactivated
	assert.NoError(t, os.Upsert(data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)
}

func testGetSpecByID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	for _, proto := range specProtos {
		data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, os.Upsert(data))
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)

	got, err := os.GetSpecByID(ctx, "DEADBEEF")
	require.NoError(t, err)

	want, err := entities.OracleSpecFromProto(specProtos[0], block.VegaTime)
	assert.NoError(t, err)
	// truncate the time to microseconds as postgres doesn't support nanosecond granularity.
	want.UpdatedAt = want.UpdatedAt.Truncate(time.Microsecond)
	want.CreatedAt = want.CreatedAt.Truncate(time.Microsecond)
	assert.Equal(t, *want, got)
}

func testGetSpecs(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	want := make([]entities.OracleSpec, 0)

	for _, proto := range specProtos {
		data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, os.Upsert(data))

		// truncate the time to microseconds as postgres doesn't support nanosecond granularity.
		data.CreatedAt = data.CreatedAt.Truncate(time.Microsecond)
		data.UpdatedAt = data.UpdatedAt.Truncate(time.Microsecond)
		want = append(want, *data)
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)

	got, err := os.GetSpecs(ctx, entities.Pagination{})
	require.NoError(t, err)
	assert.ElementsMatch(t, want, got)
}

func getTestSpecs() []*oraclespb.OracleSpec {
	return []*oraclespb.OracleSpec{
		{
			Id:        "deadbeef",
			CreatedAt: time.Now().UnixNano(),
			UpdatedAt: time.Now().UnixNano(),
			PubKeys:   []string{"b105f00d", "baddcafe"},
			Filters: []*oraclespb.Filter{
				{
					Key: &oraclespb.PropertyKey{
						Name: "Ticker",
						Type: oraclespb.PropertyKey_TYPE_STRING,
					},
					Conditions: []*oraclespb.Condition{
						{
							Operator: oraclespb.Condition_OPERATOR_EQUALS,
							Value:    "USDETH",
						},
					},
				},
			},
			Status: oraclespb.OracleSpec_STATUS_ACTIVE,
		},
		{
			Id:        "cafed00d",
			CreatedAt: time.Now().UnixNano(),
			UpdatedAt: time.Now().UnixNano(),
			PubKeys:   []string{"b105f00d", "baddcafe"},
			Filters: []*oraclespb.Filter{
				{
					Key: &oraclespb.PropertyKey{
						Name: "Ticker",
						Type: oraclespb.PropertyKey_TYPE_STRING,
					},
					Conditions: []*oraclespb.Condition{
						{
							Operator: oraclespb.Condition_OPERATOR_EQUALS,
							Value:    "USDBTC",
						},
					},
				},
			},
			Status: oraclespb.OracleSpec_STATUS_ACTIVE,
		},
		{
			Id:        "deadbaad",
			CreatedAt: time.Now().UnixNano(),
			UpdatedAt: time.Now().UnixNano(),
			PubKeys:   []string{"b105f00d", "baddcafe"},
			Filters: []*oraclespb.Filter{
				{
					Key: &oraclespb.PropertyKey{
						Name: "Ticker",
						Type: oraclespb.PropertyKey_TYPE_STRING,
					},
					Conditions: []*oraclespb.Condition{
						{
							Operator: oraclespb.Condition_OPERATOR_EQUALS,
							Value:    "USDSOL",
						},
					},
				},
			},
			Status: oraclespb.OracleSpec_STATUS_ACTIVE,
		},
	}
}
