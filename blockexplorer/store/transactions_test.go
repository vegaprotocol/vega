package store_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/config"
	"github.com/stretchr/testify/assert"
	tmTypes "github.com/tendermint/tendermint/abci/types"

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	"code.vegaprotocol.io/vega/blockexplorer/store"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"github.com/cenkalti/backoff"
	"github.com/jackc/pgx/v4"
)

var (
	postgresServerTimeout = time.Minute * 10
	postgresRuntimePath   string
	connectionSource      *sqlstore.ConnectionSource
)

func TestMain(m *testing.M) {
	log := logging.NewTestLogger()

	tempDir, err := os.MkdirTemp("", "block_explorer")
	if err != nil {
		panic(err)
	}
	postgresRuntimePath = filepath.Join(tempDir, "sqlstore")
	err = os.Mkdir(postgresRuntimePath, fs.ModePerm)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(postgresRuntimePath)

	testDBPort := 5432
	testDBHost := ""
	sqlConfig := databasetest.NewTestConfig(testDBPort, testDBHost, postgresRuntimePath)
	postgresLog := &bytes.Buffer{}
	embeddedPostgres, err := sqlstore.StartEmbeddedPostgres(log, sqlConfig, postgresRuntimePath, postgresLog)
	if err != nil {
		log.Errorf("failed to start postgres: %s", postgresLog.String())
		panic(err)
	}

	log.Infof("Test DB Socket Directory: %s", postgresRuntimePath)

	// Make sure the database has started before we run the tests.
	ctx, cancel := context.WithTimeout(context.Background(), postgresServerTimeout)

	op := func() error {
		connStr := sqlConfig.ConnectionConfig.GetConnectionString()
		conn, err := pgx.Connect(ctx, connStr)
		if err != nil {
			return err
		}

		return conn.Ping(ctx)
	}

	if err := backoff.Retry(op, backoff.NewExponentialBackOff()); err != nil {
		cancel()
		panic(err)
	}

	cancel()
	connectionSource, err = sqlstore.NewTransactionalConnectionSource(log, sqlConfig.ConnectionConfig)
	if err != nil {
		panic(err)
	}
	defer embeddedPostgres.Stop()

	if err = sqlstore.WipeDatabaseAndMigrateSchemaToLatestVersion(log, sqlConfig.ConnectionConfig, store.EmbedMigrations, false); err != nil {
		log.Errorf("failed to wipe database and migrate schema, dumping postgres log:\n %s", postgresLog.String())
		panic(err)
	}

	code := m.Run()
	os.Exit(code)
}

type txResult struct {
	height    int64
	index     int64
	createdAt time.Time
	txHash    string
	txResult  []byte
	submitter string
	cmdType   string
}

func addTestTxResults(ctx context.Context, t *testing.T, txResults ...txResult) []*pb.Transaction {
	t.Helper()
	conn := connectionSource.Connection
	rows := make([]*pb.Transaction, 0, len(txResults))
	blockIDs := make(map[int64]int64)

	for _, txr := range txResults {
		var blockID int64
		var ok bool

		blockSQL := `INSERT INTO blocks (height, chain_id, created_at) VALUES ($1, $2, $3) ON CONFLICT (height, chain_id) DO UPDATE SET created_at = EXCLUDED.created_at RETURNING rowid`
		resultSQL := `INSERT INTO tx_results (block_id, index, created_at, tx_hash, tx_result, submitter, cmd_type) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING rowid`

		if blockID, ok = blockIDs[txr.height]; !ok {
			err := conn.QueryRow(ctx, blockSQL, txr.height, "test-chain", txr.createdAt).Scan(&blockID)
			require.NoError(t, err)
			blockIDs[txr.height] = blockID
		}

		row := entities.TxResultRow{
			BlockID:   blockID,
			Index:     txr.index,
			CreatedAt: txr.createdAt,
			TxHash:    txr.txHash,
			TxResult:  txr.txResult,
			Submitter: txr.submitter,
			CmdType:   txr.cmdType,
		}

		var rowID int64

		err := conn.QueryRow(ctx, resultSQL, blockID, txr.index, txr.createdAt, txr.txHash, txr.txResult, txr.submitter, txr.cmdType).Scan(&rowID)
		require.NoError(t, err)
		row.RowID = rowID

		proto, err := row.ToProto()
		require.NoError(t, err)

		rows = append(rows, proto)
	}

	return rows
}

func setupTestTransactions(ctx context.Context, t *testing.T) []*pb.Transaction {
	t.Helper()

	txr, err := (&tmTypes.TxResult{}).Marshal()
	require.NoError(t, err)

	txResults := []txResult{
		{
			height:    1,
			index:     1,
			createdAt: time.Date(2023, 7, 10, 9, 0, 0, 100, time.UTC),
			txHash:    "deadbeef01",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    1,
			index:     2,
			createdAt: time.Date(2023, 7, 10, 9, 0, 0, 200, time.UTC),
			txHash:    "deadbeef02",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    2,
			index:     1,
			createdAt: time.Date(2023, 7, 10, 9, 0, 1, 100, time.UTC),
			txHash:    "deadbeef03",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    3,
			index:     1,
			createdAt: time.Date(2023, 7, 10, 9, 0, 2, 100, time.UTC),
			txHash:    "deadbeef04",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    3,
			index:     2,
			createdAt: time.Date(2023, 7, 10, 9, 0, 2, 150, time.UTC),
			txHash:    "deadbeef05",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    3,
			index:     4,
			createdAt: time.Date(2023, 7, 10, 9, 0, 2, 800, time.UTC),
			txHash:    "deadbeef06",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    4,
			index:     1,
			createdAt: time.Date(2023, 7, 10, 9, 0, 3, 100, time.UTC),
			txHash:    "deadbeef07",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    5,
			index:     1,
			createdAt: time.Date(2023, 7, 10, 9, 0, 4, 100, time.UTC),
			txHash:    "deadbeef08",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    6,
			index:     1,
			createdAt: time.Date(2023, 7, 10, 9, 0, 5, 100, time.UTC),
			txHash:    "deadbeef09",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    7,
			index:     1,
			createdAt: time.Date(2023, 7, 10, 9, 0, 6, 100, time.UTC),
			txHash:    "deadbeef10",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
	}

	return addTestTxResults(ctx, t, txResults...)
}

func TestStore_ListTransactions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), postgresServerTimeout)
	defer cancel()

	inserted := setupTestTransactions(ctx, t)

	s := store.MustNewStore(store.Config{
		Postgres: config.PostgresConnection{
			Host:      "",
			Port:      5432,
			Username:  "vega",
			Password:  "vega",
			Database:  "vega",
			SocketDir: postgresRuntimePath,
		},
	}, logging.NewTestLogger())

	t.Run("should return the most recent transactions when first is set without cursor", func(t *testing.T) {
		got, err := s.ListTransactions(ctx, nil, nil, nil, nil, 2, nil, 0, nil)
		require.NoError(t, err)
		want := []*pb.Transaction{inserted[9], inserted[8]}
		assert.Equal(t, want, got)
	})

	t.Run("should return the oldest transactions when last is set without cursor", func(t *testing.T) {
		got, err := s.ListTransactions(ctx, nil, nil, nil, nil, 0, nil, 2, nil)
		require.NoError(t, err)
		want := []*pb.Transaction{inserted[1], inserted[0]}
		assert.Equal(t, want, got)
	})

	t.Run("should return the transactions after the cursor when first is set", func(t *testing.T) {
		first := entities.TxCursor{
			BlockNumber: 6,
			TxIndex:     1,
		}
		got, err := s.ListTransactions(ctx, nil, nil, nil, nil, 2, &first, 0, nil)
		require.NoError(t, err)
		want := []*pb.Transaction{inserted[7], inserted[6]}
		assert.Equal(t, want, got)
	})

	t.Run("should return the transactions before the cursor when last is set", func(t *testing.T) {
		first := entities.TxCursor{
			BlockNumber: 2,
			TxIndex:     1,
		}
		got, err := s.ListTransactions(ctx, nil, nil, nil, nil, 2, &first, 0, nil)
		require.NoError(t, err)
		want := []*pb.Transaction{inserted[1], inserted[0]}
		assert.Equal(t, want, got)
	})
}
