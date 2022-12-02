package orders

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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

func TestUpdateCurrentOrdersState(t *testing.T) {
	ctx := context.Background()

	batcher := sqlstore.NewMapBatcher[entities.OrderKey, entities.Order](
		"orders",
		entities.OrderColumns)

	vegaTime := time.Now()

	orders := []entities.Order{
		createTestOrder("aa", vegaTime.Add(1*time.Second), 0, 0),
		createTestOrder("aa", vegaTime.Add(2*time.Second), 1, 2),
		createTestOrder("aa", vegaTime.Add(3*time.Second), 2, 3),

		createTestOrder("bb", vegaTime.Add(4*time.Second), 0, 0),
		createTestOrder("bb", vegaTime.Add(5*time.Second), 1, 1),
		createTestOrder("bb", vegaTime.Add(6*time.Second), 2, 2),
	}

	for _, order := range orders {
		batcher.Add(order)
	}

	_, err := batcher.Flush(context.Background(), connectionSource.Connection)
	require.NoError(t, err)

	err = UpdateCurrentOrdersState(ctx, connectionSource.Connection)
	require.NoError(t, err)

	connectionSource.Connection.QueryRow(ctx, "select vega_time_to")

	rows, err := connectionSource.Connection.Query(context.Background(),
		"select vega_time, vega_time_to from orders")

	require.NoError(t, err)

	type queryResult struct {
		VegaTime   time.Time
		VegaTimeTo interface{}
	}

	results := []queryResult{}
	expectedResult := map[int64]interface{}{
		vegaTime.Add(1 * time.Second).UnixMicro(): vegaTime.Add(2 * time.Second),
		vegaTime.Add(2 * time.Second).UnixMicro(): vegaTime.Add(3 * time.Second),
		vegaTime.Add(3 * time.Second).UnixMicro(): pgtype.InfinityModifier(1),

		vegaTime.Add(4 * time.Second).UnixMicro(): vegaTime.Add(5 * time.Second),
		vegaTime.Add(5 * time.Second).UnixMicro(): vegaTime.Add(6 * time.Second),
		vegaTime.Add(6 * time.Second).UnixMicro(): pgtype.InfinityModifier(1),
	}

	err = pgxscan.ScanAll(&results, rows)
	rows.Close()
	require.NoError(t, err)
	for _, result := range results {
		expected := expectedResult[result.VegaTime.UnixMicro()]
		switch v := expected.(type) {
		case time.Time:
			timeTo := result.VegaTimeTo.(time.Time)
			assert.Equal(t, v.UnixMicro(), timeTo.UnixMicro())
		default:
			assert.Equal(t, v, result.VegaTimeTo)
		}
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
