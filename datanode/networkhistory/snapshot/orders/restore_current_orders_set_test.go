package orders

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var (
	connectionSource *sqlstore.ConnectionSource
	sqlConfig        sqlstore.Config
	snapshotsDir     string
)

func TestMain(t *testing.M) {
	tmp, err := os.MkdirTemp("", "orders")
	if err != nil {
		panic(err)
	}
	postgresRuntimePath := filepath.Join(tmp, "sqlstore")
	defer os.RemoveAll(tmp)

	snapsTmp, err := os.MkdirTemp("", "snapshots")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(snapsTmp)

	snapshotCopyToPath := filepath.Join(snapsTmp, "snapshotsCopyTo")
	err = os.MkdirAll(snapshotCopyToPath, os.ModePerm)
	if err != nil {
		panic(fmt.Errorf("failed to create snapshots directory: %w", err))
	}

	databasetest.TestMain(t, func(config sqlstore.Config, source *sqlstore.ConnectionSource,
		postgresLog *bytes.Buffer,
	) {
		sqlConfig = config
		connectionSource = source
		snapshotsDir = snapshotCopyToPath
	}, postgresRuntimePath, sqlstore.EmbedMigrations)
}

func TestRestoreCurrentOrdersSet(t *testing.T) {
	ctx := context.Background()

	batcher := sqlstore.NewMapBatcher[entities.OrderKey, entities.Order](
		"orders",
		entities.OrderColumns)

	vegaTime := time.Now().Truncate(1 * time.Millisecond)

	orders := []entities.Order{
		createTestOrder("aa", vegaTime.Add(1*time.Second), 0, 0),
		createTestOrder("bb", vegaTime.Add(1*time.Second), 0, 1),

		createTestOrder("aa", vegaTime.Add(2*time.Second), 1, 0),
		createTestOrder("bb", vegaTime.Add(2*time.Second), 1, 1),

		// Add two versions for the same vega-time to ensure the update query picks the correct one to set as current
		createTestOrder("aa", vegaTime.Add(3*time.Second), 2, 0),

		createTestOrder("bb", vegaTime.Add(3*time.Second), 2, 1),
		createTestOrder("bb", vegaTime.Add(3*time.Second), 3, 2),

		createTestOrder("aa", vegaTime.Add(3*time.Second), 3, 3),
	}

	type queryResult struct {
		Id       entities.OrderID
		VegaTime time.Time
		Version  int
		SeqNum   int
		Current  bool
	}

	expectedResult := []queryResult{
		{"aa", vegaTime.Add(1 * time.Second), 0, 0, false},
		{"aa", vegaTime.Add(2 * time.Second), 1, 0, false},
		{"aa", vegaTime.Add(3 * time.Second), 2, 0, false},
		{"aa", vegaTime.Add(3 * time.Second), 3, 3, true},

		{"bb", vegaTime.Add(1 * time.Second), 0, 1, false},
		{"bb", vegaTime.Add(2 * time.Second), 1, 1, false},
		{"bb", vegaTime.Add(3 * time.Second), 2, 1, false},
		{"bb", vegaTime.Add(3 * time.Second), 3, 2, true},
	}

	for _, order := range orders {
		batcher.Add(order)
	}

	_, err := batcher.Flush(context.Background(), connectionSource.Connection)
	require.NoError(t, err)

	err = RestoreCurrentOrdersSet(ctx, connectionSource.Connection)
	require.NoError(t, err)

	connectionSource.Connection.QueryRow(ctx, "select current")

	rows, err := connectionSource.Connection.Query(context.Background(),
		"select id, vega_time, version, seq_num, current from orders order by id, vega_time, seq_num")

	require.NoError(t, err)

	results := []queryResult{}
	err = pgxscan.ScanAll(&results, rows)
	rows.Close()

	require.NoError(t, err)

	for i := 0; i < len(results); i++ {
		assert.Equal(t, expectedResult[i], results[i])
	}
}

func createTestOrder(id string, vegaTime time.Time, version int32, seqNum uint64) entities.Order {
	order := entities.Order{
		ID:              entities.OrderID(id),
		MarketID:        entities.MarketID("1B"),
		PartyID:         entities.PartyID("1A"),
		Side:            types.SideBuy,
		Price:           decimal.NewFromInt32(100),
		Size:            10,
		Remaining:       0,
		TimeInForce:     types.OrderTimeInForceGTC,
		Type:            types.OrderTypeLimit,
		Status:          types.OrderStatusFilled,
		Reference:       "ref1",
		Version:         version,
		PeggedOffset:    decimal.NewFromInt32(0),
		PeggedReference: types.PeggedReferenceMid,
		CreatedAt:       time.Now().Truncate(time.Microsecond),
		UpdatedAt:       time.Now().Add(5 * time.Second).Truncate(time.Microsecond),
		ExpiresAt:       time.Now().Add(10 * time.Second).Truncate(time.Microsecond),
		VegaTime:        vegaTime,
		SeqNum:          seqNum,
	}

	return order
}
