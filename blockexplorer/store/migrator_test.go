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
	"context"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	"code.vegaprotocol.io/vega/blockexplorer/store"
	"code.vegaprotocol.io/vega/libs/config"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"

	tmTypes "github.com/cometbft/cometbft/abci/types"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigratorMigrate(t *testing.T) {
	// first we need to populate the database with some test data
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	t.Cleanup(func() {
		cleanupTransactionsTest(ctx, t)
		cancel()
	})

	want := populateMigrationData(t, ctx)

	pgConfig, err := config.PostgresConnection{
		Host:      "",
		Port:      5432,
		Username:  "vega",
		Password:  "vega",
		Database:  "vega",
		SocketDir: postgresRuntimePath,
	}.ToPgxPoolConfig()
	require.NoError(t, err)

	pool, err := pgxpool.ConnectConfig(ctx, pgConfig)
	require.NoError(t, err)

	// Confirm that there's no data in the tx_results table first
	query := `SELECT rowid, block_height, index, created_at, tx_hash, tx_result, submitter, cmd_type
		FROM tx_results ORDER BY block_height desc, index desc`
	var rows []entities.TxResultRow
	require.NoError(t, pgxscan.Select(ctx, pool, &rows, query))

	assert.Len(t, rows, 0)

	migrator := store.NewMigrator(pool, true)

	err = migrator.Migrate()

	require.NoError(t, err)

	// now get the data from the new tx_results table and make sure that we have everything we expect
	require.NoError(t, pgxscan.Select(ctx, pool, &rows, query))

	sort.Slice(want, func(i, j int) bool {
		return want[i].Block > want[j].Block ||
			(want[i].Block == want[j].Block && want[i].Index > want[j].Index)
	})

	got := make([]*pb.Transaction, 0, len(rows))

	for _, row := range rows {
		r, e := row.ToProto()
		require.NoError(t, e)
		got = append(got, r)
	}

	require.Equal(t, want, got)

	// We want to check the old data has been removed
	sql := `select table_name from information_schema.tables where table_name = 'tx_results_old'`
	var tableName string
	require.Errorf(t, pool.QueryRow(context.Background(), sql).Scan(&tableName), "no rows in result set")
}

func populateMigrationData(t *testing.T, ctx context.Context) []*pb.Transaction {
	t.Helper()

	txr, err := (&tmTypes.TxResult{}).Marshal()
	require.NoError(t, err)
	now := time.Now()
	day := time.Hour * 24
	txns := []txResult{
		{
			height:    1,
			index:     1,
			createdAt: now,
			txHash:    "deadbeef01",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    2,
			index:     1,
			createdAt: now.Add(day),
			txHash:    "deadbeef02",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    3,
			index:     1,
			createdAt: now.Add(day * 2),
			txHash:    "deadbeef03",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    4,
			index:     1,
			createdAt: now.Add(day * 3),
			txHash:    "deadbeef04",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    5,
			index:     1,
			createdAt: now.Add(day * 4),
			txHash:    "deadbeef05",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    6,
			index:     1,
			createdAt: now.Add(day * 5),
			txHash:    "deadbeef06",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    7,
			index:     1,
			createdAt: now.Add(day * 6),
			txHash:    "deadbeef07",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    8,
			index:     1,
			createdAt: now.Add(day * 7),
			txHash:    "deadbeef08",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    9,
			index:     1,
			createdAt: now.Add(day * 8),
			txHash:    "deadbeef09",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
		{
			height:    10,
			index:     1,
			createdAt: now.Add(day * 9),
			txHash:    "deadbeef10",
			txResult:  txr,
			submitter: "TEST",
			cmdType:   "TEST",
		},
	}

	want := addTestTxResults(ctx, t, "tx_results_old", txns...)

	return want
}
