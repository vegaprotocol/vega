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

package networkhistory_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/cmd/data-node/commands/start"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	config2 "code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	"code.vegaprotocol.io/vega/logging"
	eventsv1 "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	snapshotInterval     = int64(1000)
	chainID              = "testnet-001"
	compressedEventsFile = "testdata/smoketest_to_block_5000.evts.gz"
	numSnapshots         = 6
	testMigrationSQL     = "testdata/testmigration.sql"
)

var (
	sqlConfig              sqlstore.Config
	networkHistoryConnPool *pgxpool.Pool

	fromEventHashes             []string
	fromEventsDatabaseSummaries []databaseSummary

	fromEventsIntervalToHistoryTableDelta []map[string]tableDataSummary

	snapshotsBackupDir string
	eventsDir          string
	eventsFile         string

	goldenSourceHistorySegment map[int64]segment.Full

	expectedHistorySegmentsFromHeights = []int64{1, 1001, 2001, 2501, 3001, 4001}
	expectedHistorySegmentsToHeights   = []int64{1000, 2000, 2500, 3000, 4000, 5000}

	networkHistoryStore *store.Store

	postgresLog *bytes.Buffer

	testMigrationsDir       string
	highestMigrationNumber  int64
	testMigrationVersionNum int64
	sqlFs                   fs.FS
)

func TestMain(t *testing.M) {
	outerCtx, cancelOuterCtx := context.WithCancel(context.Background())
	defer cancelOuterCtx()

	// because we have a ton of panics in here:
	defer func() {
		if r := recover(); r != nil {
			cancelOuterCtx()
			panic(r) // propagate panic
		}
	}()
	testMigrationVersionNum, sqlFs = setupTestSQLMigrations()
	highestMigrationNumber = testMigrationVersionNum - 1

	var err error
	snapshotsBackupDir, err = os.MkdirTemp("", "snapshotbackup")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(snapshotsBackupDir)

	eventsDir, err = os.MkdirTemp("", "eventsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(eventsDir)

	log := logging.NewTestLogger()

	eventsFile = filepath.Join(eventsDir, "smoketest_to_block_5000_or_above.evts")
	decompressEventFile()

	tempDir, err := os.MkdirTemp("", "networkhistory")
	if err != nil {
		panic(err)
	}
	postgresRuntimePath := filepath.Join(tempDir, "sqlstore")
	defer os.RemoveAll(tempDir)

	networkHistoryHome, err := os.MkdirTemp("", "networkhistoryhome")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(networkHistoryHome)

	defer func() {
		if networkHistoryConnPool != nil {
			networkHistoryConnPool.Close()
		}
	}()

	exitCode := databasetest.TestMain(t, func(config sqlstore.Config, source *sqlstore.ConnectionSource,
		pgLog *bytes.Buffer,
	) {
		sqlConfig = config
		log.Infof("DB Connection String: ", sqlConfig.ConnectionConfig.GetConnectionString())

		pool, err := sqlstore.CreateConnectionPool(sqlConfig.ConnectionConfig)
		if err != nil {
			panic(fmt.Errorf("failed to create connection pool: %w", err))
		}
		networkHistoryConnPool = pool

		postgresLog = pgLog

		emptyDatabaseAndSetSchemaVersion(highestMigrationNumber)

		// Do initial run to get the expected state of the datanode from just event playback
		ctx, cancel := context.WithCancel(outerCtx)
		defer cancel()

		snapshotCopyToPath := filepath.Join(networkHistoryHome, "snapshotsCopyTo")

		snapshotService := setupSnapshotService(snapshotCopyToPath)

		var snapshots []segment.Unpublished

		ctxWithCancel, cancelFn := context.WithCancel(ctx)
		defer cancelFn()

		evtSource := newTestEventSourceWithProtocolUpdateMessage()

		pus := service.NewProtocolUpgrade(nil, log)
		puh := networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context, chainID string,
			toHeight int64,
		) error {
			ss, err := snapshotService.CreateSnapshot(ctx, chainID, toHeight)
			if err != nil {
				panic(fmt.Errorf("failed to create snapshot: %w", err))
			}

			waitForSnapshotToComplete(ss)

			snapshots = append(snapshots, ss)

			md5Hash, err := Md5Hash(ss.UnpublishedSnapshotDataDirectory())
			if err != nil {
				panic(fmt.Errorf("failed to get snapshot hash:%w", err))
			}

			fromEventHashes = append(fromEventHashes, md5Hash)

			updateAllContinuousAggregateData(ctx)
			summary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

			fromEventsDatabaseSummaries = append(fromEventsDatabaseSummaries, summary)

			return nil
		})

		preUpgradeBroker, err := setupSQLBroker(ctx, sqlConfig, snapshotService,
			func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
				if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
					lastSnapshot, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
					if err != nil {
						panic(fmt.Errorf("failed to create snapshot:%w", err))
					}

					waitForSnapshotToComplete(lastSnapshot)
					snapshots = append(snapshots, lastSnapshot)
					md5Hash, err := Md5Hash(lastSnapshot.UnpublishedSnapshotDataDirectory())
					if err != nil {
						panic(fmt.Errorf("failed to get snapshot hash:%w", err))
					}

					fromEventHashes = append(fromEventHashes, md5Hash)

					updateAllContinuousAggregateData(ctx)
					summary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

					fromEventsDatabaseSummaries = append(fromEventsDatabaseSummaries, summary)

					if lastCommittedBlockHeight == numSnapshots*snapshotInterval {
						cancelFn()
					}
				}
			},
			evtSource, puh)
		if err != nil {
			panic(fmt.Errorf("failed to setup pre-protocol upgrade sqlbroker:%w", err))
		}

		err = preUpgradeBroker.Receive(ctxWithCancel)
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(fmt.Errorf("failed to process events:%w", err))
		}

		protocolUpgradeStarted := pus.GetProtocolUpgradeStarted()
		if !protocolUpgradeStarted {
			panic("expected protocol upgrade to have started")
		}

		// Here after exit of the broker because of protocol upgrade, we simulate a restart of the node by recreating
		// the broker.
		// First simulate a schema update
		err = migrateUpToDatabaseVersion(testMigrationVersionNum)
		if err != nil {
			panic(err)
		}

		pus = service.NewProtocolUpgrade(nil, log)
		nonInterceptPuh := networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context,
			chainID string, toHeight int64,
		) error {
			return nil
		})

		postUpgradeBroker, err := setupSQLBroker(ctx, sqlConfig, snapshotService,
			func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
				if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
					lastSnapshot, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
					if err != nil {
						panic(fmt.Errorf("failed to create snapshot:%w", err))
					}

					waitForSnapshotToComplete(lastSnapshot)
					snapshots = append(snapshots, lastSnapshot)
					md5Hash, err := Md5Hash(lastSnapshot.UnpublishedSnapshotDataDirectory())
					if err != nil {
						panic(fmt.Errorf("failed to get snapshot hash:%w", err))
					}

					fromEventHashes = append(fromEventHashes, md5Hash)

					updateAllContinuousAggregateData(ctx)
					summary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

					fromEventsDatabaseSummaries = append(fromEventsDatabaseSummaries, summary)

					if lastCommittedBlockHeight == (numSnapshots-1)*snapshotInterval {
						cancelFn()
					}
				}
			},
			evtSource, nonInterceptPuh)
		if err != nil {
			panic(fmt.Errorf("failed to setup post protocol upgrade sqlbroker:%w", err))
		}

		err = postUpgradeBroker.Receive(ctxWithCancel)
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(fmt.Errorf("failed to process events:%w", err))
		}

		if len(fromEventHashes) != numSnapshots {
			panic(fmt.Errorf("expected 5 snapshots, got %d", len(fromEventHashes)))
		}

		if len(fromEventsDatabaseSummaries) != numSnapshots {
			panic(fmt.Errorf("expected %d database summaries, got %d", numSnapshots, len(fromEventHashes)))
		}

		fromEventsIntervalToHistoryTableDelta = getSnapshotIntervalToHistoryTableDeltaSummary(ctx, sqlConfig.ConnectionConfig,
			expectedHistorySegmentsFromHeights, expectedHistorySegmentsToHeights)

		if len(fromEventsIntervalToHistoryTableDelta) != numSnapshots {
			panic(fmt.Errorf("expected %d history table deltas, got %d", numSnapshots, len(fromEventHashes)))
		}

		// Network history store setup
		storeCfg := store.NewDefaultConfig()
		storeCfg.SwarmKeyOverride = uuid.NewV4().String()

		storeCfg.SwarmPort = databasetest.GetNextFreePort()

		storeLog := logging.NewTestLogger()
		storeLog.SetLevel(logging.InfoLevel)
		networkHistoryStore, err = store.New(outerCtx, storeLog, chainID, storeCfg, networkHistoryHome, 33)
		if err != nil {
			panic(err)
		}

		datanodeConfig := config2.NewDefaultConfig()
		cfg := networkhistory.NewDefaultConfig()

		_, err = networkhistory.New(outerCtx, log, chainID, cfg, networkHistoryConnPool, snapshotService,
			networkHistoryStore, datanodeConfig.API.Port, snapshotCopyToPath)

		if err != nil {
			panic(err)
		}

		startTime := time.Now()
		timeout := 1 * time.Minute

		for {
			if time.Now().After(startTime.Add(timeout)) {
				panic(fmt.Sprintf("history not found in network store after %s", timeout))
			}

			time.Sleep(10 * time.Millisecond)

			storedSegments, err := networkHistoryStore.ListAllIndexEntriesOldestFirst()
			if err != nil {
				panic(err)
			}

			goldenSourceHistorySegment = map[int64]segment.Full{}
			for _, storedSegment := range storedSegments {
				goldenSourceHistorySegment[storedSegment.HeightTo] = storedSegment
			}

			allExpectedSegmentsFound := true
			for _, expected := range expectedHistorySegmentsToHeights {
				if _, ok := goldenSourceHistorySegment[expected]; !ok {
					allExpectedSegmentsFound = false
					break
				}
			}

			if allExpectedSegmentsFound {
				break
			}
		}

		// For the same events file and block height the history segment ID should be the same across all OS/Arch
		// If the events file is updated, schema changes, or snapshot height changed this will need updating
		// Easiest way to update is to put a breakpoint here or inspect the log for the lines printed below
		log.Info("expected history segment IDs:")
		log.Infof("%s", goldenSourceHistorySegment[1000].HistorySegmentID)
		log.Infof("%s", goldenSourceHistorySegment[2000].HistorySegmentID)
		log.Infof("%s", goldenSourceHistorySegment[2500].HistorySegmentID)
		log.Infof("%s", goldenSourceHistorySegment[3000].HistorySegmentID)
		log.Infof("%s", goldenSourceHistorySegment[4000].HistorySegmentID)
		log.Infof("%s", goldenSourceHistorySegment[5000].HistorySegmentID)

		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[1000].HistorySegmentID, "QmVEJnhB2YSPJ8n9GALHocxvxSWu7AJwDQrhXA8ungMVi2", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[2000].HistorySegmentID, "QmPcQW2sZrqmig6eCaeoqzZNBun93cJ5cVuDjwi3TG7M98", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[2500].HistorySegmentID, "QmNkv1Ljxd8o6xMM1D9DiSdKKPrJdPGvEv1q71bcYQ4vUX", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[3000].HistorySegmentID, "QmdBCioTj7yetkUv7Ua7Kr9w6Um4AWZ54Zf13MtsXyir4n", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[4000].HistorySegmentID, "QmPoHmjy59wVgBCMBUZu4WKyQgJJNuUPT3KwQeXEhBhwr7", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[5000].HistorySegmentID, "QmUfbnQ87tZkczZ4BC297g8zxf7P6FbtoeDhzPVysQwAZT", snapshots)
	}, postgresRuntimePath, sqlFs)

	if exitCode != 0 {
		log.Errorf("One or more tests failed, dumping postgres log:\n%s", postgresLog.String())
	}
}

func updateAllContinuousAggregateData(ctx context.Context) {
	blockspan, err := sqlstore.GetDatanodeBlockSpan(ctx, networkHistoryConnPool)
	if err != nil {
		panic(err)
	}

	err = snapshot.UpdateContinuousAggregateDataFromHighWaterMark(ctx, networkHistoryConnPool, blockspan.ToHeight)
	if err != nil {
		panic(err)
	}
}

func TestLoadingDataFetchedAsynchronously(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logging.NewTestLogger()

	require.NoError(t, networkHistoryStore.ResetIndex())
	emptyDatabaseAndSetSchemaVersion(highestMigrationNumber)

	snapshotCopyToPath := t.TempDir()
	snapshotService := setupSnapshotService(snapshotCopyToPath)

	fetched, err := fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[4000].HistorySegmentID, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), fetched)

	networkhistoryService := setupNetworkHistoryService(ctx, log, snapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.NoError(t, err)
	assert.Equal(t, int64(3001), loaded.LoadedFromHeight)
	assert.Equal(t, int64(4000), loaded.LoadedToHeight)

	dbSummary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[4].currentTableSummaries, dbSummary.currentTableSummaries)

	// Run events to height 5000
	ctxWithCancel, cancelFn := context.WithCancel(ctx)
	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	pus := service.NewProtocolUpgrade(nil, log)
	puh := networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context, chainID string,
		toHeight int64,
	) error {
		return nil
	})

	var md5Hash string
	broker, err := setupSQLBroker(ctx, sqlConfig, snapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				ss, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
				require.NoError(t, err)

				waitForSnapshotToComplete(ss)

				md5Hash, err = Md5Hash(ss.UnpublishedSnapshotDataDirectory())
				require.NoError(t, err)

				fromEventHashes = append(fromEventHashes, md5Hash)
			}

			if lastCommittedBlockHeight == 5000 {
				cancelFn()
			}
		},
		evtSource, puh)
	require.NoError(t, err)

	err = broker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	require.Equal(t, fromEventHashes[5], md5Hash)

	networkhistoryService.PublishSegments(ctx)

	// Now simulate the situation where the previous history segments were fetched asynchronously during event processing
	// and full history is then subsequently loaded
	emptyDatabaseAndSetSchemaVersion(0)

	fetched, err = fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[3000].HistorySegmentID, 3000)
	require.NoError(t, err)
	require.Equal(t, int64(3000), fetched)

	segments, err = networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	segmentsInRange, err := segments.ContiguousHistoryInRange(1, 5000)
	require.NoError(t, err)
	loaded, err = networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, segmentsInRange, sqlConfig.ConnectionConfig, false, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1), loaded.LoadedFromHeight)
	assert.Equal(t, int64(5000), loaded.LoadedToHeight)

	dbSummary = getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[5].currentTableSummaries, dbSummary.currentTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[5].historyTableSummaries, dbSummary.historyTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[5].caggSummaries, dbSummary.caggSummaries)
}

func TestRestoringNodeThatAlreadyContainsData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logging.NewTestLogger()

	require.NoError(t, networkHistoryStore.ResetIndex())
	emptyDatabaseAndSetSchemaVersion(highestMigrationNumber)

	snapshotCopyToPath := t.TempDir()
	snapshotService := setupSnapshotService(snapshotCopyToPath)

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	pus := service.NewProtocolUpgrade(nil, log)
	puh := networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context, chainID string,
		toHeight int64,
	) error {
		return nil
	})

	// Run events to height 1800

	broker, err := setupSQLBroker(ctx, sqlConfig, snapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight == 1800 {
				cancelFn()
			}
		},
		evtSource, puh)
	require.NoError(t, err)

	err = broker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	fetched, err := fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[4000].HistorySegmentID, 3000)
	require.NoError(t, err)
	require.Equal(t, int64(3000), fetched)

	snapshotCopyToPath = t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1801), loaded.LoadedFromHeight)
	assert.Equal(t, int64(4000), loaded.LoadedToHeight)

	dbSummary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[4].currentTableSummaries, dbSummary.currentTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[4].historyTableSummaries, dbSummary.historyTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[4].caggSummaries, dbSummary.caggSummaries)

	// Run events to height 5000
	ctxWithCancel, cancelFn = context.WithCancel(ctx)
	evtSource = newTestEventSourceWithProtocolUpdateMessage()

	pus = service.NewProtocolUpgrade(nil, log)
	puh = networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context, chainID string,
		toHeight int64,
	) error {
		return nil
	})

	var md5Hash string
	broker, err = setupSQLBroker(ctx, sqlConfig, snapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				ss, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
				require.NoError(t, err)

				waitForSnapshotToComplete(ss)

				md5Hash, err = Md5Hash(ss.UnpublishedSnapshotDataDirectory())
				require.NoError(t, err)

				fromEventHashes = append(fromEventHashes, md5Hash)
			}

			if lastCommittedBlockHeight == 5000 {
				cancelFn()
			}
		},
		evtSource, puh)
	require.NoError(t, err)

	err = broker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	require.Equal(t, fromEventHashes[5], md5Hash)

	dbSummary = getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[5].currentTableSummaries, dbSummary.currentTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[5].historyTableSummaries, dbSummary.historyTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[5].caggSummaries, dbSummary.caggSummaries)
}

func TestRestoringNodeWithDataOlderAndNewerThanItContainsLoadsTheNewerData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, networkHistoryStore.ResetIndex())

	log := logging.NewTestLogger()

	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	emptyDatabaseAndSetSchemaVersion(0)

	historySegment := goldenSourceHistorySegment[4000]

	blocksFetched, err := fetchBlocks(ctx, log, networkHistoryStore, historySegment.HistorySegmentID, 1)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), blocksFetched)
	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)

	require.NoError(t, err)

	assert.Equal(t, int64(3001), loaded.LoadedFromHeight)
	assert.Equal(t, int64(4000), loaded.LoadedToHeight)

	// Now try to load in history from 0 to 5000
	require.NoError(t, networkHistoryStore.ResetIndex())
	snapshotCopyToPath = t.TempDir()
	inputSnapshotService = setupSnapshotService(snapshotCopyToPath)

	historySegment = goldenSourceHistorySegment[5000]

	blocksFetched, err = fetchBlocks(ctx, log, networkHistoryStore, historySegment.HistorySegmentID, 5000)
	require.NoError(t, err)

	assert.Equal(t, int64(5000), blocksFetched)
	networkhistoryService = setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err = networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err = segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	result, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.Nil(t, err)

	assert.Equal(t, int64(4001), result.LoadedFromHeight)
	assert.Equal(t, int64(5000), result.LoadedToHeight)

	span, err := sqlstore.GetDatanodeBlockSpan(ctx, networkHistoryConnPool)
	require.Nil(t, err)

	assert.Equal(t, int64(3001), span.FromHeight)
	assert.Equal(t, int64(5000), span.ToHeight)
}

func TestRestoringNodeWithHistoryOnlyFromBeforeTheNodesOldestBlockFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, networkHistoryStore.ResetIndex())

	log := logging.NewTestLogger()

	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	emptyDatabaseAndSetSchemaVersion(0)

	historySegment := goldenSourceHistorySegment[4000]

	blocksFetched, err := fetchBlocks(ctx, log, networkHistoryStore, historySegment.HistorySegmentID, 1)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), blocksFetched)
	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.NoError(t, err)

	assert.Equal(t, int64(3001), loaded.LoadedFromHeight)
	assert.Equal(t, int64(4000), loaded.LoadedToHeight)

	// Now try to load in history from 1000 to 2000
	require.NoError(t, networkHistoryStore.ResetIndex())
	snapshotCopyToPath = t.TempDir()
	inputSnapshotService = setupSnapshotService(snapshotCopyToPath)

	historySegment = goldenSourceHistorySegment[1000]

	blocksFetched, err = fetchBlocks(ctx, log, networkHistoryStore, historySegment.HistorySegmentID, 1)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), blocksFetched)
	networkhistoryService = setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err = networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err = segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	_, err = networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.NotNil(t, err)
}

func TestRestoringNodeWithExistingDataFailsWhenLoadingWouldResultInNonContiguousHistory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logging.NewTestLogger()

	require.NoError(t, networkHistoryStore.ResetIndex())
	emptyDatabaseAndSetSchemaVersion(highestMigrationNumber)

	snapshotCopyToPath := t.TempDir()
	snapshotService := setupSnapshotService(snapshotCopyToPath)

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	pus := service.NewProtocolUpgrade(nil, log)
	puh := networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context, chainID string,
		toHeight int64,
	) error {
		return nil
	})

	// Run events to height 1800

	broker, err := setupSQLBroker(ctx, sqlConfig, snapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight == 1800 {
				cancelFn()
			}
		},
		evtSource, puh)
	require.NoError(t, err)

	err = broker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	// Now fetch history but not enough to form a contiguous history with the existing data

	fetched, err := fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[4000].HistorySegmentID, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(2000), fetched)

	snapshotCopyToPath = t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	_, err = networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.NotNil(t, err)
}

func TestRestoringFromDifferentHeightsWithFullHistory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, networkHistoryStore.ResetIndex())

	log := logging.NewTestLogger()

	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	for i := int64(0); i < numSnapshots; i++ {
		emptyDatabaseAndSetSchemaVersion(0)
		fromHeight := expectedHistorySegmentsFromHeights[i]
		toHeight := expectedHistorySegmentsToHeights[i]

		historySegment := goldenSourceHistorySegment[toHeight]

		expectedBlocks := toHeight - fromHeight + 1
		blocksFetched, err := fetchBlocks(ctx, log, networkHistoryStore, historySegment.HistorySegmentID, expectedBlocks)
		require.NoError(t, err)

		assert.Equal(t, expectedBlocks, blocksFetched)
		networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
		segments, err := networkhistoryService.ListAllHistorySegments()
		require.NoError(t, err)

		chunk, err := segments.MostRecentContiguousHistory()
		require.NoError(t, err)

		loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
		require.NoError(t, err)

		assert.Equal(t, int64(1), loaded.LoadedFromHeight)
		assert.Equal(t, toHeight, loaded.LoadedToHeight)

		dbSummary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
		assertSummariesAreEqual(t, fromEventsDatabaseSummaries[i].currentTableSummaries, dbSummary.currentTableSummaries)
		assertSummariesAreEqual(t, fromEventsDatabaseSummaries[i].historyTableSummaries, dbSummary.historyTableSummaries)
		assertSummariesAreEqual(t, fromEventsDatabaseSummaries[i].caggSummaries, dbSummary.caggSummaries)
	}
}

func TestRestoreFromPartialHistoryAndProcessEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, networkHistoryStore.ResetIndex())

	var err error
	log := logging.NewTestLogger()

	emptyDatabaseAndSetSchemaVersion(0)

	fetched, err := fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[3000].HistorySegmentID, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), fetched)

	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.NoError(t, err)
	assert.Equal(t, int64(2001), loaded.LoadedFromHeight)
	assert.Equal(t, int64(3000), loaded.LoadedToHeight)

	connSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), sqlConfig.ConnectionConfig)
	require.NoError(t, err)
	defer connSource.Close()

	evtSource, err := newTestEventSource(func(events.Event, chan<- events.Event) {})
	require.NoError(t, err)

	pus := service.NewProtocolUpgrade(nil, log)
	puh := networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context,
		chainID string, toHeight int64,
	) error {
		return nil
	})

	// Play events from 3001 to 4000
	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	var ss segment.Unpublished
	var newSnapshotFileHashAt4000 string

	outputSnapshotService := setupSnapshotService(t.TempDir())
	sqlBroker, err := setupSQLBroker(ctx, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				ss, err = service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
				require.NoError(t, err)
				waitForSnapshotToComplete(ss)

				if lastCommittedBlockHeight == 4000 {
					newSnapshotFileHashAt4000, err = Md5Hash(ss.UnpublishedSnapshotDataDirectory())
					require.NoError(t, err)
				}

				if lastCommittedBlockHeight == 5000 {
					cancelFn()
				}
			}
		},
		evtSource, puh)
	require.NoError(t, err)

	err = sqlBroker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	assert.Equal(t, fromEventHashes[4], newSnapshotFileHashAt4000)

	historyTableDelta := getSnapshotIntervalToHistoryTableDeltaSummary(ctx, sqlConfig.ConnectionConfig,
		expectedHistorySegmentsFromHeights, expectedHistorySegmentsToHeights)

	for i := 2; i < 5; i++ {
		assertSummariesAreEqual(t, fromEventsIntervalToHistoryTableDelta[i], historyTableDelta[i])
	}

	assertIntervalHistoryIsEmpty(t, historyTableDelta, 0)
	assertIntervalHistoryIsEmpty(t, historyTableDelta, 1)
}

func TestRestoreFromFullHistorySnapshotAndProcessEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, networkHistoryStore.ResetIndex())

	var err error
	log := logging.NewTestLogger()

	emptyDatabaseAndSetSchemaVersion(0)

	fetched, err := fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[2000].HistorySegmentID, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(2000), fetched)

	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, false, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1), loaded.LoadedFromHeight)
	assert.Equal(t, int64(2000), loaded.LoadedToHeight)

	connSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), sqlConfig.ConnectionConfig)
	require.NoError(t, err)
	defer connSource.Close()

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	var snapshotFileHashAfterReloadAt2000AndEventReplayTo3000 string
	outputSnapshotService := setupSnapshotService(t.TempDir())

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	puh := networkhistory.NewProtocolUpgradeHandler(log, service.NewProtocolUpgrade(nil, log), evtSource,
		func(ctx context.Context, chainID string, toHeight int64) error {
			return networkhistoryService.CreateAndPublishSegment(ctx, chainID, toHeight)
		})

	var lastCommittedBlockHeight int64
	sqlBroker, err := setupSQLBroker(ctx, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, blockHeight int64, snapshotTaken bool) {
			lastCommittedBlockHeight = blockHeight
		},
		evtSource, puh,
	)
	require.NoError(t, err)

	err = sqlBroker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(2500), lastCommittedBlockHeight)

	err = migrateUpToDatabaseVersion(testMigrationVersionNum)
	require.NoError(t, err)

	// After protocol upgrade restart the broker
	sqlBroker, err = setupSQLBroker(ctx, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				if lastCommittedBlockHeight == 3000 {
					ss, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
					require.NoError(t, err)
					waitForSnapshotToComplete(ss)

					snapshotFileHashAfterReloadAt2000AndEventReplayTo3000, err = Md5Hash(ss.UnpublishedSnapshotDataDirectory())
					require.NoError(t, err)
					cancelFn()
				}
			}
		},
		evtSource, networkhistory.NewProtocolUpgradeHandler(log, service.NewProtocolUpgrade(nil, log), evtSource,
			func(ctx context.Context, chainID string, toHeight int64) error {
				return nil
			}),
	)
	require.NoError(t, err)

	err = sqlBroker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	require.Equal(t, fromEventHashes[3], snapshotFileHashAfterReloadAt2000AndEventReplayTo3000)

	updateAllContinuousAggregateData(ctx)

	databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000 := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[3].currentTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.currentTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[3].historyTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.historyTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[3].caggSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.caggSummaries)
}

func TestRestoreFromFullHistorySnapshotWithIndexesAndOrderTriggersAndProcessEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, networkHistoryStore.ResetIndex())

	var err error
	log := logging.NewTestLogger()

	emptyDatabaseAndSetSchemaVersion(0)

	fetched, err := fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[2000].HistorySegmentID, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(2000), fetched)

	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)
	segments, err := networkhistoryService.ListAllHistorySegments()
	require.NoError(t, err)

	chunk, err := segments.MostRecentContiguousHistory()
	require.NoError(t, err)

	loaded, err := networkhistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunk, sqlConfig.ConnectionConfig, true, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1), loaded.LoadedFromHeight)
	assert.Equal(t, int64(2000), loaded.LoadedToHeight)

	connSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), sqlConfig.ConnectionConfig)
	require.NoError(t, err)
	defer connSource.Close()

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	var snapshotFileHashAfterReloadAt2000AndEventReplayTo3000 string
	outputSnapshotService := setupSnapshotService(t.TempDir())

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	puh := networkhistory.NewProtocolUpgradeHandler(log, service.NewProtocolUpgrade(nil, log), evtSource,
		func(ctx context.Context, chainID string, toHeight int64) error {
			return networkhistoryService.CreateAndPublishSegment(ctx, chainID, toHeight)
		})

	var lastCommittedBlockHeight int64
	sqlBroker, err := setupSQLBroker(ctx, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, blockHeight int64, snapshotTaken bool) {
			lastCommittedBlockHeight = blockHeight
		},
		evtSource, puh,
	)
	require.NoError(t, err)

	err = sqlBroker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(2500), lastCommittedBlockHeight)

	err = migrateUpToDatabaseVersion(testMigrationVersionNum)
	require.NoError(t, err)

	// After protocol upgrade restart the broker
	sqlBroker, err = setupSQLBroker(ctx, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				if lastCommittedBlockHeight == 3000 {
					ss, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
					require.NoError(t, err)
					waitForSnapshotToComplete(ss)

					snapshotFileHashAfterReloadAt2000AndEventReplayTo3000, err = Md5Hash(ss.UnpublishedSnapshotDataDirectory())
					require.NoError(t, err)
					cancelFn()
				}
			}
		},
		evtSource, networkhistory.NewProtocolUpgradeHandler(log, service.NewProtocolUpgrade(nil, log), evtSource,
			func(ctx context.Context, chainID string, toHeight int64) error {
				return nil
			}),
	)
	require.NoError(t, err)

	err = sqlBroker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	require.Equal(t, fromEventHashes[3], snapshotFileHashAfterReloadAt2000AndEventReplayTo3000)

	updateAllContinuousAggregateData(ctx)

	databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000 := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[3].currentTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.currentTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[3].historyTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.historyTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[3].caggSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.caggSummaries)
}

func fetchBlocks(ctx context.Context, log *logging.Logger, st *store.Store, rootSegmentID string, numBlocksToFetch int64) (int64, error) {
	var err error
	var fetched int64
	for i := 0; i < 5; i++ {
		ctxWithTimeout, cancelFn := context.WithTimeout(ctx, 10*time.Second)

		fetched, err = networkhistory.FetchHistoryBlocks(ctxWithTimeout, log.Infof, rootSegmentID,
			func(ctx context.Context, historySegmentID string) (networkhistory.FetchResult, error) {
				segment, err := st.FetchHistorySegment(ctx, historySegmentID)
				if err != nil {
					return networkhistory.FetchResult{}, err
				}
				return networkhistory.FromSegmentIndexEntry(segment), nil
			}, numBlocksToFetch)
		cancelFn()
		if err == nil {
			return fetched, nil
		}
	}

	return 0, fmt.Errorf("failed to fetch blocks:%w", err)
}

func TestRollingBackToHeightAcrossSchemaUpdateBoundary(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logging.NewTestLogger()

	require.NoError(t, networkHistoryStore.ResetIndex())
	emptyDatabaseAndSetSchemaVersion(highestMigrationNumber)

	snapshotCopyToPath := t.TempDir()
	snapshotService := setupSnapshotService(snapshotCopyToPath)

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	pus := service.NewProtocolUpgrade(nil, log)
	puh := networkhistory.NewProtocolUpgradeHandler(log, pus, evtSource, func(ctx context.Context, chainID string,
		toHeight int64,
	) error {
		return nil
	})

	// Run events to height 5000
	broker, err := setupSQLBroker(ctx, sqlConfig, snapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight == 5000 {
				cancelFn()
			}
		},
		evtSource, puh)
	require.NoError(t, err)

	err = broker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}
	updateAllContinuousAggregateData(ctx)

	fetched, err := fetchBlocks(ctx, log, networkHistoryStore, goldenSourceHistorySegment[5000].HistorySegmentID, 5000)
	require.NoError(t, err)
	require.Equal(t, int64(5000), fetched)

	snapshotCopyToPath = t.TempDir()

	inputSnapshotService := setupSnapshotService(snapshotCopyToPath)

	networkhistoryService := setupNetworkHistoryService(ctx, log, inputSnapshotService, networkHistoryStore, snapshotCopyToPath)

	// Rollback to a height pre protocol upgrade
	err = networkhistoryService.RollbackToHeight(ctx, log, 2000)
	require.NoError(t, err)

	dbSummary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[1].currentTableSummaries, dbSummary.currentTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[1].historyTableSummaries, dbSummary.historyTableSummaries)
	assertSummariesAreEqual(t, fromEventsDatabaseSummaries[1].caggSummaries, dbSummary.caggSummaries)

	historySegments, err := networkHistoryStore.ListAllIndexEntriesMostRecentFirst()
	require.NoError(t, err)
	assert.Equal(t, 2, len(historySegments))
	assert.Equal(t, int64(2000), historySegments[0].HeightTo)
	assert.Equal(t, int64(1000), historySegments[1].HeightTo)
}

func setupNetworkHistoryService(ctx context.Context, log *logging.Logger, inputSnapshotService *snapshot.Service, store *store.Store,
	snapshotCopyToPath string,
) *networkhistory.Service {
	cfg := networkhistory.NewDefaultConfig()
	cfg.Publish = false

	datanodeConfig := config2.NewDefaultConfig()

	networkHistoryService, err := networkhistory.New(ctx, log, chainID, cfg, networkHistoryConnPool,
		inputSnapshotService, store, datanodeConfig.API.Port, snapshotCopyToPath)
	if err != nil {
		panic(err)
	}

	return networkHistoryService
}

type sqlStoreBroker interface {
	Receive(ctx context.Context) error
}

func emptyDatabaseAndSetSchemaVersion(schemaVersion int64) {
	// For these we need a totally fresh database every time to ensure we model as closely as
	// possible what happens in practice
	var err error
	var poolConfig *pgxpool.Config

	poolConfig, err = sqlConfig.ConnectionConfig.GetPoolConfig()
	if err != nil {
		panic(fmt.Errorf("failed to get pool config: %w", err))
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)

	if _, err = db.Exec(`SELECT alter_job(job_id, scheduled => false) FROM timescaledb_information.jobs WHERE proc_name = 'policy_refresh_continuous_aggregate'`); err != nil {
		panic(fmt.Errorf("failed to stop continuous aggregates: %w", err))
	}
	db.Close()

	for i := 0; i < 5; i++ {
		err = sqlstore.WipeDatabaseAndMigrateSchemaToVersion(logging.NewTestLogger(), sqlConfig.ConnectionConfig, schemaVersion, sqlFs, false)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		panic(err)
	}
}

func panicIfHistorySegmentIdsNotEqual(actual, expected string, snapshots []segment.Unpublished) {
	if expected != actual {
		snapshotPaths := ""
		for _, sn := range snapshots {
			snapshotPaths += "," + sn.ZipFileName()
		}

		panic(fmt.Errorf("history segment ids are not equal, expected: %s  actual: %s\n"+
			"If the database schema has changed or event file been updated the history segment ids "+
			"will need updating.  Snapshot files: %s", expected, actual, snapshotPaths))
	}
}

func assertIntervalHistoryIsEmpty(t *testing.T, historyTableDelta []map[string]tableDataSummary, interval int) {
	t.Helper()
	totalRowCount := 0
	for _, summary := range historyTableDelta[interval] {
		totalRowCount += summary.rowCount
	}
	assert.Equal(t, 0, totalRowCount, "expected interval history to be empty but found %d rows", totalRowCount)
}

func setupSnapshotService(snapshotCopyToPath string) *snapshot.Service {
	snapshotServiceCfg := snapshot.NewDefaultConfig()
	snapshotService, err := snapshot.NewSnapshotService(logging.NewTestLogger(), snapshotServiceCfg,
		networkHistoryConnPool, networkHistoryStore, snapshotCopyToPath, migrateUpToDatabaseVersion,
		migrateDownToDatabaseVersion)
	if err != nil {
		panic(err)
	}

	return snapshotService
}

type ProtocolUpgradeHandler interface {
	OnProtocolUpgradeEvent(ctx context.Context, chainID string, lastCommittedBlockHeight int64) error
	GetProtocolUpgradeStarted() bool
}

func setupSQLBroker(ctx context.Context, testDbConfig sqlstore.Config, snapshotService *snapshot.Service,
	onBlockCommitted func(ctx context.Context, service *snapshot.Service, chainId string,
		lastCommittedBlockHeight int64, snapshotTaken bool), evtSource eventSource, protocolUpdateHandler ProtocolUpgradeHandler,
) (sqlStoreBroker, error) {
	transactionalConnectionSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), testDbConfig.ConnectionConfig)
	if err != nil {
		return nil, err
	}
	go func() {
		for range ctx.Done() {
			transactionalConnectionSource.Close()
		}
	}()

	candlesV2Config := candlesv2.NewDefaultConfig()
	subscribers := start.SQLSubscribers{}
	subscribers.CreateAllStores(ctx, logging.NewTestLogger(), transactionalConnectionSource, candlesV2Config.CandleStore)
	err = subscribers.SetupServices(ctx, logging.NewTestLogger(), candlesV2Config)
	if err != nil {
		return nil, err
	}

	subscribers.SetupSQLSubscribers()

	blockStore := sqlstore.NewBlocks(transactionalConnectionSource)
	if err != nil {
		return nil, fmt.Errorf("failed to create block store: %w", err)
	}

	config := broker.NewDefaultConfig()

	sqlBroker := broker.NewSQLStoreBroker(logging.NewTestLogger(), config, chainID, evtSource,
		transactionalConnectionSource, blockStore, func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			onBlockCommitted(ctx, snapshotService, chainId, lastCommittedBlockHeight, snapshotTaken)
		}, protocolUpdateHandler, subscribers.GetSQLSubscribers(),
	)
	return sqlBroker, nil
}

type eventSource interface {
	Listen() error
	Receive(ctx context.Context) (<-chan events.Event, <-chan error)
}

type TestEventSource struct {
	fileSource eventSource
	onEvent    func(events.Event, chan<- events.Event)
}

func newTestEventSource(onEvent func(events.Event, chan<- events.Event)) (*TestEventSource, error) {
	rawEvtSource, err := broker.NewBufferFilesEventSource(eventsDir, 0, 0, chainID)
	if err != nil {
		return nil, err
	}
	evtSource := broker.NewDeserializer(rawEvtSource)

	return &TestEventSource{
		fileSource: evtSource,
		onEvent:    onEvent,
	}, nil
}

func (e *TestEventSource) Listen() error {
	e.fileSource.Listen()
	return nil
}

func (e *TestEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	sourceEventCh, sourceErrCh := e.fileSource.Receive(ctx)

	sinkEventCh := make(chan events.Event)
	sinkErrCh := make(chan error)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-sourceErrCh:
				sinkErrCh <- err
			case event := <-sourceEventCh:
				e.onEvent(event, sinkEventCh)
				sinkEventCh <- event
			}
		}
	}()

	return sinkEventCh, sinkErrCh
}

func (e *TestEventSource) Send(evt events.Event) error {
	return nil
}

type tableDataSummary struct {
	tableName string
	rowCount  int
	dataHash  string
}

func assertSummariesAreEqual(t *testing.T, expected map[string]tableDataSummary, actual map[string]tableDataSummary) {
	t.Helper()
	if len(expected) != len(actual) {
		require.Equalf(t, len(expected), len(actual), "expected table count: %d actual: %d", len(expected), len(actual))
	}

	for k, v := range expected {
		if v.rowCount != actual[k].rowCount {
			assert.Equalf(t, v.rowCount, actual[k].rowCount, "expected table row count for %s: %d actual row count: %d", k, v.rowCount, actual[k].rowCount)
		}

		if v.dataHash != actual[k].dataHash {
			assert.Equalf(t, v.dataHash, actual[k].dataHash, "expected data hash for %s: %s actual data hash: %s", k, v.dataHash, actual[k].dataHash)
		}
	}
}

type databaseSummary struct {
	currentTableSummaries map[string]tableDataSummary
	historyTableSummaries map[string]tableDataSummary
	caggSummaries         map[string]tableDataSummary
	dbMetaData            snapshot.DatabaseMetadata
}

func getDatabaseDataSummary(ctx context.Context, connConfig sqlstore.ConnectionConfig) databaseSummary {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		panic(err)
	}

	currentStateDataSummaries := map[string]tableDataSummary{}
	historyStateDataSummaries := map[string]tableDataSummary{}
	dbMetaData, err := snapshot.NewDatabaseMetaData(ctx, conn)
	if err != nil {
		panic(err)
	}

	for table, meta := range dbMetaData.TableNameToMetaData {
		summary := getTableOrViewSummary(ctx, conn, table, meta.SortOrder)

		if meta.Hypertable {
			historyStateDataSummaries[table] = summary
		} else {
			currentStateDataSummaries[table] = summary
		}
	}

	// No sensible way to get the order by from database metadata so it is hardcoded here, will need to be added
	// to if new CAGGS are added
	viewNameToGroupBy := map[string]string{
		"conflated_balances":       "account_id, bucket",
		"conflated_margin_levels":  "account_id, bucket",
		"conflated_positions":      "market_id, party_id, bucket",
		"trades_candle_15_minutes": "market_id, period_start",
		"trades_candle_1_day":      "market_id, period_start",
		"trades_candle_1_hour":     "market_id, period_start",
		"trades_candle_1_minute":   "market_id, period_start",
		"trades_candle_5_minutes":  "market_id, period_start",
		"trades_candle_6_hours":    "market_id, period_start",
		"trades_candle_30_minutes": "market_id, period_start",
		"trades_candle_4_hours":    "market_id, period_start",
		"trades_candle_8_hours":    "market_id, period_start",
		"trades_candle_12_hours":   "market_id, period_start",
		"trades_candle_7_days":     "market_id, period_start",
	}

	caggSummaries := map[string]tableDataSummary{}
	for _, caggMeta := range dbMetaData.ContinuousAggregatesMetaData {
		summary := getTableOrViewSummary(ctx, conn, caggMeta.Name, viewNameToGroupBy[caggMeta.Name])
		caggSummaries[caggMeta.Name] = summary
	}

	return databaseSummary{
		historyTableSummaries: historyStateDataSummaries, currentTableSummaries: currentStateDataSummaries,
		caggSummaries: caggSummaries,
		dbMetaData:    dbMetaData,
	}
}

func getTableOrViewSummary(ctx context.Context, conn *pgxpool.Pool, table string, sortOrder string) tableDataSummary {
	summary := tableDataSummary{tableName: table}
	err := conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s", table)).Scan(&summary.rowCount)
	if err != nil {
		panic(err)
	}

	if summary.rowCount > 0 {
		err = conn.QueryRow(ctx, fmt.Sprintf("SELECT md5(CAST((array_agg(f.* order by %s))AS text)) FROM %s f; ",
			sortOrder, table)).Scan(&summary.dataHash)
		if err != nil {
			panic(err)
		}
	}
	return summary
}

func getSnapshotIntervalToHistoryTableDeltaSummary(ctx context.Context,
	connConfig sqlstore.ConnectionConfig, expectedHistorySegmentsFromHeights []int64,
	expectedHistorySegmentsToHeights []int64,
) []map[string]tableDataSummary {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		panic(err)
	}

	dbMetaData, err := snapshot.NewDatabaseMetaData(ctx, conn)
	if err != nil {
		panic(err)
	}

	var snapshotNumToHistoryTableSummary []map[string]tableDataSummary

	for i := 0; i < len(expectedHistorySegmentsFromHeights); i++ {
		fromHeight := expectedHistorySegmentsFromHeights[i]
		toHeight := expectedHistorySegmentsToHeights[i]

		whereClause := fmt.Sprintf("Where vega_time >= (SELECT vega_time from blocks where height = %d) and  vega_time <= (SELECT vega_time from blocks where height = %d)",
			fromHeight, toHeight)

		intervalHistoryTableSummary := map[string]tableDataSummary{}
		for table, meta := range dbMetaData.TableNameToMetaData {
			if meta.Hypertable {
				summary := tableDataSummary{tableName: table}
				err := conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s %s", table, whereClause)).Scan(&summary.rowCount)
				if err != nil {
					panic(err)
				}

				if summary.rowCount > 0 {
					err = conn.QueryRow(ctx, fmt.Sprintf("SELECT md5(CAST((array_agg(f.* order by %s))AS text)) FROM %s f %s; ",
						meta.SortOrder, table, whereClause)).Scan(&summary.dataHash)
					if err != nil {
						panic(err)
					}
				}

				intervalHistoryTableSummary[table] = summary
			}
		}
		snapshotNumToHistoryTableSummary = append(snapshotNumToHistoryTableSummary, intervalHistoryTableSummary)
	}
	return snapshotNumToHistoryTableSummary
}

func waitForSnapshotToComplete(sf segment.Unpublished) {
	for {
		time.Sleep(10 * time.Millisecond)
		// wait for snapshot current  file
		_, err := os.Stat(sf.UnpublishedSnapshotDataDirectory())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				panic(err)
			}
		}

		// wait for snapshot data dump in progress file to be removed

		_, err = os.Stat(sf.InProgressFilePath())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			} else {
				panic(err)
			}
		} else {
			continue
		}
	}
}

func decompressEventFile() {
	sourceFile, err := os.Open(compressedEventsFile)
	if err != nil {
		panic(err)
	}

	zr, err := gzip.NewReader(sourceFile)
	if err != nil {
		panic(err)
	}

	fileToWrite, err := os.Create(eventsFile)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(fileToWrite, zr); err != nil {
		panic(err)
	}
}

func setupTestSQLMigrations() (int64, fs.FS) {
	sourceMigrationsDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	sourceMigrationsDir, _ = filepath.Split(sourceMigrationsDir)
	sourceMigrationsDir = filepath.Join(sourceMigrationsDir, "sqlstore", "migrations")

	testMigrationsDir, err = os.MkdirTemp("", "migrations")
	if err != nil {
		panic(err)
	}

	if err := os.Mkdir(filepath.Join(testMigrationsDir, sqlstore.SQLMigrationsDir), fs.ModePerm); err != nil {
		panic(fmt.Errorf("failed to create migrations dir: %w", err))
	}

	var highestMigrationNumber int64
	err = filepath.Walk(sourceMigrationsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || (info != nil && info.IsDir()) {
			return nil //nolint:nilerr
		}
		if strings.HasSuffix(info.Name(), ".sql") {
			split := strings.Split(info.Name(), "_")
			if len(split) < 2 {
				return errors.New("expected sql filename of form <version>_<name>.sql")
			}

			migrationNum, err := strconv.Atoi(split[0])
			if err != nil {
				return fmt.Errorf("expected first part of file name to be integer, is %s", split[0])
			}

			if int64(migrationNum) > highestMigrationNumber {
				highestMigrationNumber = int64(migrationNum)
			}

			data, err := os.ReadFile(filepath.Join(sourceMigrationsDir, info.Name()))
			if err != nil {
				return fmt.Errorf("failed to read file:%w", err)
			}

			err = os.WriteFile(filepath.Join(testMigrationsDir, sqlstore.SQLMigrationsDir, info.Name()), data, fs.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to write file:%w", err)
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	// Create a file with a new migration
	sql, err := os.ReadFile(testMigrationSQL)
	if err != nil {
		panic(err)
	}

	testMigrationVersionNum := highestMigrationNumber + 1
	err = os.WriteFile(filepath.Join(testMigrationsDir, sqlstore.SQLMigrationsDir,
		fmt.Sprintf("%04d_testmigration.sql", testMigrationVersionNum)), sql, fs.ModePerm)
	if err != nil {
		panic(err)
	}

	return testMigrationVersionNum, os.DirFS(testMigrationsDir)
}

func migrateUpToDatabaseVersion(version int64) error {
	poolConfig, err := sqlConfig.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	goose.SetBaseFS(nil)
	err = goose.UpTo(db, filepath.Join(testMigrationsDir, sqlstore.SQLMigrationsDir), version)
	if err != nil {
		return fmt.Errorf("failed to migrate up to version %d:%w", version, err)
	}

	return nil
}

func migrateDownToDatabaseVersion(version int64) error {
	poolConfig, err := sqlConfig.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	goose.SetBaseFS(nil)
	err = goose.DownTo(db, filepath.Join(testMigrationsDir, sqlstore.SQLMigrationsDir), version)
	if err != nil {
		return fmt.Errorf("failed to migrate down to version %d:%w", version, err)
	}

	return nil
}

func newTestEventSourceWithProtocolUpdateMessage() *TestEventSource {
	var currentBlock *entities.Block
	var m sync.RWMutex
	evtSource, err := newTestEventSource(func(e events.Event, evtsCh chan<- events.Event) {
		if e == nil {
			return
		}
		var err error
		switch e.Type() {
		case events.EndBlockEvent:

		case events.BeginBlockEvent:
			m.Lock()
			if currentBlock != nil && currentBlock.Height == 2500 {
				evtsCh <- events.NewProtocolUpgradeStarted(context.Background(), eventsv1.ProtocolUpgradeStarted{
					LastBlockHeight: uint64(currentBlock.Height),
				})
			}
			beginBlock := e.(entities.BeginBlockEvent)
			currentBlock, err = entities.BlockFromBeginBlock(beginBlock)
			m.Unlock()
			if err != nil {
				panic(err)
			}
		}
	})
	if err != nil {
		panic(err)
	}
	return evtSource
}

func Md5Hash(dir string) (string, error) {
	hash := md5.New()
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(hash, file)
		if err != nil {
			return err
		}

		return nil
	})

	return hex.EncodeToString(hash.Sum(nil)), nil
}
