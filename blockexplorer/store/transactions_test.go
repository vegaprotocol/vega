// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package store_test

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	"code.vegaprotocol.io/vega/blockexplorer/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	"code.vegaprotocol.io/vega/libs/config"
	"code.vegaprotocol.io/vega/logging"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"

	"github.com/cenkalti/backoff"
	tmTypes "github.com/cometbft/cometbft/abci/types"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		panic(fmt.Errorf("could not create temporary root directory for block_explorer tests: %w", err))
	}
	postgresRuntimePath = filepath.Join(tempDir, "sqlstore")
	err = os.Mkdir(postgresRuntimePath, fs.ModePerm)
	if err != nil {
		panic(fmt.Errorf("could not create temporary directory for postgres runtime: %w", err))
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

	blockSQL := `INSERT INTO blocks (height, chain_id, created_at) VALUES ($1, $2, $3) ON CONFLICT (height, chain_id) DO UPDATE SET created_at = EXCLUDED.created_at RETURNING rowid`
	resultSQL := `INSERT INTO tx_results (block_id, index, created_at, tx_hash, tx_result, submitter, cmd_type) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING rowid`

	for _, txr := range txResults {
		var blockID int64
		var ok bool

		if blockID, ok = blockIDs[txr.height]; !ok {
			require.NoError(t, conn.QueryRow(ctx, blockSQL, txr.height, "test-chain", txr.createdAt).Scan(&blockID))
			blockIDs[txr.height] = blockID
		}

		index := txr.index

		var rowID int64
		require.NoError(t, conn.QueryRow(ctx, resultSQL, blockID, index, txr.createdAt, txr.txHash, txr.txResult, txr.submitter, txr.cmdType).Scan(&rowID))

		row := entities.TxResultRow{
			RowID:       rowID,
			BlockHeight: txr.height,
			Index:       index,
			CreatedAt:   txr.createdAt,
			TxHash:      txr.txHash,
			TxResult:    txr.txResult,
			Submitter:   txr.submitter,
			CmdType:     txr.cmdType,
		}

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

	now := time.Now()
	txResults := []txResult{
		{
			height:    0,
			index:     1,
			createdAt: now,
			txHash:    "deadbeef01",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    0,
			index:     2,
			createdAt: now.Add(100),
			txHash:    "deadbeef02",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    1,
			index:     1,
			createdAt: now.Add(1 * time.Second),
			txHash:    "deadbeef11",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    2,
			index:     1,
			createdAt: now.Add(2 * time.Second),
			txHash:    "deadbeef21",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    2,
			index:     2,
			createdAt: now.Add(2*time.Second + 50),
			txHash:    "deadbeef22",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    2,
			index:     4,
			createdAt: now.Add(2*time.Second + 700),
			txHash:    "deadbeef24",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    3,
			index:     1,
			createdAt: now.Add(3 * time.Second),
			txHash:    "deadbeef31",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    4,
			index:     1,
			createdAt: now.Add(4 * time.Second),
			txHash:    "deadbeef41",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    5,
			index:     1,
			createdAt: now.Add(5 * time.Second),
			txHash:    "deadbeef51",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    6,
			index:     1,
			createdAt: now.Add(6 * time.Second),
			txHash:    "deadbeef61",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
	}

	return addTestTxResults(ctx, t, txResults...)
}

func TestStore_ListTransactions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), postgresServerTimeout)
	t.Cleanup(cancel)

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
		after := entities.TxCursor{
			BlockNumber: 2,
			TxIndex:     1,
		}
		got, err := s.ListTransactions(ctx, nil, nil, nil, nil, 2, &after, 0, nil)
		require.NoError(t, err)
		want := []*pb.Transaction{inserted[5], inserted[4]}
		assert.Equal(t, want, got)
	})

	t.Run("should return the transactions before the cursor when last is set", func(t *testing.T) {
		before := entities.TxCursor{
			BlockNumber: 2,
			TxIndex:     1,
		}
		got, err := s.ListTransactions(ctx, nil, nil, nil, nil, 0, nil, 2, &before)
		require.NoError(t, err)
		want := []*pb.Transaction{inserted[2], inserted[1]}
		assert.Equal(t, want, got)
	})

	t.Run("should return the transactions before the cursor when last is set", func(t *testing.T) {
		before := entities.TxCursor{
			BlockNumber: 5,
			TxIndex:     1,
		}
		after := entities.TxCursor{
			BlockNumber: 2,
			TxIndex:     2,
		}
		got, err := s.ListTransactions(ctx, nil, nil, nil, nil, 0, &after, 0, &before)
		require.NoError(t, err)
		want := []*pb.Transaction{inserted[7], inserted[6], inserted[5]}
		assert.Equal(t, want, got)
	})
}
