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

func TestOracleData(t *testing.T) {
	t.Run("Add should insert oracle data", testAddOracleData)
	t.Run("GetOracleDataBySpecID should return all data where matched spec ids contains the provided id", testGetOracleDataBySpecID)
}

func setupOracleDataTest(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.OracleData, *pgx.Conn) {
	t.Helper()
	err := testStore.DeleteEverything()
	require.NoError(t, err)

	bs := sqlstore.NewBlocks(testStore)
	od := sqlstore.NewOracleData(testStore)

	config := NewTestConfig(testDBPort)
	conn, err := pgx.Connect(ctx, connectionString(config))
	require.NoError(t, err)

	return bs, od, conn
}

func testAddOracleData(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, od, conn := setupOracleDataTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	dataProtos := getTestOracleData()

	for _, proto := range dataProtos {
		data, err := entities.OracleDataFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, od.Add(data))
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount))
	assert.Equal(t, len(dataProtos), rowCount)
}

func testGetOracleDataBySpecID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, od, conn := setupOracleDataTest(t, ctx)

	var rowCount int
	err := conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	dataProtos := getTestOracleData()

	for _, proto := range dataProtos {
		data, err := entities.OracleDataFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		err = od.Add(data)
		require.NoError(t, err)
	}

	err = conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, len(dataProtos), rowCount)

	got, err := od.GetOracleDataBySpecID(ctx, "DEADBEEF", entities.Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(got))
}

func getTestOracleData() []*oraclespb.OracleData {
	return []*oraclespb.OracleData{
		{
			PubKeys: []string{"B105F00D", "BADDCAFE"},
			Data: []*oraclespb.Property{
				{
					Name:  "Ticker",
					Value: "USDBTC",
				},
			},
			MatchedSpecIds: []string{"CAFED00D"},
			BroadcastAt:    0,
		},
		{
			PubKeys: []string{"B105F00D", "BADDCAFE"},
			Data: []*oraclespb.Property{
				{
					Name:  "Ticker",
					Value: "USDETH",
				},
			},
			MatchedSpecIds: []string{"DEADBEEF"},
			BroadcastAt:    0,
		},
		{
			PubKeys: []string{"B105F00D", "BADDCAFE"},
			Data: []*oraclespb.Property{
				{
					Name:  "Ticker",
					Value: "USDETH",
				},
			},
			MatchedSpecIds: []string{"DEADBEEF"},
			BroadcastAt:    0,
		},
		{
			PubKeys: []string{"B105F00D", "BADDCAFE"},
			Data: []*oraclespb.Property{
				{
					Name:  "Ticker",
					Value: "USDSOL",
				},
			},
			MatchedSpecIds: []string{"DEADBAAD"},
			BroadcastAt:    0,
		},
	}
}
