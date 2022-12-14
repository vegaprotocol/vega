package dehistory_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	eventsv1 "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"

	"code.vegaprotocol.io/vega/cmd/data-node/commands/start"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	config2 "code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/dehistory"
	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/dehistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	snapshotInterval     = int64(1000)
	chainID              = "testnet"
	compressedEventsFile = "testdata/smoketest_to_block_5000.evts.gz"
	numSnapshots         = 6
	testMigrationSQL     = "testdata/testmigration.sql"
)

var (
	sqlConfig sqlstore.Config

	fromEventsSnapshotHashes    []string
	fromEventsDatabaseSummaries []databaseSummary

	fromEventsIntervalToHistoryHashes     []string
	fromEventsIntervalToHistoryTableDelta []map[string]tableDataSummary

	snapshotsBackupDir string
	eventsFile         string

	deHistoryService *dehistory.Service

	goldenSourceHistorySegment map[int64]store.SegmentIndexEntry

	expectedHistorySegmentsFromHeights = []int64{1, 1001, 2001, 2501, 3001, 4001}
	expectedHistorySegmentsToHeights   = []int64{1000, 2000, 2500, 3000, 4000, 5000}

	deHistoryStore *store.Store

	postgresLog *bytes.Buffer

	testMigrationsDir       string
	testMigrationVersionNum int
	sqlFs                   fs.FS
)

func TestMain(t *testing.M) {
	outerCtx, cancelOuterCtx := context.WithCancel(context.Background())
	defer cancelOuterCtx()

	testMigrationVersionNum, sqlFs = setupTestSQLMigrations()

	var err error
	snapshotsBackupDir, err = os.MkdirTemp("", "snapshotbackup")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(snapshotsBackupDir)

	eventsDir, err := os.MkdirTemp("", "eventsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(eventsDir)

	log := logging.NewTestLogger()

	eventsFile = filepath.Join(eventsDir, "smoketest_to_block_5000_or_above.evts")
	decompressEventFile()

	tempDir, err := os.MkdirTemp("", "dehistory")
	if err != nil {
		panic(err)
	}
	postgresRuntimePath := filepath.Join(tempDir, "sqlstore")
	defer os.RemoveAll(tempDir)

	deHistoryHome, err := os.MkdirTemp("", "dehistoryhome")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(deHistoryHome)

	exitCode := databasetest.TestMain(t, func(config sqlstore.Config, source *sqlstore.ConnectionSource,
		pgLog *bytes.Buffer,
	) {
		sqlConfig = config

		postgresLog = pgLog

		emptyDatabase()

		// Do initial run to get the expected state of the datanode from just event playback
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		snapshotCopyToPath := filepath.Join(deHistoryHome, "snapshotsCopyTo")
		snapshotCopyFromPath := filepath.Join(deHistoryHome, "snapshotsCopyFrom")

		snapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

		var snapshots []snapshot.MetaData

		ctxWithCancel, cancelFn := context.WithCancel(ctx)

		evtSource := newTestEventSourceWithProtocolUpdateMessage()

		pus := service.NewProtocolUpgrade(nil, log)
		puh := dehistory.NewProtocolUpgradeHandler(log, pus, func(ctx context.Context, chainID string,
			toHeight int64,
		) error {
			meta, err := snapshotService.CreateSnapshot(ctx, chainID, toHeight)
			if err != nil {
				panic(fmt.Errorf("failed to create snapshot: %w", err))
			}

			waitForSnapshotToCompleteUseMeta(meta)

			snapshots = append(snapshots, meta)

			md5Hash, err := snapshot.GetSnapshotMd5Hash(meta.CurrentStateSnapshotPath, meta.HistorySnapshotPath)
			if err != nil {
				panic(fmt.Errorf("failed to get snapshot hash:%w", err))
			}

			fromEventsSnapshotHashes = append(fromEventsSnapshotHashes, md5Hash)

			historyMd5Hash, err := snapshot.GetHistoryMd5Hash(meta)
			if err != nil {
				panic(fmt.Errorf("failed to get history hash:%w", err))
			}

			fromEventsIntervalToHistoryHashes = append(fromEventsIntervalToHistoryHashes, historyMd5Hash)

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

					waitForSnapshotToCompleteUseMeta(lastSnapshot)
					snapshots = append(snapshots, lastSnapshot)
					md5Hash, err := snapshot.GetSnapshotMd5Hash(lastSnapshot.CurrentStateSnapshotPath, lastSnapshot.HistorySnapshotPath)
					if err != nil {
						panic(fmt.Errorf("failed to get snapshot hash:%w", err))
					}

					fromEventsSnapshotHashes = append(fromEventsSnapshotHashes, md5Hash)

					historyMd5Hash, err := snapshot.GetHistoryMd5Hash(lastSnapshot)
					if err != nil {
						panic(fmt.Errorf("failed to get history hash:%w", err))
					}

					fromEventsIntervalToHistoryHashes = append(fromEventsIntervalToHistoryHashes, historyMd5Hash)

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
		err = migrateDatabase(int64(testMigrationVersionNum))
		if err != nil {
			panic(err)
		}

		pus = service.NewProtocolUpgrade(nil, log)
		nonInterceptPuh := dehistory.NewProtocolUpgradeHandler(log, pus, func(ctx context.Context,
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

					waitForSnapshotToCompleteUseMeta(lastSnapshot)
					snapshots = append(snapshots, lastSnapshot)
					md5Hash, err := snapshot.GetSnapshotMd5Hash(lastSnapshot.CurrentStateSnapshotPath, lastSnapshot.HistorySnapshotPath)
					if err != nil {
						panic(fmt.Errorf("failed to get snapshot hash:%w", err))
					}

					fromEventsSnapshotHashes = append(fromEventsSnapshotHashes, md5Hash)

					historyMd5Hash, err := snapshot.GetHistoryMd5Hash(lastSnapshot)
					if err != nil {
						panic(fmt.Errorf("failed to get history hash:%w", err))
					}

					fromEventsIntervalToHistoryHashes = append(fromEventsIntervalToHistoryHashes, historyMd5Hash)

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

		if len(fromEventsSnapshotHashes) != numSnapshots {
			panic(fmt.Errorf("expected 5 snapshots, got %d", len(fromEventsSnapshotHashes)))
		}

		if len(fromEventsDatabaseSummaries) != numSnapshots {
			panic(fmt.Errorf("expected %d database summaries, got %d", numSnapshots, len(fromEventsSnapshotHashes)))
		}

		fromEventsIntervalToHistoryTableDelta = getSnapshotIntervalToHistoryTableDeltaSummary(ctx, sqlConfig.ConnectionConfig,
			expectedHistorySegmentsFromHeights, expectedHistorySegmentsToHeights)

		if len(fromEventsIntervalToHistoryTableDelta) != numSnapshots {
			panic(fmt.Errorf("expected %d history table deltas, got %d", numSnapshots, len(fromEventsSnapshotHashes)))
		}

		// Decentralised store setup
		storeCfg := store.NewDefaultConfig()
		storeCfg.SwarmKeyOverride = uuid.NewV4().String()

		storeCfg.SwarmPort = databasetest.GetNextFreePort()
		storeCfg.UseIpfsDefaultPeers = false
		storeCfg.StartWebUI = false

		deHistoryStore, err = store.New(outerCtx, log, chainID, storeCfg, deHistoryHome, false)
		if err != nil {
			panic(err)
		}

		datanodeConfig := config2.NewDefaultConfig()
		cfg := dehistory.NewDefaultConfig()
		cfg.WipeOnStartup = false
		deHistoryService, err = dehistory.NewWithStore(outerCtx, log, chainID, cfg, sqlConfig.ConnectionConfig, snapshotService,
			deHistoryStore, datanodeConfig.API.Port, snapshotCopyFromPath, snapshotCopyToPath)

		if err != nil {
			panic(err)
		}

		start := time.Now()
		timeout := 1 * time.Minute

		for {
			if time.Now().After(start.Add(timeout)) {
				panic(fmt.Sprintf("history not found in decentralised store after %s", timeout))
			}

			time.Sleep(10 * time.Millisecond)

			storedSegments, err := deHistoryStore.ListAllHistorySegmentsOldestFirst()
			if err != nil {
				panic(err)
			}

			goldenSourceHistorySegment = map[int64]store.SegmentIndexEntry{}
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

		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[1000].HistorySegmentID, "QmdNuP2pa7hBWjqLrUCs1GHV9EbXcLCd1oi5mR4xLghApc", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[2000].HistorySegmentID, "QmXat11d8fRtQcbWb6jKaRQvjts9KScaUhU2F8vBMhoGWd", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[2500].HistorySegmentID, "QmUuemAJ58XnLvf8hZCSq7kb7xQTJoXmymvuph4mv1GwwA", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[3000].HistorySegmentID, "QmQDVXn7WKm5duxHiKmuWbXC8pif86dHM3xhiiDy3MtP2C", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[4000].HistorySegmentID, "QmS5NuL3exbPEwFPZFunpPedwn1gJQnNSCKg7dSu4A12B8", snapshots)
		panicIfHistorySegmentIdsNotEqual(goldenSourceHistorySegment[5000].HistorySegmentID, "QmPEfToxPV7jwJrFzPzsvN9UeYnzXq7j3aDpTFvYDeXARb", snapshots)
	}, postgresRuntimePath, sqlFs)

	if exitCode != 0 {
		log.Errorf("One or more tests failed, dumping postgres log:\n%s", postgresLog.String())
	}
}

func TestRestoringNodeThatAlreadyContainsData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logging.NewTestLogger()

	deHistoryStore.ResetIndex()
	emptyDatabase()

	snapshotCopyFromPath := t.TempDir()
	snapshotCopyToPath := t.TempDir()
	snapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	pus := service.NewProtocolUpgrade(nil, log)
	puh := dehistory.NewProtocolUpgradeHandler(log, pus, func(ctx context.Context, chainID string,
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

	// Now restore the node to height 4000

	fetched, err := fetchBlocks(ctx, log, deHistoryStore, goldenSourceHistorySegment[4000].HistorySegmentID, 3000)
	require.NoError(t, err)
	require.Equal(t, int64(3000), fetched)

	snapshotCopyFromPath = t.TempDir()
	snapshotCopyToPath = t.TempDir()

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	dehistoryService := setupDeHistoryService(ctx, log, inputSnapshotService, deHistoryStore, snapshotCopyFromPath, snapshotCopyToPath)

	loaded, err := dehistoryService.LoadAllAvailableHistoryIntoDatanode(ctx, sqlFs)
	require.NoError(t, err)
	assert.Equal(t, int64(1801), loaded.LoadedFromHeight)
	assert.Equal(t, int64(4000), loaded.LoadedToHeight)

	dbSummary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[4].currentTableSummaries, dbSummary.currentTableSummaries)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[4].historyTableSummaries, dbSummary.historyTableSummaries)

	// Run events to height 5000
	ctxWithCancel, cancelFn = context.WithCancel(ctx)
	evtSource = newTestEventSourceWithProtocolUpdateMessage()

	pus = service.NewProtocolUpgrade(nil, log)
	puh = dehistory.NewProtocolUpgradeHandler(log, pus, func(ctx context.Context, chainID string,
		toHeight int64,
	) error {
		return nil
	})

	var md5Hash string
	var historyMd5Hash string
	broker, err = setupSQLBroker(ctx, sqlConfig, snapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				meta, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
				require.NoError(t, err)

				waitForSnapshotToCompleteUseMeta(meta)

				md5Hash, err = snapshot.GetSnapshotMd5Hash(meta.CurrentStateSnapshotPath, meta.HistorySnapshotPath)
				require.NoError(t, err)

				fromEventsSnapshotHashes = append(fromEventsSnapshotHashes, md5Hash)

				historyMd5Hash, err = snapshot.GetHistoryMd5Hash(meta)
				require.NoError(t, err)
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

	require.Equal(t, fromEventsSnapshotHashes[5], md5Hash)
	require.Equal(t, fromEventsIntervalToHistoryHashes[5], historyMd5Hash)

	dbSummary = getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[5].currentTableSummaries, dbSummary.currentTableSummaries)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[5].historyTableSummaries, dbSummary.historyTableSummaries)
}

func TestRestoringNodeWithHistoryFromBeforeTheNodesOldestBlockFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deHistoryStore.ResetIndex()

	log := logging.NewTestLogger()

	snapshotCopyFromPath := t.TempDir()
	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	emptyDatabase()

	historySegment := goldenSourceHistorySegment[4000]

	blocksFetched, err := fetchBlocks(ctx, log, deHistoryStore, historySegment.HistorySegmentID, 1)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), blocksFetched)
	dehistoryService := setupDeHistoryService(ctx, log, inputSnapshotService, deHistoryStore, snapshotCopyFromPath, snapshotCopyToPath)

	loaded, err := dehistoryService.LoadAllAvailableHistoryIntoDatanode(ctx, sqlFs)
	require.NoError(t, err)

	assert.Equal(t, int64(3001), loaded.LoadedFromHeight)
	assert.Equal(t, int64(4000), loaded.LoadedToHeight)

	// Now try to load in history from 1000 to 2000
	deHistoryStore.ResetIndex()
	snapshotCopyFromPath = t.TempDir()
	snapshotCopyToPath = t.TempDir()
	inputSnapshotService = setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	historySegment = goldenSourceHistorySegment[1000]

	blocksFetched, err = fetchBlocks(ctx, log, deHistoryStore, historySegment.HistorySegmentID, 1)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), blocksFetched)
	dehistoryService = setupDeHistoryService(ctx, log, inputSnapshotService, deHistoryStore, snapshotCopyFromPath, snapshotCopyToPath)

	_, err = dehistoryService.LoadAllAvailableHistoryIntoDatanode(ctx, sqlFs)
	require.NotNil(t, err)
}

func TestRestoringNodeWithExistingDataFailsWhenLoadingWouldResultInNonContiguousHistory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logging.NewTestLogger()

	deHistoryStore.ResetIndex()
	emptyDatabase()

	snapshotCopyFromPath := t.TempDir()
	snapshotCopyToPath := t.TempDir()
	snapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	pus := service.NewProtocolUpgrade(nil, log)
	puh := dehistory.NewProtocolUpgradeHandler(log, pus, func(ctx context.Context, chainID string,
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

	fetched, err := fetchBlocks(ctx, log, deHistoryStore, goldenSourceHistorySegment[4000].HistorySegmentID, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(2000), fetched)

	snapshotCopyFromPath = t.TempDir()
	snapshotCopyToPath = t.TempDir()

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	dehistoryService := setupDeHistoryService(ctx, log, inputSnapshotService, deHistoryStore, snapshotCopyFromPath, snapshotCopyToPath)

	_, err = dehistoryService.LoadAllAvailableHistoryIntoDatanode(ctx, sqlFs)
	require.NotNil(t, err)
}

func TestRestoringFromDifferentHeightsWithFullHistory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deHistoryStore.ResetIndex()

	log := logging.NewTestLogger()

	snapshotCopyFromPath := t.TempDir()
	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	for i := int64(0); i < numSnapshots; i++ {
		emptyDatabase()
		fromHeight := expectedHistorySegmentsFromHeights[i]
		toHeight := expectedHistorySegmentsToHeights[i]

		historySegment := goldenSourceHistorySegment[toHeight]

		expectedBlocks := toHeight - fromHeight + 1
		blocksFetched, err := fetchBlocks(ctx, log, deHistoryStore, historySegment.HistorySegmentID, expectedBlocks)
		require.NoError(t, err)

		assert.Equal(t, expectedBlocks, blocksFetched)
		dehistoryService := setupDeHistoryService(ctx, log, inputSnapshotService, deHistoryStore, snapshotCopyFromPath, snapshotCopyToPath)

		loaded, err := dehistoryService.LoadAllAvailableHistoryIntoDatanode(ctx, sqlFs)
		require.NoError(t, err)

		assert.Equal(t, int64(1), loaded.LoadedFromHeight)
		assert.Equal(t, toHeight, loaded.LoadedToHeight)

		dbSummary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
		assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[i].currentTableSummaries, dbSummary.currentTableSummaries)
		assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[i].historyTableSummaries, dbSummary.historyTableSummaries)
	}
}

func TestRestoreFromPartialHistoryAndProcessEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deHistoryStore.ResetIndex()

	var err error
	log := logging.NewTestLogger()

	emptyDatabase()

	fetched, err := fetchBlocks(ctx, log, deHistoryStore, goldenSourceHistorySegment[3000].HistorySegmentID, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(1000), fetched)

	snapshotCopyFromPath := t.TempDir()
	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	dehistoryService := setupDeHistoryService(ctx, log, inputSnapshotService, deHistoryStore, snapshotCopyFromPath, snapshotCopyToPath)

	loaded, err := dehistoryService.LoadAllAvailableHistoryIntoDatanode(ctx, sqlFs)
	require.NoError(t, err)
	assert.Equal(t, int64(2001), loaded.LoadedFromHeight)
	assert.Equal(t, int64(3000), loaded.LoadedToHeight)

	connSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), sqlConfig.ConnectionConfig)
	require.NoError(t, err)
	defer connSource.Close()

	evtSource, err := newTestEventSource(func(events.Event, chan<- events.Event) {})
	require.NoError(t, err)

	pus := service.NewProtocolUpgrade(nil, log)
	puh := dehistory.NewProtocolUpgradeHandler(log, pus, func(ctx context.Context,
		chainID string, toHeight int64,
	) error {
		return nil
	})

	// Play events from 3001 to 4000
	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	var snapshotMeta snapshot.MetaData
	var newSnapshotFileHashAt4000 string
	outDeHistoryHome := t.TempDir()
	outputSnapshotService := setupSnapshotService(sqlConfig, outDeHistoryHome, t.TempDir())
	sqlBroker, err := setupSQLBroker(ctx, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				snapshotMeta, err = service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
				require.NoError(t, err)
				waitForSnapshotToCompleteUseMeta(snapshotMeta)

				if lastCommittedBlockHeight == 4000 {
					newSnapshotFileHashAt4000, err = snapshot.GetSnapshotMd5Hash(snapshotMeta.CurrentStateSnapshotPath,
						snapshotMeta.HistorySnapshotPath)
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

	assert.Equal(t, fromEventsSnapshotHashes[4], newSnapshotFileHashAt4000)

	historyTableDelta := getSnapshotIntervalToHistoryTableDeltaSummary(ctx, sqlConfig.ConnectionConfig,
		expectedHistorySegmentsFromHeights, expectedHistorySegmentsToHeights)

	for i := 2; i < 5; i++ {
		assertTableSummariesAreEqual(t, fromEventsIntervalToHistoryTableDelta[i], historyTableDelta[i])
	}

	assertIntervalHistoryIsEmpty(t, historyTableDelta, 0)
	assertIntervalHistoryIsEmpty(t, historyTableDelta, 1)
}

func TestRestoreFromFullHistorySnapshotAndProcessEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deHistoryStore.ResetIndex()

	var err error
	log := logging.NewTestLogger()

	emptyDatabase()

	fetched, err := fetchBlocks(ctx, log, deHistoryStore, goldenSourceHistorySegment[2000].HistorySegmentID, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(2000), fetched)

	snapshotCopyFromPath := t.TempDir()
	snapshotCopyToPath := t.TempDir()

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotCopyFromPath, snapshotCopyToPath)

	dehistoryService := setupDeHistoryService(ctx, log, inputSnapshotService, deHistoryStore, snapshotCopyFromPath, snapshotCopyToPath)

	loaded, err := dehistoryService.LoadAllAvailableHistoryIntoDatanode(ctx, sqlFs)
	require.NoError(t, err)
	assert.Equal(t, int64(1), loaded.LoadedFromHeight)
	assert.Equal(t, int64(2000), loaded.LoadedToHeight)

	connSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), sqlConfig.ConnectionConfig)
	require.NoError(t, err)
	defer connSource.Close()

	ctxWithCancel, cancelFn := context.WithCancel(ctx)

	var snapshotFileHashAfterReloadAt2000AndEventReplayTo3000 string
	outSnapshotCopyToDir := t.TempDir()
	outputSnapshotService := setupSnapshotService(sqlConfig, outSnapshotCopyToDir, t.TempDir())

	evtSource := newTestEventSourceWithProtocolUpdateMessage()

	puh := dehistory.NewProtocolUpgradeHandler(log, service.NewProtocolUpgrade(nil, log),
		func(ctx context.Context, chainID string, toHeight int64) error {
			return dehistoryService.CreateAndPublishSegment(ctx, chainID, toHeight)
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

	err = migrateDatabase(int64(testMigrationVersionNum))
	require.NoError(t, err)

	// After protocol upgrade restart the broker
	sqlBroker, err = setupSQLBroker(ctx, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				if lastCommittedBlockHeight == 3000 {
					ss, err := service.CreateSnapshotAsynchronously(ctx, chainId, lastCommittedBlockHeight)
					require.NoError(t, err)
					waitForSnapshotToCompleteUseMeta(ss)

					snapshotFileHashAfterReloadAt2000AndEventReplayTo3000, err = snapshot.GetSnapshotMd5Hash(ss.CurrentStateSnapshotPath, ss.HistorySnapshotPath)
					require.NoError(t, err)
					cancelFn()
				}
			}
		},
		evtSource, dehistory.NewProtocolUpgradeHandler(log, service.NewProtocolUpgrade(nil, log),
			func(ctx context.Context, chainID string, toHeight int64) error {
				return nil
			}),
	)
	require.NoError(t, err)

	err = sqlBroker.Receive(ctxWithCancel)
	if err != nil && !errors.Is(err, context.Canceled) {
		require.NoError(t, err)
	}

	require.Equal(t, fromEventsSnapshotHashes[3], snapshotFileHashAfterReloadAt2000AndEventReplayTo3000)

	databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000 := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[3].currentTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.currentTableSummaries)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[3].historyTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.historyTableSummaries)
}

func fetchBlocks(ctx context.Context, log *logging.Logger, st *store.Store, rootSegmentID string, numBlocksToFetch int64) (int64, error) {
	var err error
	var fetched int64
	for i := 0; i < 5; i++ {
		ctxWithTimeout, cancelFn := context.WithTimeout(ctx, 10*time.Second)

		fetched, err = dehistory.FetchHistoryBlocks(ctxWithTimeout, log.Infof, rootSegmentID,
			func(ctx context.Context, historySegmentID string) (dehistory.FetchResult, error) {
				segment, err := st.FetchHistorySegment(ctx, historySegmentID)
				if err != nil {
					return dehistory.FetchResult{}, err
				}
				return dehistory.FromSegmentIndexEntry(segment), nil
			}, numBlocksToFetch)
		cancelFn()
		if err == nil {
			return fetched, nil
		}
	}

	return 0, fmt.Errorf("failed to fetch blocks:%w", err)
}

func setupDeHistoryService(ctx context.Context, log *logging.Logger, inputSnapshotService *snapshot.Service, store *store.Store,
	snapshotCopyFromPath, snapshotCopyToPath string,
) *dehistory.Service {
	cfg := dehistory.NewDefaultConfig()
	cfg.Publish = false

	datanodeConfig := config2.NewDefaultConfig()
	deHistoryService, err := dehistory.NewWithStore(ctx, log, chainID, cfg, sqlConfig.ConnectionConfig,
		inputSnapshotService, store, datanodeConfig.API.Port, snapshotCopyFromPath, snapshotCopyToPath)
	if err != nil {
		panic(err)
	}

	return deHistoryService
}

type sqlStoreBroker interface {
	Receive(ctx context.Context) error
}

func emptyDatabase() {
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
		err = sqlstore.WipeDatabase(logging.NewTestLogger(), sqlConfig.ConnectionConfig, sqlFs)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		panic(err)
	}
}

func panicIfHistorySegmentIdsNotEqual(actual string, expected string, snapshots []snapshot.MetaData) {
	if expected != actual {
		snapshotPaths := ""
		for _, sn := range snapshots {
			snapshotPaths += "," + sn.CurrentStateSnapshotPath + "," + sn.HistorySnapshotPath
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

func setupSnapshotService(testDbConfig sqlstore.Config, snapshotCopyFromPath string, snapshotCopyToPath string) *snapshot.Service {
	brokerCfg := broker.NewDefaultConfig()
	brokerCfg.UseEventFile = true
	brokerCfg.FileEventSourceConfig.TimeBetweenBlocks = encoding.Duration{Duration: 0}

	return setupSnapshotServiceWithNetworkParamFunc(testDbConfig, snapshotCopyFromPath, snapshotCopyToPath)
}

func setupSnapshotServiceWithNetworkParamFunc(testDbConfig sqlstore.Config, snapshotCopyFromPath string, snapshotCopyToPath string) *snapshot.Service {
	snapshotServiceCfg := snapshot.NewDefaultConfig()

	snapshotService, err := snapshot.NewSnapshotService(logging.NewTestLogger(), snapshotServiceCfg,
		testDbConfig.ConnectionConfig, snapshotCopyFromPath, snapshotCopyToPath, migrateDatabase)
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

	subscribers.SetupSQLSubscribers(ctx, logging.NewTestLogger())

	blockStore := sqlstore.NewBlocks(transactionalConnectionSource)

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
	evtSource, err := broker.NewFileEventSource(eventsFile, 0, 0, chainID)
	if err != nil {
		return nil, err
	}

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

type tableDataSummary struct {
	tableName string
	rowCount  int
	dataHash  string
}

func assertTableSummariesAreEqual(t *testing.T, expected map[string]tableDataSummary, actual map[string]tableDataSummary) {
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
	dbMetaData            snapshot.DatabaseMetadata
}

func getDatabaseDataSummary(ctx context.Context, connConfig sqlstore.ConnectionConfig) databaseSummary {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		panic(err)
	}

	currentStateDataSummaries := map[string]tableDataSummary{}
	historyStateDataSummaries := map[string]tableDataSummary{}
	dbMetaData, err := snapshot.NewDatabaseMetaData(ctx, connConfig)
	if err != nil {
		panic(err)
	}

	for table, meta := range dbMetaData.TableNameToMetaData {
		summary := tableDataSummary{tableName: table}
		err = conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s", table)).Scan(&summary.rowCount)
		if err != nil {
			panic(err)
		}

		if summary.rowCount > 0 {
			err = conn.QueryRow(ctx, fmt.Sprintf("SELECT md5(CAST((array_agg(f.* order by %s))AS text)) FROM %s f; ",
				meta.SortOrder, table)).Scan(&summary.dataHash)
			if err != nil {
				panic(err)
			}
		}

		if meta.Hypertable {
			historyStateDataSummaries[table] = summary
		} else {
			currentStateDataSummaries[table] = summary
		}
	}

	return databaseSummary{
		historyTableSummaries: historyStateDataSummaries, currentTableSummaries: currentStateDataSummaries,
		dbMetaData: dbMetaData,
	}
}

func getSnapshotIntervalToHistoryTableDeltaSummary(ctx context.Context,
	connConfig sqlstore.ConnectionConfig, expectedHistorySegmentsFromHeights []int64,
	expectedHistorySegmentsToHeights []int64,
) []map[string]tableDataSummary {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		panic(err)
	}

	dbMetaData, err := snapshot.NewDatabaseMetaData(ctx, connConfig)
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

func waitForSnapshotToCompleteUseMeta(sn snapshot.MetaData) {
	currentSnapshotFileName := sn.CurrentStateSnapshotPath
	historySnapshotFileName := sn.HistorySnapshotPath
	snapshotInProgressFileName := filepath.Join(path.Dir(sn.CurrentStateSnapshotPath), snapshot.InProgressFileName(sn.CurrentStateSnapshot.ChainID, sn.CurrentStateSnapshot.Height))

	waitForSnapshotToComplete(currentSnapshotFileName, historySnapshotFileName, snapshotInProgressFileName)
}

func waitForSnapshotToComplete(currentSnapshotFileName string, historySnapshotFileName string, snapshotInProgressFileName string) {
	for {
		time.Sleep(10 * time.Millisecond)
		// wait for snapshot current  file
		_, err := os.Stat(currentSnapshotFileName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				panic(err)
			}
		}

		// wait for snapshot history file
		_, err = os.Stat(historySnapshotFileName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				panic(err)
			}
		}

		// wait for snapshot data dump in progress file to be removed

		_, err = os.Stat(snapshotInProgressFileName)
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

func setupTestSQLMigrations() (int, fs.FS) {
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

	if os.Mkdir(filepath.Join(testMigrationsDir, sqlstore.SQLMigrationsDir), fs.ModePerm); err != nil {
		panic(fmt.Errorf("failed to create migrations dir: %w", err))
	}

	var highestMigrationNumber int
	err = filepath.Walk(sourceMigrationsDir, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() {
			if strings.HasSuffix(info.Name(), ".sql") {
				split := strings.Split(info.Name(), "_")
				if len(split) < 2 {
					return errors.New("expected sql filename of form <version>_<name>.sql")
				}

				migrationNum, err := strconv.Atoi(split[0])
				if err != nil {
					return fmt.Errorf("expected first part of file name to be integer, is %s", split[0])
				}

				if migrationNum > highestMigrationNumber {
					highestMigrationNumber = migrationNum
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

func migrateDatabase(version int64) error {
	poolConfig, err := sqlConfig.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	goose.SetBaseFS(nil)
	err = goose.UpTo(db, testMigrationsDir, version)
	if err != nil {
		return fmt.Errorf("failed to migrate up to version %d:%w", version, err)
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
